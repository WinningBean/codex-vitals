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
