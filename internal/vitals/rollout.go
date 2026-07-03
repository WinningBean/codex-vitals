package vitals

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
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
}

func LoadSnapshot(options LoadOptions) (Snapshot, error) {
	config, err := LoadConfig(options.CodexHome)
	if err != nil {
		return Snapshot{}, err
	}

	rolloutPath := options.RolloutPath
	if rolloutPath == "" {
		rolloutPath, err = FindLatestRollout(options.CodexHome)
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

func FindLatestRollout(codexHome string) (string, error) {
	sessionsDir := filepath.Join(codexHome, "sessions")
	var latestPath string
	var latestTime time.Time

	err := filepath.WalkDir(sessionsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
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
		if latestPath == "" || info.ModTime().After(latestTime) {
			latestPath = path
			latestTime = info.ModTime()
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNoRollout
		}
		return "", err
	}
	if latestPath == "" {
		return "", ErrNoRollout
	}
	return latestPath, nil
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
			if parseErr := applyRolloutLine(&snapshot, []byte(line), mode); parseErr != nil {
				return Snapshot{}, parseErr
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return Snapshot{}, err
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
