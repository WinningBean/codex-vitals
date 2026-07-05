package vitals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleSnapshot() Snapshot {
	return Snapshot{
		Session: SessionInfo{
			CWD:       "/Users/wsb/my-project",
			GitBranch: "main",
		},
		Model: ModelInfo{
			Model:   "gpt-5.5",
			Effort:  "xhigh",
			HasTurn: true,
		},
		Tokens: TokenInfo{
			Context: ContextUsage{
				UsedTokens:  61305,
				TotalTokens: 246400,
				Percent:     25,
			},
			Primary:   &RateLimit{UsedPercent: 1, WindowMinutes: 300},
			Secondary: &RateLimit{UsedPercent: 26, WindowMinutes: 10080},
			HasTokens: true,
		},
	}
}

// noEnv forces renderEnv to report "no env" so tests are deterministic
// regardless of the caller's shell environment.
func noEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CONDA_DEFAULT_ENV", "")
	t.Setenv("VIRTUAL_ENV", "")
}

// Default size (no options) is medium.
func TestRenderLineDefaultsToMedium(t *testing.T) {
	noEnv(t)
	got := RenderLine(sampleSnapshot(), "/Users/wsb")
	want := "🤖 gpt-5.5 ⚡xhigh │ no git │ no env\n" +
		"📂 ~/my-project 🌿(main)\n" +
		"🧠 Context  ████████░░░░░░░░░░░░░░░░░░░░░░░░ 25% used\n" +
		"🚀 Usage 5H ░░░░░░░░░░ 1% │ 📅 7D ███░░░░░░░ 26%"
	if got != want {
		t.Fatalf("RenderLine() =\n%q\nwant\n%q", got, want)
	}
}

func TestParseSize(t *testing.T) {
	cases := map[string]Size{
		"":       SizeM,
		"m":      SizeM,
		"medium": SizeM,
		"xs":     SizeXS,
		"xsmall": SizeXS,
		"S":      SizeS,
		"small":  SizeS,
		"l":      SizeL,
		"large":  SizeL,
		"xl":     SizeXL,
		"xlarge": SizeXL,
		" M ":    SizeM,
	}
	for input, want := range cases {
		got, err := ParseSize(input)
		if err != nil {
			t.Fatalf("ParseSize(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("ParseSize(%q) = %q, want %q", input, got, want)
		}
	}
	if _, err := ParseSize("huge"); err == nil {
		t.Fatalf("ParseSize(\"huge\") expected error, got nil")
	}
}

func TestRenderSizeLineCounts(t *testing.T) {
	noEnv(t)
	snapshot := sampleSnapshot()
	cases := []struct {
		size  Size
		lines int
	}{
		{SizeXS, 2},
		{SizeS, 2},
		{SizeM, 4},
		{SizeL, 5},
		{SizeXL, 5},
	}
	for _, c := range cases {
		got := RenderLineWithOptions(snapshot, "/Users/wsb", RenderOptions{Size: c.size})
		if n := strings.Count(got, "\n") + 1; n != c.lines {
			t.Fatalf("size %q rendered %d lines, want %d:\n%s", c.size, n, c.lines, got)
		}
	}
}

// xs shows tiny bars without labels or percentages.
func TestRenderSizeXS(t *testing.T) {
	noEnv(t)
	got := RenderLineWithOptions(sampleSnapshot(), "/Users/wsb", RenderOptions{Size: SizeXS})
	want := "🤖 gpt-5.5 ⚡xhigh 📂 ~/my-project 🌿(main)\n" +
		"🧠 ███░░░░░░░  🚀 5H ░░░░░░░░░░  📅 7D ███░░░░░░░"
	if got != want {
		t.Fatalf("xs =\n%q\nwant\n%q", got, want)
	}
}

// l shows reset times, token counts, and 20-wide bars.
func TestRenderSizeLShowsResetsAndTokens(t *testing.T) {
	noEnv(t)
	got := RenderLineWithOptions(sampleSnapshot(), "/Users/wsb", RenderOptions{Size: SizeL})
	for _, want := range []string{"🧾 0 tokens", "(61k/246k)", "Usage 5H", "Usage 7D"} {
		if !strings.Contains(got, want) {
			t.Fatalf("l output missing %q:\n%s", want, got)
		}
	}
	// 20-wide context bar: 25% -> 5 filled blocks.
	if !strings.Contains(got, "🧠 Context  "+strings.Repeat("█", 5)+strings.Repeat("░", 15)) {
		t.Fatalf("l context bar not 20-wide:\n%s", got)
	}
}

// xl uses 40-wide bars.
func TestRenderSizeXLUsesWideBars(t *testing.T) {
	noEnv(t)
	got := RenderLineWithOptions(sampleSnapshot(), "/Users/wsb", RenderOptions{Size: SizeXL})
	// 40-wide context bar: 25% -> 10 filled blocks.
	if !strings.Contains(got, "🧠 Context  "+strings.Repeat("█", 10)+strings.Repeat("░", 30)) {
		t.Fatalf("xl context bar not 40-wide:\n%s", got)
	}
}

// Color output wraps the model label in the teal truecolor escape.
func TestRenderColorModel(t *testing.T) {
	got := RenderLineWithOptions(sampleSnapshot(), "/Users/wsb", RenderOptions{Size: SizeXS, Color: true})
	if !strings.HasPrefix(got, "🤖 \x1b[38;2;148;226;213mgpt-5.5 ⚡xhigh\x1b[0m") {
		t.Fatalf("expected teal model prefix, got:\n%q", got)
	}
}

// Context / 5H / 7D bars must start at the same column in l and xl.
func TestUsageBarsAligned(t *testing.T) {
	noEnv(t)
	for _, size := range []Size{SizeL, SizeXL} {
		got := RenderLineWithOptions(sampleSnapshot(), "/Users/wsb", RenderOptions{Size: size})
		var cols []int
		for _, line := range strings.Split(got, "\n") {
			if strings.HasPrefix(line, "🧠") || strings.HasPrefix(line, "🚀") || strings.HasPrefix(line, "📅") {
				i := strings.IndexAny(line, "█░")
				cols = append(cols, len([]rune(line[:i])))
			}
		}
		if len(cols) != 3 {
			t.Fatalf("size %s: expected 3 usage bars, got %d", size, len(cols))
		}
		for _, c := range cols {
			if c != cols[0] {
				t.Fatalf("size %s bars not aligned: cols=%v\n%s", size, cols, got)
			}
		}
	}
}

// ConfiguredSize reads a valid size from the config file and ignores junk.
func TestConfiguredSize(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if _, ok := ConfiguredSize(); ok {
		t.Fatalf("expected no configured size before the file exists")
	}

	cfg := SizeConfigPath()
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("  xl\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, ok := ConfiguredSize(); !ok || got != SizeXL {
		t.Fatalf("ConfiguredSize() = %q, %v; want xl, true", got, ok)
	}

	if err := os.WriteFile(cfg, []byte("nonsense"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ConfiguredSize(); ok {
		t.Fatalf("expected invalid size file to be ignored")
	}
}

// A snapshot with no data renders the waiting message.
func TestRenderWaiting(t *testing.T) {
	got := RenderLineWithOptions(Snapshot{}, "/Users/wsb", RenderOptions{Size: SizeM})
	if got != "codex-vitals: waiting for Codex session data" {
		t.Fatalf("empty snapshot = %q", got)
	}
}
