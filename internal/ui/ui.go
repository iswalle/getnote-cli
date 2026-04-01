// Package ui provides shared output helpers for the getnote CLI.
package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// FriendlyError extracts a human-readable message from raw API error strings.
// If the error contains a JSON "message" field, that value is returned.
// Otherwise the original error is returned unchanged.
func FriendlyError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if idx := strings.Index(msg, `"message":"`); idx != -1 {
		start := idx + len(`"message":"`)
		end := strings.Index(msg[start:], `"`)
		if end > 0 {
			return fmt.Errorf("%s", msg[start:start+end])
		}
	}
	return err
}

// DisplayWidth returns the terminal display width of s.
// ASCII = 1 col, CJK/fullwidth = 2 cols.
func DisplayWidth(s string) int {
	w := 0
	for _, r := range s {
		if isWide(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// isWide reports whether r is a wide (double-column) rune.
func isWide(r rune) bool {
	return r >= 0x1100 && (
		r <= 0x115F || // Hangul Jamo
		r == 0x2329 || r == 0x232A ||
		(r >= 0x2E80 && r <= 0x303E) || // CJK Radicals
		(r >= 0x3040 && r <= 0x33FF) || // Japanese
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x4E00 && r <= 0xA4CF) || // CJK Unified
		(r >= 0xA960 && r <= 0xA97F) ||
		(r >= 0xAC00 && r <= 0xD7FF) || // Hangul Syllables
		(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility
		(r >= 0xFE10 && r <= 0xFE19) ||
		(r >= 0xFE30 && r <= 0xFE6F) ||
		(r >= 0xFF00 && r <= 0xFF60) || // Fullwidth
		(r >= 0xFFE0 && r <= 0xFFE6) ||
		(r >= 0x1B000 && r <= 0x1B001) ||
		(r >= 0x1F300 && r <= 0x1F64F) || // Emoji
		(r >= 0x1F900 && r <= 0x1FAFF) ||
		(r >= 0x20000 && r <= 0x2FFFD) ||
		(r >= 0x30000 && r <= 0x3FFFD))
}

// Truncate shortens s to fit within max display columns, appending "…".
func Truncate(s string, max int) string {
	width := 0
	for i, r := range s {
		w := 1
		if isWide(r) {
			w = 2
		}
		if width+w > max {
			return s[:i] + "…"
		}
		width += w
	}
	return s
}

// PadRight pads s with spaces on the right so its display width equals width.
// If s is already wider than width, it is returned as-is (no truncation).
func PadRight(s string, width int) string {
	dw := DisplayWidth(s)
	if dw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-dw)
}

// NoteID returns the canonical note ID string from note_id or id fields.
func NoteID(noteID, id fmt.Stringer) string {
	if s := noteID.String(); s != "" && s != "0" {
		return s
	}
	return id.String()
}

// Col formats s into a fixed display-width column followed by sep.
// Unlike %-*s, this accounts for wide CJK characters.
func Col(s string, width int, sep string) string {
	_ = utf8.RuneCountInString // import guard
	return PadRight(Truncate(s, width), width) + sep
}

// PrintRow writes one table row to buf (returns the formatted line).
// cols is a slice of (value, width) pairs; sep is the column separator.
func PrintRow(cols []ColSpec, sep string) string {
	var sb strings.Builder
	for i, c := range cols {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(PadRight(Truncate(c.Value, c.Width), c.Width))
	}
	sb.WriteString("\n")
	return sb.String()
}

// ColSpec holds a column value and its fixed display width.
type ColSpec struct {
	Value string
	Width int
}

// PrintHeader writes the header row matching PrintRow column widths.
func PrintHeader(labels []ColSpec, sep string) string {
	values := make([]ColSpec, len(labels))
	copy(values, labels)
	return PrintRow(values, sep)
}

// DividerLine returns a "─" divider spanning total display width.
func DividerLine(cols []ColSpec, sep string) string {
	total := 0
	for _, c := range cols {
		total += c.Width
	}
	total += len(sep) * (len(cols) - 1)
	return strings.Repeat("─", total) + "\n"
}
