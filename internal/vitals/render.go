package vitals

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

const defaultContextBarWidth = 6

type RenderStyle string

const (
	RenderStyleCompact      RenderStyle = "compact"
	RenderStyleCurrentHUD   RenderStyle = "current-hud"
	RenderStyleAnswerFooter RenderStyle = "answer-footer"
)

type RenderOptions struct {
	Style RenderStyle
	Color bool
}

func ParseRenderStyle(value string) (RenderStyle, error) {
	switch RenderStyle(value) {
	case "", RenderStyleCompact:
		return RenderStyleCompact, nil
	case RenderStyleCurrentHUD:
		return RenderStyleCurrentHUD, nil
	case RenderStyleAnswerFooter:
		return RenderStyleAnswerFooter, nil
	default:
		return "", ErrInvalidRenderStyle
	}
}

var ErrInvalidRenderStyle = errInvalidRenderStyle{}

type errInvalidRenderStyle struct{}

func (errInvalidRenderStyle) Error() string {
	return "style must be one of: compact, current-hud, answer-footer"
}

func RenderLine(snapshot Snapshot, homeDir string) string {
	return RenderLineWithOptions(snapshot, homeDir, RenderOptions{Style: RenderStyleCompact})
}

func RenderLineWithOptions(snapshot Snapshot, homeDir string, options RenderOptions) string {
	switch options.Style {
	case RenderStyleCurrentHUD:
		return renderCurrentHUDLine(snapshot, homeDir, options.Color)
	case RenderStyleAnswerFooter:
		return renderAnswerFooter(snapshot, homeDir, options.Color)
	default:
		return renderCompactLine(snapshot, homeDir)
	}
}

func renderCompactLine(snapshot Snapshot, homeDir string) string {
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

func renderCurrentHUDLine(snapshot Snapshot, homeDir string, color bool) string {
	var segments []coloredSegment

	model := formatModel(snapshot.Model)
	if model != "" {
		segments = append(segments, coloredSegment{text: model, rgb: rgb{148, 226, 213}})
	}
	if snapshot.Session.CWD != "" {
		segments = append(segments, coloredSegment{
			text: abbreviateHome(snapshot.Session.CWD, homeDir),
			rgb:  rgb{166, 173, 200},
		})
	}
	if snapshot.Tokens.HasTokens {
		segments = append(segments, coloredSegment{
			text: fmt.Sprintf("Context %d%% used", snapshot.Tokens.Context.Percent),
			rgb:  rgb{245, 194, 231},
		})
	}
	if snapshot.Tokens.Primary != nil {
		segments = append(segments, coloredSegment{
			text: fmt.Sprintf("5h %.0f%% left", remainingPercent(snapshot.Tokens.Primary.UsedPercent)),
			rgb:  rgb{180, 190, 254},
		})
	}
	if snapshot.Tokens.Secondary != nil {
		segments = append(segments, coloredSegment{
			text: fmt.Sprintf("weekly %.0f%% left", remainingPercent(snapshot.Tokens.Secondary.UsedPercent)),
			rgb:  rgb{249, 226, 175},
		})
	}

	if len(segments) == 0 {
		return "codex-vitals: waiting for Codex session data"
	}
	return joinColoredSegments(segments, color)
}

func renderAnswerFooter(snapshot Snapshot, homeDir string, color bool) string {
	lines := make([]string, 0, 5)

	model := formatModel(snapshot.Model)
	if model != "" {
		lines = append(lines, colorizeIf("🤖 "+model, rgb{148, 226, 213}, color))
	}

	var location []string
	if snapshot.Session.GitBranch != "" {
		location = append(location, colorizeIf("🌿 "+snapshot.Session.GitBranch, rgb{166, 173, 200}, color))
	}
	if snapshot.Session.CWD != "" {
		location = append(location, colorizeIf("📁 "+abbreviateHome(snapshot.Session.CWD, homeDir), rgb{166, 173, 200}, color))
	}
	if len(location) > 0 {
		lines = append(lines, strings.Join(location, dimSeparator(color)))
	}

	if snapshot.Tokens.HasTokens {
		context := snapshot.Tokens.Context
		line := fmt.Sprintf(
			"🧠 Context %s %d%% used (%s/%s)",
			RenderBar(context.Percent, defaultContextBarWidth),
			context.Percent,
			FormatTokenCount(context.UsedTokens),
			FormatTokenCount(context.TotalTokens),
		)
		lines = append(lines, colorizeIf(line, rgb{245, 194, 231}, color))
	}

	var limits []string
	if snapshot.Tokens.Primary != nil {
		limits = append(limits, colorizeIf(fmt.Sprintf("⏱ 5h %.0f%% left", remainingPercent(snapshot.Tokens.Primary.UsedPercent)), rgb{180, 190, 254}, color))
	}
	if snapshot.Tokens.Secondary != nil {
		limits = append(limits, colorizeIf(fmt.Sprintf("📅 weekly %.0f%% left", remainingPercent(snapshot.Tokens.Secondary.UsedPercent)), rgb{249, 226, 175}, color))
	}
	if len(limits) > 0 {
		lines = append(lines, strings.Join(limits, dimSeparator(color)))
	}

	if len(lines) == 0 {
		return "codex-vitals: waiting for Codex session data"
	}
	return strings.Join(lines, "\n")
}

type coloredSegment struct {
	text string
	rgb  rgb
}

type rgb struct {
	red   int
	green int
	blue  int
}

func joinColoredSegments(segments []coloredSegment, color bool) string {
	rendered := make([]string, 0, len(segments))
	for _, segment := range segments {
		if color {
			rendered = append(rendered, colorize(segment.text, segment.rgb))
		} else {
			rendered = append(rendered, segment.text)
		}
	}
	separator := " · "
	if color {
		separator = "\x1b[2m · \x1b[0m"
	}
	return strings.Join(rendered, separator)
}

func colorize(text string, color rgb) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", color.red, color.green, color.blue, text)
}

func colorizeIf(text string, color rgb, enabled bool) string {
	if !enabled {
		return text
	}
	return colorize(text, color)
}

func dimSeparator(color bool) string {
	if color {
		return "\x1b[2m · \x1b[0m"
	}
	return " · "
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
