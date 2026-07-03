package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/WinningBean/codex-vitals/internal/vitals"
)

func main() {
	rolloutPath := flag.String("rollout", "", "path to a rollout JSONL file")
	codexHome := flag.String("codex-home", "", "path to CODEX_HOME; defaults to $CODEX_HOME or ~/.codex")
	contextModeValue := flag.String("context-mode", string(vitals.ContextModeCodex), "context usage formula: codex or current-hud")
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

	options := vitals.LoadOptions{
		CodexHome:   *codexHome,
		RolloutPath: *rolloutPath,
		ContextMode: contextMode,
	}
	if *once {
		fmt.Println(render(options, homeDir))
		return
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	fmt.Print(render(options, homeDir))
	for {
		select {
		case <-ticker.C:
			fmt.Printf("\r\033[2K%s", render(options, homeDir))
		case <-signals:
			fmt.Println()
			return
		}
	}
}

func render(options vitals.LoadOptions, homeDir string) string {
	snapshot, err := vitals.LoadSnapshot(options)
	if err != nil && !errors.Is(err, vitals.ErrNoRollout) {
		return "codex-vitals: " + err.Error()
	}
	return vitals.RenderLine(snapshot, homeDir)
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "codex-vitals:", err)
	os.Exit(1)
}
