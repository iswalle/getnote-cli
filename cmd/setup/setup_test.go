package setup

import (
	"bytes"
	"os/exec"
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

func TestJSONSetupKeepsDependencyProgressOffStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := exec.Command("true")
	configureInstallProcess(command, "json", &stdout, &stderr)
	if command.Stdout != &stderr || command.Stderr != &stderr {
		t.Fatalf("json setup must route installer output to stderr: stdout=%T stderr=%T", command.Stdout, command.Stderr)
	}

	command = exec.Command("true")
	configureInstallProcess(command, "table", &stdout, &stderr)
	if command.Stdout != &stdout || command.Stderr != &stderr {
		t.Fatalf("table setup output routing changed unexpectedly")
	}
}

func TestResolveTargetsRejectsUnsupportedHost(t *testing.T) {
	if _, err := resolveTargets([]string{"workbuddy"}); err == nil {
		t.Fatal("resolveTargets() should reject hosts that require marketplace installation")
	}
}
