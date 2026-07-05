package vitals

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseRolloutUsesLatestRelevantEvents(t *testing.T) {
	input := strings.NewReader(`
{"timestamp":"2026-07-03T09:00:00Z","type":"session_meta","payload":{"id":"session-1","cwd":"/Users/wsb/project","model_provider":"openai","cli_version":"1.2.3","git":{"branch":"main","commit_hash":"abc123"}}}
{"timestamp":"2026-07-03T09:01:00Z","type":"turn_context","payload":{"model":"gpt-5","effort":"high","cwd":"/Users/wsb/project"}}
{"timestamp":"2026-07-03T09:02:00Z","type":"turn_context","payload":{"model":"gpt-5.5","effort":"xhigh","cwd":"/Users/wsb/project"}}
{"timestamp":"2026-07-03T09:03:00Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":73305,"input_tokens":70000,"cached_input_tokens":1000,"output_tokens":3305},"total_token_usage":{"total_tokens":73305,"input_tokens":70000,"output_tokens":3305},"model_context_window":258400},"rate_limits":{"primary":{"used_percent":1,"window_minutes":300,"resets_at":1783072800},"secondary":{"used_percent":26,"window_minutes":10080,"resets_at":1783677600}}}}
`)

	got, err := ParseRollout(input)
	if err != nil {
		t.Fatal(err)
	}

	want := Snapshot{
		Session: SessionInfo{
			ID:         "session-1",
			CWD:        "/Users/wsb/project",
			GitBranch:  "main",
			GitCommit:  "abc123",
			Provider:   "openai",
			CLIVersion: "1.2.3",
		},
		Model: ModelInfo{
			Model:     "gpt-5.5",
			Effort:    "xhigh",
			Timestamp: time.Date(2026, time.July, 3, 9, 2, 0, 0, time.UTC),
			HasTurn:   true,
		},
		Tokens: TokenInfo{
			Context: ContextUsage{
				UsedTokens:  61305,
				TotalTokens: 246400,
				Percent:     25,
			},
			SessionTotalTokens: 73305,
			Primary: &RateLimit{
				UsedPercent:   1,
				WindowMinutes: 300,
				ResetsAt:      1783072800,
			},
			Secondary: &RateLimit{
				UsedPercent:   26,
				WindowMinutes: 10080,
				ResetsAt:      1783677600,
			},
			HasTokens: true,
		},
		StartedAt: time.Date(2026, time.July, 3, 9, 0, 0, 0, time.UTC),
		EndedAt:   time.Date(2026, time.July, 3, 9, 3, 0, 0, time.UTC),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseRollout() = %+v, want %+v", got, want)
	}
}

func TestParseRolloutPatchedContextMode(t *testing.T) {
	input := strings.NewReader(`
{"timestamp":"2026-07-03T09:00:00Z","type":"session_meta","payload":{"id":"session-1","cwd":"/Users/wsb/project","model_provider":"openai","cli_version":"1.2.3","git":{"branch":"main","commit_hash":"abc123"}}}
{"timestamp":"2026-07-03T09:02:00Z","type":"turn_context","payload":{"model":"gpt-5.5","effort":"xhigh","cwd":"/Users/wsb/project"}}
{"timestamp":"2026-07-03T09:03:00Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":73305,"input_tokens":70000,"cached_input_tokens":1000,"output_tokens":3305},"total_token_usage":{"total_tokens":73305,"input_tokens":70000,"output_tokens":3305},"model_context_window":258400},"rate_limits":{"primary":{"used_percent":1,"window_minutes":300,"resets_at":1783072800},"secondary":{"used_percent":26,"window_minutes":10080,"resets_at":1783677600}}}}
`)

	got, err := ParseRolloutWithContextMode(input, ContextModePatched)
	if err != nil {
		t.Fatal(err)
	}

	want := ContextUsage{
		UsedTokens:  71000,
		TotalTokens: 258400,
		Percent:     27,
	}
	if got.Tokens.Context != want {
		t.Fatalf("context = %+v, want %+v", got.Tokens.Context, want)
	}
}

func TestFindLatestRollout(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "sessions", "2026", "07", "02", "rollout-old.jsonl")
	newPath := filepath.Join(root, "sessions", "2026", "07", "03", "rollout-new.jsonl")
	for _, path := range []string{oldPath, newPath} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	oldTime := time.Date(2026, time.July, 2, 9, 0, 0, 0, time.UTC)
	newTime := oldTime.Add(time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	got, err := FindLatestRollout(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != newPath {
		t.Fatalf("FindLatestRollout() = %q, want %q", got, newPath)
	}
}

func TestParseRolloutCarriesForwardRateLimits(t *testing.T) {
	// The first token_count carries rate limits; the latest one omits them
	// (primary/secondary null). The 5H/7D values must persist.
	input := strings.NewReader(`
{"timestamp":"2026-07-03T09:00:00Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":50000,"input_tokens":48000,"cached_input_tokens":1000,"output_tokens":2000},"total_token_usage":{"total_tokens":50000},"model_context_window":258400},"rate_limits":{"primary":{"used_percent":1,"window_minutes":300,"resets_at":1783072800},"secondary":{"used_percent":26,"window_minutes":10080,"resets_at":1783677600}}}}
{"timestamp":"2026-07-03T09:05:00Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":79794,"input_tokens":78000,"cached_input_tokens":1500,"output_tokens":1794},"total_token_usage":{"total_tokens":79794},"model_context_window":258400},"rate_limits":{"primary":null,"secondary":null}}}
`)

	got, err := ParseRollout(input)
	if err != nil {
		t.Fatal(err)
	}

	if got.Tokens.Primary == nil {
		t.Fatalf("primary rate limit should carry forward, got nil")
	}
	if got.Tokens.Primary.UsedPercent != 1 || got.Tokens.Primary.WindowMinutes != 300 {
		t.Errorf("primary not carried forward: %+v", got.Tokens.Primary)
	}
	if got.Tokens.Secondary == nil {
		t.Fatalf("secondary rate limit should carry forward, got nil")
	}
	if got.Tokens.Secondary.UsedPercent != 26 || got.Tokens.Secondary.WindowMinutes != 10080 {
		t.Errorf("secondary not carried forward: %+v", got.Tokens.Secondary)
	}
	// Context still reflects the LATEST token_count.
	if got.Tokens.Context.Percent != 28 {
		t.Errorf("context percent = %d, want 28 (latest event)", got.Tokens.Context.Percent)
	}
}

const sampleRollout = `{"timestamp":"2026-07-05T10:00:00Z","type":"session_meta","payload":{"id":"s-file","cwd":"/tmp/proj","model_provider":"openai","git":{"branch":"main"}}}
{"timestamp":"2026-07-05T10:01:00Z","type":"turn_context","payload":{"model":"gpt-5.5","effort":"high"}}
{"timestamp":"2026-07-05T10:02:00Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":50000},"total_token_usage":{"total_tokens":50000},"model_context_window":258400}}}
`

func writeRollout(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(sampleRollout), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseRolloutFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	writeRollout(t, path)

	got, err := ParseRolloutFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Session.ID != "s-file" || got.Model.Model != "gpt-5.5" {
		t.Fatalf("ParseRolloutFile got %+v", got.Session)
	}

	if _, err := ParseRolloutFileWithContextMode(path, ContextModePatched); err != nil {
		t.Fatalf("ParseRolloutFileWithContextMode: %v", err)
	}
	if _, err := ParseRolloutFile(filepath.Join(t.TempDir(), "missing.jsonl")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadSnapshot(t *testing.T) {
	// RolloutPath pins a specific file.
	path := filepath.Join(t.TempDir(), "pinned.jsonl")
	writeRollout(t, path)
	got, err := LoadSnapshot(LoadOptions{RolloutPath: path})
	if err != nil || got.Session.ID != "s-file" {
		t.Fatalf("LoadSnapshot(RolloutPath) = %+v, err %v", got.Session, err)
	}

	// CodexHome scans sessions/**/rollout-*.jsonl for the latest.
	home := t.TempDir()
	writeRollout(t, filepath.Join(home, "sessions", "2026", "07", "05", "rollout-abc.jsonl"))
	got, err = LoadSnapshot(LoadOptions{CodexHome: home})
	if err != nil || got.Session.ID != "s-file" {
		t.Fatalf("LoadSnapshot(CodexHome) = %+v, err %v", got.Session, err)
	}

	// No rollout anywhere -> ErrNoRollout.
	if _, err := LoadSnapshot(LoadOptions{CodexHome: t.TempDir()}); err != ErrNoRollout {
		t.Fatalf("LoadSnapshot(empty) err = %v, want ErrNoRollout", err)
	}
}

// A half-written last line (Codex streaming a turn) must not blank the parse.
func TestParseRolloutSkipsBadLines(t *testing.T) {
	input := strings.NewReader(
		`{"timestamp":"2026-07-05T10:00:00Z","type":"session_meta","payload":{"id":"x","cwd":"/tmp/p"}}` + "\n" +
			`{"timestamp":"2026-07-05T10:02:00Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":80000},"total_token_usage":{"total_tokens":80000},"model_context_window":258400}}}` + "\n" +
			`{"timestamp":"2026-07-05T10:03:00Z","type":"turn_`)
	got, err := ParseRollout(input)
	if err != nil {
		t.Fatalf("ParseRollout returned error on partial line: %v", err)
	}
	if !got.Tokens.HasTokens {
		t.Fatalf("token data lost when the last line was partial")
	}
}

// The panel must follow its own session (cwd), not whichever rollout is newest.
func TestFindLatestRolloutPrefersCWD(t *testing.T) {
	home := t.TempDir()
	write := func(name, cwd string, mtime time.Time) string {
		p := filepath.Join(home, "sessions", "2026", "07", "05", name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(`{"type":"session_meta","payload":{"cwd":"`+cwd+`"}}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mtime, mtime); err != nil {
			t.Fatal(err)
		}
		return p
	}
	mine := write("rollout-a.jsonl", "/work/mine", time.Now().Add(-time.Hour))
	write("rollout-b.jsonl", "/work/other", time.Now()) // newer, different session

	got, err := FindLatestRollout(home, "/work/mine")
	if err != nil || got != mine {
		t.Fatalf("cwd match = %q (err %v), want %q", got, err, mine)
	}
	// No cwd falls back to the globally newest.
	if got, _ := FindLatestRollout(home, ""); filepath.Base(got) != "rollout-b.jsonl" {
		t.Fatalf("global latest = %q, want rollout-b.jsonl", got)
	}
}
