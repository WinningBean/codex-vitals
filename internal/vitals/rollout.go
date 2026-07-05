package vitals

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ErrNoRollout = errors.New("no rollout file found")

type SessionInfo struct {
	ID         string
	CWD        string
	GitBranch  string
	GitCommit  string
	Provider   string
	CLIVersion string
}

type ModelInfo struct {
	Model     string
	Effort    string
	Timestamp time.Time
	HasTurn   bool
}

type RateLimit struct {
	UsedPercent   float64
	WindowMinutes int
	ResetsAt      int64
}

type TokenInfo struct {
	Context            ContextUsage
	SessionTotalTokens int64
	Primary            *RateLimit
	Secondary          *RateLimit
	HasTokens          bool
}

type Snapshot struct {
	RolloutPath string
	Session     SessionInfo
	Model       ModelInfo
	Tokens      TokenInfo
	Config      Config
	StartedAt   time.Time
	EndedAt     time.Time
}

type LoadOptions struct {
	CodexHome   string
	RolloutPath string
	ContextMode ContextMode
	// CWD ties the panel to the session started in this directory, so a panel
	// paired with one Codex doesn't show a different, more-recently-written
	// session. Empty means "just take the globally latest rollout".
	CWD string
	// Since (unix seconds) restricts selection to sessions written on or after
	// this time — the panel's launch time — so a brand-new panel ignores older
	// sessions in the same directory. 0 disables the restriction.
	Since int64
}

func LoadSnapshot(options LoadOptions) (Snapshot, error) {
	config, err := LoadConfig(options.CodexHome)
	if err != nil {
		return Snapshot{}, err
	}

	rolloutPath := options.RolloutPath
	if rolloutPath == "" {
		rolloutPath, err = FindLatestRollout(options.CodexHome, options.CWD, options.Since)
		if err != nil {
			return Snapshot{
				Model:  SelectModel(ModelInfo{}, config),
				Config: config,
			}, err
		}
	}

	snapshot, err := ParseRolloutFileWithContextMode(rolloutPath, options.ContextMode)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.RolloutPath = rolloutPath
	snapshot.Config = config
	snapshot.Model = SelectModelForContextMode(snapshot.Model, config, options.ContextMode)
	return snapshot, nil
}

func FindLatestRollout(codexHome, cwd string, since int64) (string, error) {
	sessionsDir := filepath.Join(codexHome, "sessions")
	type rolloutFile struct {
		path  string
		mtime time.Time
	}
	var files []rolloutFile
	err := filepath.WalkDir(sessionsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		files = append(files, rolloutFile{path, info.ModTime()})
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNoRollout
		}
		return "", err
	}
	if len(files) == 0 {
		return "", ErrNoRollout
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mtime.After(files[j].mtime) })

	// Panel mode: a launch time was given, so bind to a session in cwd that has
	// been written since the panel started. This excludes older, unrelated
	// sessions in the same directory; nothing recent means "waiting".
	if since > 0 {
		for _, f := range files {
			if f.mtime.Unix() < since {
				break // files are newest-first
			}
			if cwd == "" || rolloutCWD(f.path) == cwd {
				return f.path, nil
			}
		}
		return "", ErrNoRollout
	}

	// Standalone: prefer the most recent rollout started in cwd, else the
	// globally latest, so `codex-vitals` in any directory still shows something.
	if cwd != "" {
		const scan = 50
		for i := 0; i < len(files) && i < scan; i++ {
			if rolloutCWD(files[i].path) == cwd {
				return files[i].path, nil
			}
		}
	}
	return files[0].path, nil
}

// rolloutCWD returns the session cwd recorded in a rollout's session_meta line.
func rolloutCWD(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	line, err := bufio.NewReader(file).ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	var record struct {
		Payload struct {
			CWD string `json:"cwd"`
		} `json:"payload"`
	}
	if json.Unmarshal([]byte(line), &record) != nil {
		return ""
	}
	return record.Payload.CWD
}

func ParseRolloutFile(path string) (Snapshot, error) {
	return ParseRolloutFileWithContextMode(path, ContextModeCodex)
}

func ParseRolloutFileWithContextMode(path string, mode ContextMode) (Snapshot, error) {
	file, err := os.Open(path)
	if err != nil {
		return Snapshot{}, err
	}
	defer file.Close()

	snapshot, err := ParseRolloutWithContextMode(file, mode)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.RolloutPath = path
	return snapshot, nil
}

func ParseRollout(reader io.Reader) (Snapshot, error) {
	return ParseRolloutWithContextMode(reader, ContextModeCodex)
}

func ParseRolloutWithContextMode(reader io.Reader, mode ContextMode) (Snapshot, error) {
	var snapshot Snapshot
	buffered := bufio.NewReader(reader)
	for {
		line, err := buffered.ReadString('\n')
		if len(strings.TrimSpace(line)) > 0 {
			// Skip unparseable lines instead of failing the whole file: while
			// Codex streams a turn, the last line is often half-written, and
			// aborting would blank the panel until the write completes.
			_ = applyRolloutLine(&snapshot, []byte(line), mode)
		}
		if err != nil {
			// EOF or a read error: return whatever parsed so far (best effort).
			break
		}
	}
	return snapshot, nil
}

func applyRolloutLine(snapshot *Snapshot, line []byte, mode ContextMode) error {
	var record struct {
		Timestamp string          `json:"timestamp"`
		Type      string          `json:"type"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(line, &record); err != nil {
		return err
	}

	eventTime := parseEventTime(record.Timestamp)
	if !eventTime.IsZero() {
		if snapshot.StartedAt.IsZero() || eventTime.Before(snapshot.StartedAt) {
			snapshot.StartedAt = eventTime
		}
		if snapshot.EndedAt.IsZero() || eventTime.After(snapshot.EndedAt) {
			snapshot.EndedAt = eventTime
		}
	}
	switch record.Type {
	case "session_meta":
		return applySessionMeta(snapshot, record.Payload)
	case "turn_context":
		return applyTurnContext(snapshot, record.Payload, eventTime)
	case "event_msg":
		return applyEventMsg(snapshot, record.Payload, mode)
	default:
		return nil
	}
}

func applySessionMeta(snapshot *Snapshot, payload json.RawMessage) error {
	var meta struct {
		ID            string `json:"id"`
		SessionID     string `json:"session_id"`
		CWD           string `json:"cwd"`
		ModelProvider string `json:"model_provider"`
		CLIVersion    string `json:"cli_version"`
		Git           *struct {
			Branch     string `json:"branch"`
			CommitHash string `json:"commit_hash"`
		} `json:"git"`
	}
	if err := json.Unmarshal(payload, &meta); err != nil {
		return err
	}

	id := meta.ID
	if id == "" {
		id = meta.SessionID
	}
	session := SessionInfo{
		ID:         id,
		CWD:        meta.CWD,
		Provider:   meta.ModelProvider,
		CLIVersion: meta.CLIVersion,
	}
	if meta.Git != nil {
		session.GitBranch = meta.Git.Branch
		session.GitCommit = meta.Git.CommitHash
	}
	snapshot.Session = session
	return nil
}

func applyTurnContext(snapshot *Snapshot, payload json.RawMessage, eventTime time.Time) error {
	var turn struct {
		Model             string `json:"model"`
		Effort            string `json:"effort"`
		CWD               string `json:"cwd"`
		CollaborationMode struct {
			Settings struct {
				Model           string `json:"model"`
				ReasoningEffort string `json:"reasoning_effort"`
			} `json:"settings"`
		} `json:"collaboration_mode"`
	}
	if err := json.Unmarshal(payload, &turn); err != nil {
		return err
	}

	model := turn.Model
	if model == "" {
		model = turn.CollaborationMode.Settings.Model
	}
	effort := turn.Effort
	if effort == "" {
		effort = turn.CollaborationMode.Settings.ReasoningEffort
	}

	snapshot.Model = ModelInfo{
		Model:     model,
		Effort:    effort,
		Timestamp: eventTime,
		HasTurn:   true,
	}
	if turn.CWD != "" {
		snapshot.Session.CWD = turn.CWD
	}
	return nil
}

func applyEventMsg(snapshot *Snapshot, payload json.RawMessage, mode ContextMode) error {
	var typed struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &typed); err != nil {
		return err
	}
	if typed.Type != "token_count" {
		return nil
	}
	return applyTokenCount(snapshot, payload, mode)
}

func applyTokenCount(snapshot *Snapshot, payload json.RawMessage, mode ContextMode) error {
	var tokenCount struct {
		Info struct {
			LastTokenUsage     TokenUsage `json:"last_token_usage"`
			TotalTokenUsage    TokenUsage `json:"total_token_usage"`
			ModelContextWindow int64      `json:"model_context_window"`
		} `json:"info"`
		RateLimits struct {
			Primary   rateLimitPayload `json:"primary"`
			Secondary rateLimitPayload `json:"secondary"`
		} `json:"rate_limits"`
	}
	if err := json.Unmarshal(payload, &tokenCount); err != nil {
		return err
	}

	primary := tokenCount.RateLimits.Primary.rateLimit()
	secondary := tokenCount.RateLimits.Secondary.rateLimit()
	// Rate limits appear only on some token_count events; the latest event
	// frequently omits them. Carry forward the last non-nil value so the 5H/7D
	// lines don't vanish once they have appeared in the session.
	if primary == nil {
		primary = snapshot.Tokens.Primary
	}
	if secondary == nil {
		secondary = snapshot.Tokens.Secondary
	}
	totalUsage := tokenCount.Info.TotalTokenUsage
	sessionTotalTokens := totalUsage.InputTokens + totalUsage.OutputTokens + totalUsage.ReasoningOutputTokens
	if sessionTotalTokens == 0 {
		sessionTotalTokens = totalUsage.TotalTokens
	}
	snapshot.Tokens = TokenInfo{
		Context: CalculateContextUsageForMode(
			tokenCount.Info.LastTokenUsage,
			tokenCount.Info.ModelContextWindow,
			mode,
		),
		SessionTotalTokens: sessionTotalTokens,
		Primary:            primary,
		Secondary:          secondary,
		HasTokens:          true,
	}
	return nil
}

type rateLimitPayload struct {
	UsedPercent   float64 `json:"used_percent"`
	WindowMinutes int     `json:"window_minutes"`
	ResetsAt      int64   `json:"resets_at"`
}

func (r rateLimitPayload) rateLimit() *RateLimit {
	if r.WindowMinutes == 0 && r.UsedPercent == 0 && r.ResetsAt == 0 {
		return nil
	}
	return &RateLimit{
		UsedPercent:   r.UsedPercent,
		WindowMinutes: r.WindowMinutes,
		ResetsAt:      r.ResetsAt,
	}
}

func parseEventTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(seconds, 0)
	}
	if seconds, err := strconv.ParseFloat(value, 64); err == nil {
		whole, fraction := math.Modf(seconds)
		return time.Unix(int64(whole), int64(fraction*1_000_000_000))
	}
	return time.Time{}
}
