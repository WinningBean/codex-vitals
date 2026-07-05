package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/WinningBean/codex-vitals/internal/vitals"
)

func writeSize(t *testing.T, dir, size string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := filepath.Join(dir, "codex-vitals", "size")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveSize(t *testing.T) {
	// Explicit flag pins the size, ignoring file/env.
	xl := vitals.SizeXL
	writeSize(t, t.TempDir(), "s")
	t.Setenv("CODEX_VITALS_SIZE", "xs")
	if got := resolveSize(&xl); got != vitals.SizeXL {
		t.Errorf("explicit = %q, want xl", got)
	}

	// Config file wins over env.
	writeSize(t, t.TempDir(), "l")
	t.Setenv("CODEX_VITALS_SIZE", "s")
	if got := resolveSize(nil); got != vitals.SizeL {
		t.Errorf("file>env = %q, want l", got)
	}

	// No file -> env.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEX_VITALS_SIZE", "xs")
	if got := resolveSize(nil); got != vitals.SizeXS {
		t.Errorf("env = %q, want xs", got)
	}

	// Nothing set -> m.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEX_VITALS_SIZE", "")
	if got := resolveSize(nil); got != vitals.SizeM {
		t.Errorf("default = %q, want m", got)
	}
}
