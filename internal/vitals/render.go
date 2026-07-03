package vitals

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultContextBarWidth = 6
const answerFooterBarWidth = 20

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
	lines := make([]string, 0, 6)
	separator := " "

	model := formatModel(snapshot.Model)
	if model != "" {
		modelLabel := snapshot.Model.Model
		if snapshot.Model.Effort != "" {
			modelLabel += " ⚡" + snapshot.Model.Effort
		}
		lines = append(lines, fmt.Sprintf(
			"  🤖 %s%s%s%s%s",
			colorizeIf(modelLabel, rgb{148, 226, 213}, color),
			separator,
			renderGitDirty(snapshot.Session.CWD, color),
			separator,
			renderEnv(color),
		))
	}

	if snapshot.Session.CWD != "" {
		line := fmt.Sprintf(
			"  📂 %s",
			colorizeIf(snapshot.Session.CWD, rgb{166, 173, 200}, color),
		)
		if snapshot.Session.GitBranch != "" {
			line += " " + colorizeIf("🌿("+strings.TrimSuffix(snapshot.Session.GitBranch, "*")+")", rgb{64, 160, 43}, color)
		}
		if snapshot.Tokens.HasTokens {
			line += separator + "🧾 " + colorizeIf(FormatTokenCount(snapshot.Tokens.SessionTotalTokens)+" tokens", rgb{250, 179, 135}, color)
		}
		if !snapshot.StartedAt.IsZero() {
			line += separator + dimIf("⏰ "+formatDuration(sessionDuration(snapshot)), color)
		}
		lines = append(lines, line)
	}

	if snapshot.Tokens.HasTokens {
		context := snapshot.Tokens.Context
		lines = append(lines, renderAnswerFooterUsage(
			"🧠",
			"Context",
			context.Percent,
			"used",
			fmt.Sprintf("(%s/%s)", FormatTokenCount(context.UsedTokens), FormatTokenCount(context.TotalTokens)),
			rgb{245, 194, 231},
			"context",
			color,
		))
	}

	if snapshot.Tokens.Primary != nil {
		lines = append(lines, renderAnswerFooterUsage(
			"🚀",
			"Usage 5H",
			int(math.Round(snapshot.Tokens.Primary.UsedPercent)),
			"",
			formatReset(snapshot.Tokens.Primary, "left"),
			rgb{180, 190, 254},
			"default",
			color,
		))
	}
	if snapshot.Tokens.Secondary != nil {
		lines = append(lines, renderAnswerFooterUsage(
			"⭐",
			"Usage 7D",
			int(math.Round(snapshot.Tokens.Secondary.UsedPercent)),
			"",
			formatReset(snapshot.Tokens.Secondary, "datetime"),
			rgb{249, 226, 175},
			"7d",
			color,
		))
	}

	if len(lines) == 0 {
		return "codex-vitals: waiting for Codex session data"
	}
	return strings.Join(lines, "\n")
}

func renderAnswerFooterUsage(icon string, label string, percent int, suffix string, extra string, color rgb, gradientKind string, useColor bool) string {
	prefix := fmt.Sprintf("  %s %s", icon, label)
	gap := " "
	if label == "Context" {
		gap = "  "
	}
	bar := RenderBar(percent, answerFooterBarWidth)
	if useColor {
		bar = renderGradientBar(percent, answerFooterBarWidth, gradientKind)
		prefix = fmt.Sprintf("  %s %s", icon, colorize(label, color))
		value := fmt.Sprintf("%d%%", percent)
		if suffix != "" {
			value += " " + suffix
		}
		suffix = colorize(value, gradientColor(float64(percent), gradientKind))
		if extra != "" {
			extra = " " + dimText(extra)
		}
		return fmt.Sprintf("%s%s%s %s%s", prefix, gap, bar, suffix, extra)
	}
	value := fmt.Sprintf("%d%%", percent)
	if suffix != "" {
		value += " " + suffix
	}
	line := fmt.Sprintf("%s%s%s %s", prefix, gap, bar, value)
	if extra != "" {
		line += " " + extra
	}
	return line
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

func renderGradientBar(percent int, width int, gradientKind string) string {
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
	var builder strings.Builder
	for i := 0; i < filled; i++ {
		blockColor := gradientColor(float64(i)*100/float64(width), gradientKind)
		builder.WriteString(colorize("█", blockColor))
	}
	if empty := width - filled; empty > 0 {
		builder.WriteString(colorize(strings.Repeat("░", empty), gradientColor(float64(percent), gradientKind)))
	}
	return builder.String()
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

func dimText(text string) string {
	if text == "" {
		return ""
	}
	return "\x1b[2m" + text + "\x1b[0m"
}

func gradientColor(percent float64, kind string) rgb {
	percent = math.Max(0, math.Min(100, math.Round(percent)))
	switch kind {
	case "context":
		if percent < 30 {
			t := percent * 100 / 30
			return rgb{lerp(245, 230, t), lerp(194, 69, t), lerp(231, 83, t)}
		}
		if percent < 70 {
			t := (percent - 30) * 100 / 40
			return rgb{lerp(230, 210, t), lerp(69, 15, t), lerp(83, 57, t)}
		}
		return rgb{210, 15, 57}
	case "7d":
		if percent < 50 {
			t := percent * 2
			return rgb{lerp(249, 254, t), lerp(226, 100, t), lerp(175, 11, t)}
		}
		t := (percent - 50) * 2
		return rgb{lerp(254, 210, t), lerp(100, 15, t), lerp(11, 57, t)}
	default:
		if percent < 50 {
			t := percent * 2
			return rgb{lerp(180, 30, t), lerp(190, 102, t), lerp(254, 245, t)}
		}
		t := (percent - 50) * 2
		return rgb{lerp(30, 210, t), lerp(102, 15, t), lerp(245, 57, t)}
	}
}

func lerp(start int, end int, t float64) int {
	return int(math.Round(float64(start) + (float64(end-start) * t / 100)))
}

func renderGitDirty(cwd string, color bool) string {
	if cwd == "" {
		return dimIf("no git", color)
	}
	output, err := exec.Command("git", "-C", cwd, "status", "--porcelain", "--untracked-files=no").Output()
	if err != nil {
		return dimIf("no git", color)
	}
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	if count == 0 {
		return colorizeIf("✅ clean", rgb{64, 160, 43}, color)
	}
	return colorizeIf("📝 !"+strconv.Itoa(count), rgb{223, 142, 29}, color)
}

func renderEnv(color bool) string {
	envLabel := os.Getenv("CONDA_DEFAULT_ENV")
	if envLabel == "" {
		envLabel = os.Getenv("VIRTUAL_ENV")
	}
	if envLabel == "" {
		return dimIf("no env", color)
	}
	if strings.Contains(envLabel, string(os.PathSeparator)) {
		envLabel = filepath.Base(envLabel)
	}
	return "🐍 " + colorizeIf(envLabel, rgb{64, 160, 43}, color)
}

func dimIf(text string, color bool) string {
	if color {
		return dimText(text)
	}
	return text
}

func sessionDuration(snapshot Snapshot) time.Duration {
	end := snapshot.EndedAt
	if end.IsZero() {
		end = time.Now()
	}
	if end.Before(snapshot.StartedAt) {
		return 0
	}
	return end.Sub(snapshot.StartedAt)
}

func formatDuration(duration time.Duration) string {
	seconds := int(duration.Seconds())
	if seconds < 0 {
		seconds = 0
	}
	if seconds >= 86400 {
		return fmt.Sprintf("%dd%dh", seconds/86400, seconds%86400/3600)
	}
	if seconds >= 3600 {
		return fmt.Sprintf("%dh%dm", seconds/3600, seconds%3600/60)
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	return fmt.Sprintf("%dm", seconds/60)
}

func formatReset(limit *RateLimit, mode string) string {
	if limit == nil || limit.ResetsAt <= 0 {
		return ""
	}
	reset := time.Unix(limit.ResetsAt, 0)
	if mode == "datetime" {
		return "(Reset " + reset.Format("Mon 15:04") + ")"
	}
	remaining := time.Until(reset)
	if remaining <= 0 {
		return ""
	}
	return "(Reset " + formatDuration(remaining) + " left)"
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
	return fmt.Sprintf("%s%.1fM", sign, float64(tokens)/1_000_000)
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
