package vitals

import "testing"

func TestRenderLine(t *testing.T) {
	snapshot := Snapshot{
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
			Primary: &RateLimit{
				UsedPercent:   1,
				WindowMinutes: 300,
			},
			Secondary: &RateLimit{
				UsedPercent:   26,
				WindowMinutes: 10080,
			},
			HasTokens: true,
		},
	}

	got := RenderLine(snapshot, "/Users/wsb")
	want := "🤖 gpt-5.5 xhigh · 🌿 main · ~/my-project · Ctx ██░░░░ 25% (61k/246k) · 5h 99% · wk 74%"
	if got != want {
		t.Fatalf("RenderLine() = %q, want %q", got, want)
	}
}

func TestRenderCurrentHUDLine(t *testing.T) {
	snapshot := Snapshot{
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
				UsedTokens:  236250,
				TotalTokens: 258400,
				Percent:     91,
			},
			Primary: &RateLimit{
				UsedPercent:   40,
				WindowMinutes: 300,
			},
			Secondary: &RateLimit{
				UsedPercent:   46,
				WindowMinutes: 10080,
			},
			HasTokens: true,
		},
	}

	got := RenderLineWithOptions(snapshot, "/Users/wsb", RenderOptions{
		Style: RenderStyleCurrentHUD,
		Color: false,
	})
	want := "gpt-5.5 xhigh · ~/my-project · Context 91% used · 5h 60% left · weekly 54% left"
	if got != want {
		t.Fatalf("RenderLineWithOptions() = %q, want %q", got, want)
	}
}

func TestRenderCurrentHUDLineWithColors(t *testing.T) {
	snapshot := Snapshot{
		Model: ModelInfo{
			Model:  "gpt-5.5",
			Effort: "xhigh",
		},
	}

	got := RenderLineWithOptions(snapshot, "/Users/wsb", RenderOptions{
		Style: RenderStyleCurrentHUD,
		Color: true,
	})
	want := "\x1b[38;2;148;226;213mgpt-5.5 xhigh\x1b[0m"
	if got != want {
		t.Fatalf("RenderLineWithOptions() = %q, want %q", got, want)
	}
}

func TestRenderAnswerFooter(t *testing.T) {
	snapshot := Snapshot{
		Session: SessionInfo{
			CWD:       "/Users/wsb/my-project",
			GitBranch: "main",
		},
		Model: ModelInfo{
			Model:  "gpt-5.5",
			Effort: "xhigh",
		},
		Tokens: TokenInfo{
			Context: ContextUsage{
				UsedTokens:  236250,
				TotalTokens: 258400,
				Percent:     91,
			},
			Primary: &RateLimit{
				UsedPercent: 40,
			},
			Secondary: &RateLimit{
				UsedPercent: 46,
			},
			HasTokens: true,
		},
	}

	got := RenderLineWithOptions(snapshot, "/Users/wsb", RenderOptions{
		Style: RenderStyleAnswerFooter,
		Color: false,
	})
	want := "🤖 gpt-5.5 xhigh\n🌿 main\n📁 ~/my-project\n🧠 Context █████████░  91% used (236k/258k)\n⏱ 5h      ██████░░░░  60% left\n📅 weekly  █████░░░░░  54% left"
	if got != want {
		t.Fatalf("RenderLineWithOptions() = %q, want %q", got, want)
	}
}
