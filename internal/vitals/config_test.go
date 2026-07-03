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

func TestSelectModelForContextModeCurrentHUDKeepsTurnContext(t *testing.T) {
	turnTime := time.Date(2026, time.July, 3, 9, 0, 0, 0, time.UTC)
	turn := ModelInfo{
		Model:     "gpt-5.5",
		Effort:    "xhigh",
		Timestamp: turnTime,
		HasTurn:   true,
	}
	config := Config{
		Model:  "gpt-5.5",
		Effort: "medium",
		MTime:  turnTime.Add(time.Hour),
	}

	got := SelectModelForContextMode(turn, config, ContextModeCurrentHUD)
	if got != turn {
		t.Fatalf("SelectModelForContextMode() = %+v, want %+v", got, turn)
	}
}
