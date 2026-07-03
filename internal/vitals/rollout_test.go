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

func TestParseRolloutCurrentHUDContextMode(t *testing.T) {
	input := strings.NewReader(`
{"timestamp":"2026-07-03T09:00:00Z","type":"session_meta","payload":{"id":"session-1","cwd":"/Users/wsb/project","model_provider":"openai","cli_version":"1.2.3","git":{"branch":"main","commit_hash":"abc123"}}}
{"timestamp":"2026-07-03T09:02:00Z","type":"turn_context","payload":{"model":"gpt-5.5","effort":"xhigh","cwd":"/Users/wsb/project"}}
{"timestamp":"2026-07-03T09:03:00Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":73305,"input_tokens":70000,"cached_input_tokens":1000,"output_tokens":3305},"total_token_usage":{"total_tokens":73305,"input_tokens":70000,"output_tokens":3305},"model_context_window":258400},"rate_limits":{"primary":{"used_percent":1,"window_minutes":300,"resets_at":1783072800},"secondary":{"used_percent":26,"window_minutes":10080,"resets_at":1783677600}}}}
`)

	got, err := ParseRolloutWithContextMode(input, ContextModeCurrentHUD)
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

	got, err := FindLatestRollout(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != newPath {
		t.Fatalf("FindLatestRollout() = %q, want %q", got, newPath)
	}
}
