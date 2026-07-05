package vitals

import "math"

const baselineTokens int64 = 12000

type ContextMode string

const (
	ContextModeCodex   ContextMode = "codex"
	ContextModePatched ContextMode = "patched"
)

// ContextUsage is the effective context-window usage Codex displays after
// subtracting the reserved baseline from both used tokens and the window.
type ContextUsage struct {
	UsedTokens  int64
	TotalTokens int64
	Percent     int
}

type TokenUsage struct {
	TotalTokens           int64 `json:"total_tokens"`
	InputTokens           int64 `json:"input_tokens"`
	CachedInputTokens     int64 `json:"cached_input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
}

func ParseContextMode(value string) (ContextMode, error) {
	switch ContextMode(value) {
	case "", ContextModeCodex:
		return ContextModeCodex, nil
	case ContextModePatched, "current-hud": // current-hud kept as a back-compat alias
		return ContextModePatched, nil
	default:
		return "", ErrInvalidContextMode
	}
}

var ErrInvalidContextMode = errInvalidContextMode{}

type errInvalidContextMode struct{}

func (errInvalidContextMode) Error() string {
	return "context mode must be one of: codex, patched"
}

func CalculateContextUsageForMode(usage TokenUsage, contextWindow int64, mode ContextMode) ContextUsage {
	switch mode {
	case ContextModePatched:
		return CalculatePatchedContextUsage(usage.InputTokens, usage.CachedInputTokens, contextWindow)
	case ContextModeCodex, "":
		return CalculateContextUsage(usage.TotalTokens, contextWindow)
	default:
		return CalculateContextUsage(usage.TotalTokens, contextWindow)
	}
}

func CalculateContextUsage(totalTokens int64, contextWindow int64) ContextUsage {
	if contextWindow <= 0 {
		return ContextUsage{}
	}

	// Small window: subtracting the baseline would make the denominator
	// invalid (effective window <= 0), which otherwise reports a misleading
	// 0% (0/0). Fall back to the raw bounded ratio for these models.
	if contextWindow <= baselineTokens {
		used := clampInt64(totalTokens, 0, contextWindow)
		return ContextUsage{
			UsedTokens:  used,
			TotalTokens: contextWindow,
			Percent:     percentOf(used, contextWindow),
		}
	}

	effectiveWindow := contextWindow - baselineTokens
	used := clampInt64(totalTokens-baselineTokens, 0, effectiveWindow)
	return ContextUsage{
		UsedTokens:  used,
		TotalTokens: effectiveWindow,
		Percent:     percentOf(used, effectiveWindow),
	}
}

func clampInt64(value, min, max int64) int64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func percentOf(used, total int64) int {
	if total <= 0 {
		return 0
	}
	percent := int(math.Round(float64(used) / float64(total) * 100))
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}

func CalculatePatchedContextUsage(inputTokens int64, cachedInputTokens int64, contextWindow int64) ContextUsage {
	if contextWindow <= 0 {
		return ContextUsage{}
	}

	used := max(inputTokens, 0) + max(cachedInputTokens, 0)
	if used > contextWindow {
		used = contextWindow
	}

	percent := int(math.Round(float64(used) / float64(contextWindow) * 100))
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	return ContextUsage{
		UsedTokens:  used,
		TotalTokens: contextWindow,
		Percent:     percent,
	}
}
