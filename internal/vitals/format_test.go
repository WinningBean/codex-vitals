package vitals

import (
	"strings"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	cases := map[time.Duration]string{
		30 * time.Second:    "30s",
		90 * time.Second:    "1m",
		3661 * time.Second:  "1h1m",
		90000 * time.Second: "1d1h",
		-5 * time.Second:    "0s",
		0:                   "0s",
	}
	for in, want := range cases {
		if got := formatDuration(in); got != want {
			t.Errorf("formatDuration(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestSessionDuration(t *testing.T) {
	start := time.Date(2026, time.July, 5, 10, 0, 0, 0, time.UTC)
	// EndedAt after StartedAt.
	s := Snapshot{StartedAt: start, EndedAt: start.Add(90 * time.Second)}
	if got := sessionDuration(s); got != 90*time.Second {
		t.Errorf("sessionDuration = %v, want 90s", got)
	}
	// EndedAt before StartedAt clamps to 0.
	s = Snapshot{StartedAt: start, EndedAt: start.Add(-time.Minute)}
	if got := sessionDuration(s); got != 0 {
		t.Errorf("sessionDuration (reversed) = %v, want 0", got)
	}
}

func TestFormatTokenCount(t *testing.T) {
	cases := map[int64]string{
		0:         "0",
		500:       "500",
		1500:      "1k",
		999999:    "999k",
		2_500_000: "2.5M",
		-1500:     "-1k",
	}
	for in, want := range cases {
		if got := FormatTokenCount(in); got != want {
			t.Errorf("FormatTokenCount(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderBar(t *testing.T) {
	cases := []struct {
		percent, width int
		want           string
	}{
		{50, 10, "█████░░░░░"},
		{0, 10, "░░░░░░░░░░"},
		{100, 10, "██████████"},
		{150, 10, "██████████"}, // clamped high
		{-10, 10, "░░░░░░░░░░"}, // clamped low
		{50, 0, ""}, // no width
	}
	for _, c := range cases {
		if got := RenderBar(c.percent, c.width); got != c.want {
			t.Errorf("RenderBar(%d,%d) = %q, want %q", c.percent, c.width, got, c.want)
		}
	}
}

func TestFormatReset(t *testing.T) {
	if got := formatReset(nil, "left"); got != "" {
		t.Errorf("formatReset(nil) = %q, want empty", got)
	}
	if got := formatReset(&RateLimit{ResetsAt: 0}, "datetime"); got != "" {
		t.Errorf("formatReset(0) = %q, want empty", got)
	}

	epoch := int64(1783072800)
	want := "(Reset " + time.Unix(epoch, 0).Format("Mon 15:04") + ")"
	if got := formatReset(&RateLimit{ResetsAt: epoch}, "datetime"); got != want {
		t.Errorf("formatReset datetime = %q, want %q", got, want)
	}

	future := time.Now().Add(2 * time.Hour).Unix()
	got := formatReset(&RateLimit{ResetsAt: future}, "left")
	if !strings.HasPrefix(got, "(Reset ") || !strings.HasSuffix(got, "left)") {
		t.Errorf("formatReset left(future) = %q, want (Reset ... left)", got)
	}
	past := time.Now().Add(-time.Hour).Unix()
	if got := formatReset(&RateLimit{ResetsAt: past}, "left"); got != "" {
		t.Errorf("formatReset left(past) = %q, want empty", got)
	}
}

func TestModelBolt(t *testing.T) {
	cases := []struct {
		in   ModelInfo
		want string
	}{
		{ModelInfo{Model: "gpt-5.5", Effort: "xhigh"}, "gpt-5.5 ⚡xhigh"},
		{ModelInfo{Model: "gpt-5.5"}, "gpt-5.5"},
		{ModelInfo{Effort: "high"}, "high"},
		{ModelInfo{}, ""},
	}
	for _, c := range cases {
		if got := modelBolt(c.in); got != c.want {
			t.Errorf("modelBolt(%+v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDimHelpers(t *testing.T) {
	if got := dimText(""); got != "" {
		t.Errorf("dimText(\"\") = %q, want empty", got)
	}
	if got := dimText("x"); got != "\x1b[2mx\x1b[0m" {
		t.Errorf("dimText = %q", got)
	}
	if got := dimIf("x", false); got != "x" {
		t.Errorf("dimIf(no color) = %q, want x", got)
	}
	if got := dimIf("x", true); got != "\x1b[2mx\x1b[0m" {
		t.Errorf("dimIf(color) = %q", got)
	}
}

func TestErrorMessages(t *testing.T) {
	if _, err := ParseSize("nope"); err == nil || err.Error() != "size must be one of: xs, s, m, l, xl" {
		t.Errorf("ParseSize error = %v", err)
	}
	if _, err := ParseContextMode("nope"); err == nil || err.Error() != "context mode must be one of: codex, patched" {
		t.Errorf("ParseContextMode error = %v", err)
	}
}
