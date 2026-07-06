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
			name:          "large window mid usage",
			totalTokens:   106000,
			contextWindow: 200000,
			want: ContextUsage{
				UsedTokens:  94000,
				TotalTokens: 188000,
				Percent:     50,
			},
		},
		{
			// window <= baseline: raw bounded ratio, not a bogus 0% (0/0).
			name:          "small window half used",
			totalTokens:   4000,
			contextWindow: 8000,
			want: ContextUsage{
				UsedTokens:  4000,
				TotalTokens: 8000,
				Percent:     50,
			},
		},
		{
			name:          "small window empty",
			totalTokens:   0,
			contextWindow: 8000,
			want: ContextUsage{
				UsedTokens:  0,
				TotalTokens: 8000,
				Percent:     0,
			},
		},
		{
			name:          "window equals baseline uses raw ratio",
			totalTokens:   12000,
			contextWindow: 12000,
			want: ContextUsage{
				UsedTokens:  12000,
				TotalTokens: 12000,
				Percent:     100,
			},
		},
		{
			name:          "window just above baseline empty",
			totalTokens:   12000,
			contextWindow: 12001,
			want: ContextUsage{
				UsedTokens:  0,
				TotalTokens: 1,
				Percent:     0,
			},
		},
		{
			name:          "window just above baseline full",
			totalTokens:   12001,
			contextWindow: 12001,
			want: ContextUsage{
				UsedTokens:  1,
				TotalTokens: 1,
				Percent:     100,
			},
		},
		{
			name:          "zero window is empty",
			totalTokens:   1000,
			contextWindow: 0,
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

func TestCalculatePatchedContextUsage(t *testing.T) {
	// cached_input_tokens is a subset of input_tokens (the cached portion of the
	// same prompt), so input_tokens alone is the full context occupancy; adding
	// cached again would double-count it.
	got := CalculatePatchedContextUsage(
		/*inputTokens*/ 119642,
		/*cachedInputTokens*/ 116608,
		/*contextWindow*/ 258400,
	)
	want := ContextUsage{
		UsedTokens:  119642,
		TotalTokens: 258400,
		Percent:     46,
	}
	if got != want {
		t.Fatalf("CalculatePatchedContextUsage() = %+v, want %+v", got, want)
	}
}

// A resumed (or already-running) session reports a large input_tokens whose
// bulk is cached. Double-counting the cache saturated the gauge to 100% at
// roughly half the real fill — the "context shows 100% at startup" bug.
func TestCalculatePatchedContextUsageDoesNotDoubleCountCache(t *testing.T) {
	got := CalculatePatchedContextUsage(
		/*inputTokens*/ 130000,
		/*cachedInputTokens*/ 128000,
		/*contextWindow*/ 258400,
	)
	want := ContextUsage{
		UsedTokens:  130000,
		TotalTokens: 258400,
		Percent:     50,
	}
	if got != want {
		t.Fatalf("CalculatePatchedContextUsage() = %+v, want %+v", got, want)
	}
}

func TestParseContextMode(t *testing.T) {
	// "patched" is canonical; "current-hud" stays as a back-compat alias.
	for _, in := range []string{"patched", "current-hud"} {
		got, err := ParseContextMode(in)
		if err != nil {
			t.Fatalf("ParseContextMode(%q): %v", in, err)
		}
		if got != ContextModePatched {
			t.Fatalf("ParseContextMode(%q) = %q, want %q", in, got, ContextModePatched)
		}
	}

	if _, err := ParseContextMode("bad"); err != ErrInvalidContextMode {
		t.Fatalf("ParseContextMode() error = %v, want %v", err, ErrInvalidContextMode)
	}
}
