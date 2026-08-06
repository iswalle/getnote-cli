package setup

import (
	"reflect"
	"testing"
)

func TestResolveTargets(t *testing.T) {
	got, err := resolveTargets([]string{"cursor,codex", "claude-code", "codex"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"claude-code", "codex", "cursor"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveTargets() = %v, want %v", got, want)
	}
}

func TestResolveTargetsRejectsUnsupportedHost(t *testing.T) {
	if _, err := resolveTargets([]string{"workbuddy"}); err == nil {
		t.Fatal("resolveTargets() should reject hosts that require marketplace installation")
	}
}
