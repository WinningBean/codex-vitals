package vitals

import "testing"

func TestCalculateContextUsage(t *testing.T) {
	tests := []struct {
		name          string
		totalTokens   int64
		contextWindow int64
		want          ContextUsage
	}{
		{
			name:          "worked example matches Codex status line",
			totalTokens:   73305,
			contextWindow: 258400,
			want: ContextUsage{
				UsedTokens:  61305,
				TotalTokens: 246400,
				Percent:     25,
			},
		},
		{
			name:          "below baseline is zero",
			totalTokens:   5000,
			contextWindow: 258400,
			want: ContextUsage{
				UsedTokens:  0,
				TotalTokens: 246400,
				Percent:     0,
			},
		},
		{
			name:          "full window clamps to one hundred",
			totalTokens:   258400,
			contextWindow: 258400,
			want: ContextUsage{
				UsedTokens:  246400,
				TotalTokens: 246400,
				Percent:     100,
			},
		},
		{
			name:          "degenerate window is zero",
			totalTokens:   12000,
			contextWindow: 12000,
			want:          ContextUsage{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateContextUsage(tt.totalTokens, tt.contextWindow)
			if got != tt.want {
				t.Fatalf("CalculateContextUsage() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCalculateCurrentHUDContextUsage(t *testing.T) {
	got := CalculateCurrentHUDContextUsage(
		/*inputTokens*/ 119642,
		/*cachedInputTokens*/ 116608,
		/*contextWindow*/ 258400,
	)
	want := ContextUsage{
		UsedTokens:  236250,
		TotalTokens: 258400,
		Percent:     91,
	}
	if got != want {
		t.Fatalf("CalculateCurrentHUDContextUsage() = %+v, want %+v", got, want)
	}
}

func TestParseContextMode(t *testing.T) {
	got, err := ParseContextMode("current-hud")
	if err != nil {
		t.Fatal(err)
	}
	if got != ContextModeCurrentHUD {
		t.Fatalf("ParseContextMode() = %q, want %q", got, ContextModeCurrentHUD)
	}

	if _, err := ParseContextMode("bad"); err != ErrInvalidContextMode {
		t.Fatalf("ParseContextMode() error = %v, want %v", err, ErrInvalidContextMode)
	}
}
