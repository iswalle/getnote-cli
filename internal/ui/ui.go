// Package ui provides shared output helpers for the getnote CLI.
package ui

import (
	"fmt"
	"strings"
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

// Truncate shortens s to fit within max display columns (CJK = 2 cols each).
func Truncate(s string, max int) string {
	runes := []rune(s)
	width := 0
	for i, r := range runes {
		w := 1
		if r > 0x2E7F {
			w = 2
		}
		if width+w > max {
			return string(runes[:i]) + "…"
		}
		width += w
	}
	return s
}

// NoteID returns the canonical note ID string from note_id or id fields.
func NoteID(noteID, id fmt.Stringer) string {
	if s := noteID.String(); s != "" && s != "0" {
		return s
	}
	return id.String()
}
