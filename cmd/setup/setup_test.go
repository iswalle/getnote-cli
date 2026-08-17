package setup

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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
	if _, err := resolveTargets([]string{"unknown-host"}); err == nil {
		t.Fatal("resolveTargets() should reject unsupported hosts")
	}
}

func TestResolveTargetsIncludesMarketplaceManagedHosts(t *testing.T) {
	got, err := resolveTargets([]string{"qclaw,openclaw"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"openclaw", "qclaw"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveTargets() = %v, want %v", got, want)
	}
}

func TestSetupPlatformResultsExplainEveryInstallMethod(t *testing.T) {
	platforms, actions := setupPlatformResults([]string{"codex", "qclaw", "workbuddy"}, true)
	if len(platforms) != 3 || len(actions) != 1 {
		t.Fatalf("platforms=%+v actions=%+v", platforms, actions)
	}
	if platforms[0].Status != "installed" || !platforms[0].SkillsInstalled {
		t.Fatalf("codex=%+v", platforms[0])
	}
	if platforms[1].Status != "verify_in_platform" || actions[0].URL != clawHubURL {
		t.Fatalf("qclaw=%+v actions=%+v", platforms[1], actions)
	}
	if !platforms[2].RestartRequired {
		t.Fatalf("workbuddy=%+v", platforms[2])
	}
}

func TestDryRunDoesNotClaimSkillsAreInstalled(t *testing.T) {
	platforms, _ := setupPlatformResults([]string{"codex", "workbuddy"}, false)
	for _, item := range platforms {
		if item.Status == "installed" || item.SkillsInstalled {
			t.Fatalf("dry-run platform=%+v", item)
		}
	}
}

func TestHumanResultHidesInstallerInternals(t *testing.T) {
	command := &cobra.Command{}
	var output bytes.Buffer
	command.SetOut(&output)
	platforms, actions := setupPlatformResults([]string{"codex", "qclaw", "workbuddy"}, true)
	err := writeResult(command, "table", result{InstalledCLI: true, Authenticated: true, Platforms: platforms, NextActions: actions, Next: "可以开始使用"})
	if err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{"安装完成", "Codex", "QClaw", "WorkBuddy", clawHubURL, "可以开始使用"} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
	for _, internal := range []string{"universal:", "symlink", "overwrites:"} {
		if strings.Contains(text, internal) {
			t.Fatalf("output exposes %q:\n%s", internal, text)
		}
	}
}

func TestResolveTargetsSupportsWorkBuddy(t *testing.T) {
	got, err := resolveTargets([]string{"workbuddy,codex"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"codex", "workbuddy"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveTargets() = %v, want %v", got, want)
	}
}

func TestInstallWorkBuddySkillsCopiesOnlyBundledSkills(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	unrelated := filepath.Join(target, "my-existing-skill")
	if err := os.MkdirAll(unrelated, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unrelated, "SKILL.md"), []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range workBuddySkillNames {
		dir := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "references", "commands.md"), []byte("commands"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := installWorkBuddySkills(source, target); err != nil {
		t.Fatal(err)
	}
	for _, name := range workBuddySkillNames {
		for _, file := range []string{"SKILL.md", filepath.Join("references", "commands.md")} {
			if _, err := os.Stat(filepath.Join(target, name, file)); err != nil {
				t.Fatalf("missing installed file %s/%s: %v", name, file, err)
			}
		}
	}
	if raw, err := os.ReadFile(filepath.Join(unrelated, "SKILL.md")); err != nil || string(raw) != "keep me" {
		t.Fatalf("unrelated WorkBuddy skill was changed: content=%q err=%v", raw, err)
	}
}

func TestWorkBuddyInstallDoesNotTouchCredentials(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	credentialPath := filepath.Join(home, ".getnote", "config.json")
	if err := os.MkdirAll(filepath.Dir(credentialPath), 0o700); err != nil {
		t.Fatal(err)
	}
	want := []byte("{\n  \"api_key\": \"gk_test_keep_me\",\n  \"client_id\": \"cli_keep_me\"\n}\n")
	if err := os.WriteFile(credentialPath, want, 0o600); err != nil {
		t.Fatal(err)
	}

	source := t.TempDir()
	target := filepath.Join(home, ".workbuddy", "skills")
	for _, name := range workBuddySkillNames {
		dir := filepath.Join(source, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := installWorkBuddySkills(source, target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(credentialPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("credential changed during repeated install:\n got %s\nwant %s", got, want)
	}
	info, err := os.Stat(credentialPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("credential permissions changed: %o", info.Mode().Perm())
	}
}

func TestWorkBuddyInstallValidatesWholeBundleBeforeChangingTargets(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	for _, name := range workBuddySkillNames[:len(workBuddySkillNames)-1] {
		dir := filepath.Join(source, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("new"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range workBuddySkillNames {
		dir := filepath.Join(target, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := installWorkBuddySkills(source, target); err == nil {
		t.Fatal("incomplete source must fail")
	}
	for _, name := range workBuddySkillNames {
		raw, err := os.ReadFile(filepath.Join(target, name, "SKILL.md"))
		if err != nil || string(raw) != "old" {
			t.Fatalf("%s changed after failed install: %q %v", name, raw, err)
		}
	}
}
