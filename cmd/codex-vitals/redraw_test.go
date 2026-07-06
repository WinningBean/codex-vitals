package main

import "testing"

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"plain ascii", "hello", 5},
		{"ansi color is invisible", "\x1b[38;2;1;2;3mhi\x1b[0m", 2},
		{"emoji counts as two", "🧠", 2},
		{"emoji with label", "🧠 Context", 2 + 1 + 7},
		{"box drawing stays one", "│██░░", 5},
		{"panel emojis are wide", "🤖📂🌿🚀📅📝🐍🧾⚡✅⏰", 22},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := displayWidth(tt.in); got != tt.want {
				t.Fatalf("displayWidth(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestFrameRows(t *testing.T) {
	// Unknown width falls back to the logical line count.
	if got := frameRows("a\nbb\nccc", 0); got != 3 {
		t.Fatalf("unknown width rows = %d, want 3", got)
	}

	// Lines within the width take one row each.
	if got := frameRows("abc\nde", 80); got != 2 {
		t.Fatalf("no-wrap rows = %d, want 2", got)
	}

	// A line wider than the terminal wraps onto extra rows.
	// "aaaaaaaaaa" (10) at width 4 -> ceil(10/4) = 3 rows; second line 1 row.
	if got := frameRows("aaaaaaaaaa\nb", 4); got != 4 {
		t.Fatalf("wrap rows = %d, want 4", got)
	}

	// A line exactly filling the width stays one row (ceil(4/4) = 1).
	if got := frameRows("aaaa\nb", 4); got != 2 {
		t.Fatalf("exact-fill rows = %d, want 2", got)
	}

	// Wide runes consume two columns, so an emoji-heavy line wraps sooner.
	// "🧠🧠🧠" is width 6; at width 4 -> ceil(6/4) = 2 rows.
	if got := frameRows("🧠🧠🧠", 4); got != 2 {
		t.Fatalf("emoji wrap rows = %d, want 2", got)
	}
}
