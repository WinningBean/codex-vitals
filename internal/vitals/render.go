package vitals

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

const defaultContextBarWidth = 6

func RenderLine(snapshot Snapshot, homeDir string) string {
	var segments []string

	model := formatModel(snapshot.Model)
	if model != "" {
		segments = append(segments, "🤖 "+model)
	}
	if snapshot.Session.GitBranch != "" {
		segments = append(segments, "🌿 "+snapshot.Session.GitBranch)
	}
	if snapshot.Session.CWD != "" {
		segments = append(segments, abbreviateHome(snapshot.Session.CWD, homeDir))
	}
	if snapshot.Tokens.HasTokens {
		context := snapshot.Tokens.Context
		segments = append(segments, fmt.Sprintf(
			"Ctx %s %d%% (%s/%s)",
			RenderBar(context.Percent, defaultContextBarWidth),
			context.Percent,
			FormatTokenCount(context.UsedTokens),
			FormatTokenCount(context.TotalTokens),
		))
	}
	if snapshot.Tokens.Primary != nil {
		segments = append(segments, fmt.Sprintf("5h %.0f%%", remainingPercent(snapshot.Tokens.Primary.UsedPercent)))
	}
	if snapshot.Tokens.Secondary != nil {
		segments = append(segments, fmt.Sprintf("wk %.0f%%", remainingPercent(snapshot.Tokens.Secondary.UsedPercent)))
	}

	if len(segments) == 0 {
		return "codex-vitals: waiting for Codex session data"
	}
	return strings.Join(segments, " · ")
}

func formatModel(model ModelInfo) string {
	switch {
	case model.Model != "" && model.Effort != "":
		return model.Model + " " + model.Effort
	case model.Model != "":
		return model.Model
	case model.Effort != "":
		return model.Effort
	default:
		return ""
	}
}

func RenderBar(percent int, width int) string {
	if width <= 0 {
		return ""
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(math.Round(float64(percent) / 100 * float64(width)))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func FormatTokenCount(tokens int64) string {
	sign := ""
	if tokens < 0 {
		sign = "-"
		tokens = -tokens
	}
	if tokens < 1000 {
		return fmt.Sprintf("%s%d", sign, tokens)
	}
	if tokens < 1_000_000 {
		return fmt.Sprintf("%s%dk", sign, tokens/1000)
	}
	return fmt.Sprintf("%s%.1fm", sign, float64(tokens)/1_000_000)
}

func abbreviateHome(path string, homeDir string) string {
	if homeDir == "" {
		return path
	}
	cleanPath := filepath.Clean(path)
	cleanHome := filepath.Clean(homeDir)
	if cleanPath == cleanHome {
		return "~"
	}
	prefix := cleanHome + string(filepath.Separator)
	if strings.HasPrefix(cleanPath, prefix) {
		return "~" + string(filepath.Separator) + strings.TrimPrefix(cleanPath, prefix)
	}
	return path
}

func remainingPercent(used float64) float64 {
	remaining := 100 - used
	if remaining < 0 {
		return 0
	}
	if remaining > 100 {
		return 100
	}
	return remaining
}
