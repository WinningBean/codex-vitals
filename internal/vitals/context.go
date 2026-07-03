package vitals

import "math"

const baselineTokens int64 = 12000

// ContextUsage is the effective context-window usage Codex displays after
// subtracting the reserved baseline from both used tokens and the window.
type ContextUsage struct {
	UsedTokens  int64
	TotalTokens int64
	Percent     int
}

func CalculateContextUsage(totalTokens int64, contextWindow int64) ContextUsage {
	effectiveWindow := contextWindow - baselineTokens
	if effectiveWindow <= 0 {
		return ContextUsage{}
	}

	used := totalTokens - baselineTokens
	if used < 0 {
		used = 0
	}
	if used > effectiveWindow {
		used = effectiveWindow
	}

	percent := int(math.Round(float64(used) / float64(effectiveWindow) * 100))
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	return ContextUsage{
		UsedTokens:  used,
		TotalTokens: effectiveWindow,
		Percent:     percent,
	}
}
