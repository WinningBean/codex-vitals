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

// Shared palette (Catppuccin-ish).
var (
	cModel  = rgb{148, 226, 213}
	cBranch = rgb{64, 160, 43}
	cPath   = rgb{166, 173, 200}
	cTokens = rgb{250, 179, 135}
	cCtx    = rgb{245, 194, 231}
	c5H     = rgb{180, 190, 254}
	c7D     = rgb{249, 226, 175}
	cDirty  = rgb{223, 142, 29}
)

// Size controls how much the HUD shows, xs (least) to xl (most).
type Size string

const (
	SizeXS Size = "xs"
	SizeS  Size = "s"
	SizeM  Size = "m"
	SizeL  Size = "l"
	SizeXL Size = "xl"
)

func ParseSize(value string) (Size, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "m", "medium":
		return SizeM, nil
	case "xs", "xsmall":
		return SizeXS, nil
	case "s", "small":
		return SizeS, nil
	case "l", "large":
		return SizeL, nil
	case "xl", "xlarge":
		return SizeXL, nil
	default:
		return "", ErrInvalidSize
	}
}

var ErrInvalidSize = errInvalidSize{}

type errInvalidSize struct{}

func (errInvalidSize) Error() string {
	return "size must be one of: xs, s, m, l, xl"
}

type RenderOptions struct {
	Size  Size
	Color bool
}

func RenderLine(snapshot Snapshot, homeDir string) string {
	return RenderLineWithOptions(snapshot, homeDir, RenderOptions{Size: SizeM})
}

func RenderLineWithOptions(snapshot Snapshot, homeDir string, options RenderOptions) string {
	if formatModel(snapshot.Model) == "" && !snapshot.Tokens.HasTokens && snapshot.Session.CWD == "" {
		return "codex-vitals: waiting for Codex session data"
	}
	switch options.Size {
	case SizeXS:
		return renderXS(snapshot, homeDir, options.Color)
	case SizeS:
		return renderS(snapshot, homeDir, options.Color)
	case SizeL:
		return renderLarge(snapshot, homeDir, options.Color, 20)
	case SizeXL:
		return renderLarge(snapshot, homeDir, options.Color, 40)
	default:
		return renderM(snapshot, homeDir, options.Color)
	}
}

// --- size layouts ----------------------------------------------------------

// xs: two lines, tiny bars, no labels or percentages.
func renderXS(s Snapshot, home string, color bool) string {
	var l1 []string
	if m := modelBolt(s.Model); m != "" {
		l1 = append(l1, "🤖 "+colorizeIf(m, cModel, color))
	}
	if s.Session.CWD != "" {
		l1 = append(l1, "📂 "+colorizeIf(abbreviateHome(s.Session.CWD, home), cPath, color))
	}
	if s.Session.GitBranch != "" {
		l1 = append(l1, colorizeIf(branchText(s), cBranch, color))
	}

	l2 := []string{"🧠 " + bar(s.Tokens.Context.Percent, 10, "context", color)}
	if s.Tokens.Primary != nil {
		l2 = append(l2, "5H "+bar(pct(s.Tokens.Primary), 10, "default", color))
	}
	if s.Tokens.Secondary != nil {
		l2 = append(l2, "7D "+bar(pct(s.Tokens.Secondary), 10, "7d", color))
	}
	return strings.Join(l1, " ") + "\n" + strings.Join(l2, "  ")
}

// s: two lines, labels and percentages, pipe separators.
func renderS(s Snapshot, home string, color bool) string {
	var g1 []string
	if m := modelBolt(s.Model); m != "" {
		g1 = append(g1, "🤖 "+colorizeIf(m, cModel, color))
	}
	if s.Session.CWD != "" {
		p := "📂 " + colorizeIf(abbreviateHome(s.Session.CWD, home), cPath, color)
		if s.Session.GitBranch != "" {
			p += " " + colorizeIf(branchText(s), cBranch, color)
		}
		g1 = append(g1, p)
	}

	g2 := []string{labeledBar("🧠", "Context", s.Tokens.Context.Percent, 10, "context", cCtx, color)}
	if s.Tokens.Primary != nil {
		g2 = append(g2, labeledBar("", "5H", pct(s.Tokens.Primary), 10, "default", c5H, color))
	}
	if s.Tokens.Secondary != nil {
		g2 = append(g2, labeledBar("", "7D", pct(s.Tokens.Secondary), 10, "7d", c7D, color))
	}
	return strings.Join(g1, pipe(color)) + "\n" + strings.Join(g2, pipe(color))
}

// m (default): four lines, full-width context bar.
func renderM(s Snapshot, home string, color bool) string {
	lines := make([]string, 0, 4)

	if m := modelBolt(s.Model); m != "" {
		lines = append(lines, strings.Join([]string{
			"🤖 " + colorizeIf(m, cModel, color),
			renderGitDirty(s.Session.CWD, color),
			renderEnv(color),
		}, pipe(color)))
	}
	if s.Session.CWD != "" {
		line := "📂 " + colorizeIf(abbreviateHome(s.Session.CWD, home), cPath, color)
		if s.Session.GitBranch != "" {
			line += " " + colorizeIf(branchText(s), cBranch, color)
		}
		lines = append(lines, line)
	}
	if s.Tokens.HasTokens {
		lines = append(lines, "🧠 "+colorizeIf("Context", cCtx, color)+" "+
			bar(s.Tokens.Context.Percent, 30, "context", color)+" "+
			pctText(s.Tokens.Context.Percent, "context", color)+" "+dimIf("used", color))
	}
	var usage []string
	if s.Tokens.Primary != nil {
		usage = append(usage, labeledBar("🚀", "Usage 5H", pct(s.Tokens.Primary), 10, "default", c5H, color))
	}
	if s.Tokens.Secondary != nil {
		usage = append(usage, labeledBar("📅", "7D", pct(s.Tokens.Secondary), 10, "7d", c7D, color))
	}
	if len(usage) > 0 {
		lines = append(lines, strings.Join(usage, pipe(color)))
	}
	return strings.Join(lines, "\n")
}

// l / xl: five lines with reset times and token counts; xl uses wider bars.
func renderLarge(s Snapshot, home string, color bool, barWidth int) string {
	lines := make([]string, 0, 5)

	if m := modelBolt(s.Model); m != "" {
		lines = append(lines, strings.Join([]string{
			"🤖 " + colorizeIf(m, cModel, color),
			renderGitDirty(s.Session.CWD, color),
			renderEnv(color),
		}, pipe(color)))
	}
	if s.Session.CWD != "" {
		parts := []string{"📂 " + colorizeIf(abbreviateHome(s.Session.CWD, home), cPath, color)}
		if s.Session.GitBranch != "" {
			parts[0] += " " + colorizeIf(branchText(s), cBranch, color)
		}
		if s.Tokens.HasTokens {
			parts = append(parts, "🧾 "+colorizeIf(FormatTokenCount(s.Tokens.SessionTotalTokens)+" tokens", cTokens, color))
		}
		if !s.StartedAt.IsZero() {
			parts = append(parts, dimIf("⏰ "+formatDuration(sessionDuration(s)), color))
		}
		lines = append(lines, strings.Join(parts, pipe(color)))
	}
	if s.Tokens.HasTokens {
		c := s.Tokens.Context
		lines = append(lines, usageLine("🧠", "Context", c.Percent, barWidth, "context", cCtx,
			"used", fmt.Sprintf("(%s/%s)", FormatTokenCount(c.UsedTokens), FormatTokenCount(c.TotalTokens)), color))
	}
	if s.Tokens.Primary != nil {
		lines = append(lines, usageLine("🚀", "Usage 5H", pct(s.Tokens.Primary), barWidth, "default", c5H,
			"", formatReset(s.Tokens.Primary, "left"), color))
	}
	if s.Tokens.Secondary != nil {
		lines = append(lines, usageLine("📅", "Usage 7D", pct(s.Tokens.Secondary), barWidth, "7d", c7D,
			"", formatReset(s.Tokens.Secondary, "datetime"), color))
	}
	return strings.Join(lines, "\n")
}

// --- building blocks -------------------------------------------------------

// labeledBar: "icon Label ██░ N%" (icon optional). Used by s and m.
func labeledBar(icon, label string, percent, width int, kind string, labelColor rgb, color bool) string {
	var b strings.Builder
	if icon != "" {
		b.WriteString(icon + " ")
	}
	b.WriteString(colorizeIf(label, labelColor, color))
	b.WriteString(" " + bar(percent, width, kind, color))
	b.WriteString(" " + pctText(percent, kind, color))
	return b.String()
}

// usageLine: "  icon Label ███ N% used (extra)" for the large layouts.
func usageLine(icon, label string, percent, width int, kind string, labelColor rgb, suffix, extra string, color bool) string {
	value := fmt.Sprintf("%d%%", percent)
	if suffix != "" {
		value += " " + suffix
	}
	labelText := colorizeIf(label, labelColor, color)
	valueText := value
	if color {
		valueText = colorize(value, gradientColor(float64(percent), kind))
	}
	line := icon + " " + labelText + " " + bar(percent, width, kind, color) + " " + valueText
	if extra != "" {
		line += " " + dimIf(extra, color)
	}
	return line
}

func modelBolt(m ModelInfo) string {
	switch {
	case m.Model != "" && m.Effort != "":
		return m.Model + " ⚡" + m.Effort
	case m.Model != "":
		return m.Model
	default:
		return m.Effort
	}
}

func branchText(s Snapshot) string {
	return "🌿(" + strings.TrimSuffix(s.Session.GitBranch, "*") + ")"
}

func pct(limit *RateLimit) int {
	if limit == nil {
		return 0
	}
	return int(math.Round(limit.UsedPercent))
}

func bar(percent, width int, kind string, color bool) string {
	if color {
		return renderGradientBar(percent, width, kind)
	}
	return RenderBar(percent, width)
}

func pctText(percent int, kind string, color bool) string {
	s := fmt.Sprintf("%d%%", percent)
	if color {
		return colorize(s, gradientColor(float64(percent), kind))
	}
	return s
}

func pipe(color bool) string {
	if color {
		return " \x1b[2m│\x1b[0m "
	}
	return " │ "
}

// --- shared helpers --------------------------------------------------------

type rgb struct {
	red   int
	green int
	blue  int
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

func dimText(text string) string {
	if text == "" {
		return ""
	}
	return "\x1b[2m" + text + "\x1b[0m"
}

func dimIf(text string, color bool) string {
	if color {
		return dimText(text)
	}
	return text
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
	output, err := exec.Command("git", "-C", cwd, "status", "--porcelain").Output()
	if err != nil {
		return dimIf("no git", color)
	}
	added, modified, deleted, untracked := 0, 0, 0, 0
	for _, line := range strings.Split(strings.TrimRight(string(output), "\n"), "\n") {
		if len(line) < 2 {
			continue
		}
		code := line[:2]
		switch {
		case code == "??":
			untracked++
		case strings.Contains(code, "A"):
			added++
		case strings.Contains(code, "D"):
			deleted++
		default:
			modified++
		}
	}
	if added+modified+deleted+untracked == 0 {
		return colorizeIf("✅ clean", cBranch, color)
	}
	var parts []string
	if added > 0 {
		parts = append(parts, "+"+strconv.Itoa(added))
	}
	if modified > 0 {
		parts = append(parts, "!"+strconv.Itoa(modified))
	}
	if deleted > 0 {
		parts = append(parts, "-"+strconv.Itoa(deleted))
	}
	if untracked > 0 {
		parts = append(parts, "?"+strconv.Itoa(untracked))
	}
	return colorizeIf("📝 "+strings.Join(parts, " "), cDirty, color)
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
	return "🐍 " + colorizeIf(envLabel, cBranch, color)
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
