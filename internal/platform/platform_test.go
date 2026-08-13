package platform

import "testing"

func TestDetectReturnsStablePlatformIDs(t *testing.T) {
	got := Detect()
	want := map[string]bool{
		"workbuddy":   false,
		"qclaw":       false,
		"codex":       false,
		"claude-code": false,
		"cursor":      false,
		"openclaw":    false,
	}
	for _, item := range got {
		if _, ok := want[item.ID]; !ok {
			t.Fatalf("unexpected platform id %q", item.ID)
		}
		want[item.ID] = true
	}
	for id, seen := range want {
		if !seen {
			t.Fatalf("missing platform id %q", id)
		}
	}
}
