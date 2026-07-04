package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/WinningBean/codex-vitals/internal/vitals"
)

func main() {
	rolloutPath := flag.String("rollout", "", "path to a rollout JSONL file")
	codexHome := flag.String("codex-home", "", "path to CODEX_HOME; defaults to $CODEX_HOME or ~/.codex")
	contextModeValue := flag.String("context-mode", string(vitals.ContextModeCodex), "context usage formula: codex or current-hud")
	styleValue := flag.String("style", string(vitals.RenderStyleCompact), "render style: compact, current-hud, or answer-footer")
	noColor := flag.Bool("no-color", false, "disable ANSI colors")
	interval := flag.Duration("interval", time.Second, "refresh interval")
	once := flag.Bool("once", false, "render once and exit")
	flag.Parse()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		exitErr(err)
	}
	if *codexHome == "" {
		*codexHome = vitals.DefaultCodexHome(homeDir)
	}
	if *interval <= 0 {
		exitErr(fmt.Errorf("interval must be positive"))
	}
	contextMode, err := vitals.ParseContextMode(*contextModeValue)
	if err != nil {
		exitErr(err)
	}
	style, err := vitals.ParseRenderStyle(*styleValue)
	if err != nil {
		exitErr(err)
	}

	options := vitals.LoadOptions{
		CodexHome:   *codexHome,
		RolloutPath: *rolloutPath,
		ContextMode: contextMode,
	}
	if *once {
		fmt.Println(render(options, homeDir, newRenderOptions(style, !*noColor)))
		return
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	renderOptions := newRenderOptions(style, !*noColor)
	frame := render(options, homeDir, renderOptions)
	fmt.Print(frame)
	prevLines := strings.Count(frame, "\n") + 1
	for {
		select {
		case <-ticker.C:
			// Move the cursor back to the top of the previous frame and clear
			// it, so multi-line output redraws in place instead of scrolling.
			// Emitted together with the new frame in one write to avoid flicker.
			clear := "\r\033[2K"
			if prevLines > 1 {
				clear = fmt.Sprintf("\r\033[%dA\033[J", prevLines-1)
			}
			frame = render(options, homeDir, renderOptions)
			fmt.Print(clear + frame)
			prevLines = strings.Count(frame, "\n") + 1
		case <-signals:
			fmt.Println()
			return
		}
	}
}

func render(options vitals.LoadOptions, homeDir string, renderOptions vitals.RenderOptions) string {
	snapshot, err := vitals.LoadSnapshot(options)
	if err != nil && !errors.Is(err, vitals.ErrNoRollout) {
		return "codex-vitals: " + err.Error()
	}
	return vitals.RenderLineWithOptions(snapshot, homeDir, renderOptions)
}

func newRenderOptions(style vitals.RenderStyle, color bool) vitals.RenderOptions {
	return vitals.RenderOptions{
		Style: style,
		Color: color,
	}
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "codex-vitals:", err)
	os.Exit(1)
}
