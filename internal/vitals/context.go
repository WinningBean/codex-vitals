package vitals

import "math"

const baselineTokens int64 = 12000

type ContextMode string

const (
	ContextModeCodex      ContextMode = "codex"
	ContextModeCurrentHUD ContextMode = "current-hud"
)

// ContextUsage is the effective context-window usage Codex displays after
// subtracting the reserved baseline from both used tokens and the window.
type ContextUsage struct {
	UsedTokens  int64
	TotalTokens int64
	Percent     int
}

type TokenUsage struct {
	TotalTokens       int64 `json:"total_tokens"`
	InputTokens       int64 `json:"input_tokens"`
	CachedInputTokens int64 `json:"cached_input_tokens"`
	OutputTokens      int64 `json:"output_tokens"`
}

func ParseContextMode(value string) (ContextMode, error) {
	switch ContextMode(value) {
	case "", ContextModeCodex:
		return ContextModeCodex, nil
	case ContextModeCurrentHUD:
		return ContextModeCurrentHUD, nil
	default:
		return "", ErrInvalidContextMode
	}
}

var ErrInvalidContextMode = errInvalidContextMode{}

type errInvalidContextMode struct{}

func (errInvalidContextMode) Error() string {
	return "context mode must be one of: codex, current-hud"
}

func CalculateContextUsageForMode(usage TokenUsage, contextWindow int64, mode ContextMode) ContextUsage {
	switch mode {
	case ContextModeCurrentHUD:
		return CalculateCurrentHUDContextUsage(usage.InputTokens, usage.CachedInputTokens, contextWindow)
	case ContextModeCodex, "":
		return CalculateContextUsage(usage.TotalTokens, contextWindow)
	default:
		return CalculateContextUsage(usage.TotalTokens, contextWindow)
	}
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

func CalculateCurrentHUDContextUsage(inputTokens int64, cachedInputTokens int64, contextWindow int64) ContextUsage {
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
