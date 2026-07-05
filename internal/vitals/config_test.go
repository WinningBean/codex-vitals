package vitals

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigReadsTopLevelModelSettings(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	data := []byte(`
model = "gpt-5.5" # current model
model_reasoning_effort = 'xhigh'

[tui]
status_line = ["context-used"]
`)
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := Config{
		Model:  "gpt-5.5",
		Effort: "xhigh",
		MTime:  got.MTime,
	}
	if got != want {
		t.Fatalf("LoadConfig() = %+v, want %+v", got, want)
	}
}

func TestSelectModelPrefersNewerConfig(t *testing.T) {
	turnTime := time.Date(2026, time.July, 3, 9, 0, 0, 0, time.UTC)
	turn := ModelInfo{
		Model:     "gpt-5",
		Effort:    "high",
		Timestamp: turnTime,
		HasTurn:   true,
	}
	config := Config{
		Model:  "gpt-5.5",
		Effort: "xhigh",
		MTime:  turnTime.Add(time.Second),
	}

	got := SelectModel(turn, config)
	want := ModelInfo{
		Model:     "gpt-5.5",
		Effort:    "xhigh",
		Timestamp: turnTime,
		HasTurn:   true,
	}
	if got != want {
		t.Fatalf("SelectModel() = %+v, want %+v", got, want)
	}
}

func TestSelectModelKeepsNewerTurnContext(t *testing.T) {
	turnTime := time.Date(2026, time.July, 3, 9, 0, 0, 0, time.UTC)
	turn := ModelInfo{
		Model:     "gpt-5",
		Effort:    "high",
		Timestamp: turnTime,
		HasTurn:   true,
	}
	config := Config{
		Model:  "gpt-5.5",
		Effort: "xhigh",
		MTime:  turnTime.Add(-time.Second),
	}

	got := SelectModel(turn, config)
	if got != turn {
		t.Fatalf("SelectModel() = %+v, want %+v", got, turn)
	}
}

func TestSelectModelForContextModeReflectsNewerConfig(t *testing.T) {
	turnTime := time.Date(2026, time.July, 3, 9, 0, 0, 0, time.UTC)
	turn := ModelInfo{Model: "gpt-5.5", Effort: "xhigh", Timestamp: turnTime, HasTurn: true}
	// config.toml changed (via /model) after the last turn: effort -> medium.
	config := Config{Model: "gpt-5.5", Effort: "medium", MTime: turnTime.Add(time.Hour)}

	got := SelectModelForContextMode(turn, config, ContextModePatched)
	if got.Effort != "medium" {
		t.Fatalf("effort = %q, want medium (config is newer than the last turn)", got.Effort)
	}

	// When config predates the turn, the session's own model/effort stays.
	stale := Config{Model: "gpt-5.5", Effort: "medium", MTime: turnTime.Add(-time.Hour)}
	if got := SelectModelForContextMode(turn, stale, ContextModePatched); got != turn {
		t.Fatalf("stale config = %+v, want %+v", got, turn)
	}
}

func TestDefaultCodexHome(t *testing.T) {
	t.Setenv("CODEX_HOME", "/custom/codex")
	if got := DefaultCodexHome("/home/u"); got != "/custom/codex" {
		t.Errorf("DefaultCodexHome(env) = %q, want /custom/codex", got)
	}
	t.Setenv("CODEX_HOME", "")
	if got := DefaultCodexHome("/home/u"); got != filepath.Join("/home/u", ".codex") {
		t.Errorf("DefaultCodexHome(default) = %q", got)
	}
}
