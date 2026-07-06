package main

import (
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// terminalWidth returns the current terminal column count, or 0 when it can't be
// determined (e.g. output is redirected to a pipe or file).
func terminalWidth() int {
	var ws struct {
		row, col, xpixel, ypixel uint16
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		os.Stdout.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return 0
	}
	return int(ws.col)
}

// frameRows reports how many physical terminal rows a rendered frame occupies at
// the given width, accounting for long lines that wrap. When width <= 0 (unknown
// terminal), it falls back to the logical line count so the redraw still works
// for the common single-line case.
func frameRows(frame string, width int) int {
	lines := strings.Split(frame, "\n")
	if width <= 0 {
		return len(lines)
	}
	rows := 0
	for _, line := range lines {
		w := displayWidth(line)
		if w <= 0 {
			rows++ // an empty line still occupies one row
			continue
		}
		rows += (w-1)/width + 1 // ceil(w / width)
	}
	return rows
}

// displayWidth returns the visible column width of s: ANSI escape sequences count
// for nothing, and wide runes (emoji, CJK) count for two columns.
func displayWidth(s string) int {
	width := 0
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == 0x1b { // ESC: skip a CSI sequence like "\x1b[38;2;…m"
			if i+1 < len(runes) && runes[i+1] == '[' {
				i += 2
				for i < len(runes) && !(runes[i] >= 0x40 && runes[i] <= 0x7e) {
					i++
				}
				continue // the loop's i++ skips the final byte
			}
			continue
		}
		width += runeWidth(r)
	}
	return width
}

// runeWidth is a compact wcwidth: 0 for combining marks and variation selectors,
// 2 for the emoji and CJK ranges the panel uses, 1 otherwise. Box-drawing and
// block glyphs (│ █ ░, all < 0x2600) stay width 1.
func runeWidth(r rune) int {
	switch {
	case r == 0:
		return 0
	case r >= 0x0300 && r <= 0x036f, // combining diacritical marks
		r >= 0xfe00 && r <= 0xfe0f, // variation selectors
		r >= 0x200b && r <= 0x200f: // zero-width spaces/marks
		return 0
	case r >= 0x1100 && r <= 0x115f, // Hangul Jamo
		r >= 0x231a && r <= 0x231b, // ⌚⌛
		r >= 0x23e9 && r <= 0x23ff, // ⏰ and clock/media symbols
		r == 0x2328,
		r >= 0x2600 && r <= 0x26ff, // Misc symbols (⚡)
		r >= 0x2700 && r <= 0x27bf, // Dingbats (✅)
		r >= 0x2b00 && r <= 0x2bff,
		r >= 0x2e80 && r <= 0x303e, // CJK radicals, Kangxi
		r >= 0x3041 && r <= 0x33ff, // Hiragana .. CJK compat
		r >= 0x3400 && r <= 0x4dbf, // CJK Ext A
		r >= 0x4e00 && r <= 0x9fff, // CJK Unified
		r >= 0xa000 && r <= 0xa4cf, // Yi
		r >= 0xac00 && r <= 0xd7a3, // Hangul syllables
		r >= 0xf900 && r <= 0xfaff, // CJK compat ideographs
		r >= 0xfe30 && r <= 0xfe4f, // CJK compat forms
		r >= 0xff00 && r <= 0xff60, // fullwidth forms
		r >= 0xffe0 && r <= 0xffe6,
		r >= 0x1f000 && r <= 0x1faff, // emoji
		r >= 0x20000 && r <= 0x3fffd: // CJK Ext B+
		return 2
	default:
		return 1
	}
}
