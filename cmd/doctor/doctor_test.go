package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/iswalle/getnote-cli/internal/platform"
	"github.com/iswalle/getnote-cli/internal/version"
)

func TestTemporaryNpxPath(t *testing.T) {
	tests := map[string]bool{
		"/Users/test/.npm/_npx/abc/node_modules/@getnote/cli/bin/getnote": true,
		"/tmp/_npx/abc/getnote":                         true,
		"/opt/homebrew/bin/getnote":                     false,
		"C:/Users/test/AppData/Roaming/npm/getnote.cmd": false,
	}
	for path, want := range tests {
		if got := isTemporaryNpxPath(path); got != want {
			t.Fatalf("isTemporaryNpxPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestRequiredChecksReadyIgnoresOptionalWarnings(t *testing.T) {
	checks := []check{
		baseCheck("cli", "runtime", true, true, "cli.available", "ok", nil),
		baseCheck("gnote", "runtime", false, false, "cli.alias_missing", "optional alias missing", &action{ID: "repair"}),
	}
	if !requiredChecksReady(checks) {
		t.Fatal("optional warning must not make the connection unavailable")
	}
	if checks[1].Severity != "warning" || checks[1].Required {
		t.Fatalf("optional check = %+v", checks[1])
	}
}

func TestRequiredChecksReadyFailsForBlockingCheck(t *testing.T) {
	checks := []check{baseCheck("auth", "connection", true, false, "auth.missing", "login required", &action{ID: "login"})}
	if requiredChecksReady(checks) {
		t.Fatal("failed required check must block readiness")
	}
	issues, actions := issuesAndActions(checks)
	if len(issues) != 1 || !issues[0].Blocking || issues[0].Code != "auth.missing" {
		t.Fatalf("issues = %+v", issues)
	}
	if len(actions) != 1 || actions[0].ID != "login" {
		t.Fatalf("actions = %+v", actions)
	}
}

func TestApplySkillDirectoryStateReportsMissingAndInstalledSkills(t *testing.T) {
	root := t.TempDir()
	for _, name := range requiredSkillNames[:2] {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	partial := applySkillDirectoryState(integration{ID: "codex", Detected: true}, root)
	if partial.SkillStatus != "missing" || partial.Ready == nil || *partial.Ready || len(partial.MissingSkills) != 3 {
		t.Fatalf("partial integration = %+v", partial)
	}
	for _, name := range requiredSkillNames[2:] {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	complete := applySkillDirectoryState(integration{ID: "codex", Detected: true}, root)
	if complete.SkillStatus != "installed" || complete.Ready == nil || !*complete.Ready || len(complete.InstalledSkills) != 5 {
		t.Fatalf("complete integration = %+v", complete)
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		candidate string
		current   string
		want      bool
	}{
		{"v1.5.2", "1.5.1", true},
		{"v1.5.2", "1.5.2", false},
		{"v1.5.1", "1.5.2", false},
		{"v2.0.0", "1.99.99", true},
	}
	for _, test := range tests {
		if got := version.Compare(test.candidate, test.current) > 0; got != test.want {
			t.Fatalf("version.Compare(%q, %q) > 0 = %v, want %v", test.candidate, test.current, got, test.want)
		}
	}
}

func TestOfflineSummaryNeverClaimsRemoteReadiness(t *testing.T) {
	status, summary := readinessSummary(false, true, nil)
	if status != "partial" || summary == "" {
		t.Fatalf("status=%q summary=%q", status, summary)
	}
}

func TestResponseJSONPreservesLegacyAndMachineFields(t *testing.T) {
	ready := true
	payload, err := json.Marshal(response{
		Success: true, DiagnosticsCompleted: true, Ready: &ready, LocalReady: true, Status: "ready", Summary: "ready",
		SchemaVersion: diagnosticSchemaVersion, CLIVersion: "1.5.2", Checks: []check{}, Platforms: []platform.Info{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"success", "cli_version", "checks", "platforms", "diagnostics_completed", "ready", "local_ready", "status", "summary", "schema_version", "issues", "next_actions", "integrations", "update"} {
		if _, ok := decoded[field]; !ok {
			t.Fatalf("doctor JSON is missing %q: %s", field, payload)
		}
	}
}

func TestReadinessSummaryReportsDegradedForWarnings(t *testing.T) {
	status, _ := readinessSummary(true, false, []issue{{Code: "integration.skills_missing.codex", Severity: "warning"}})
	if status != "degraded" {
		t.Fatalf("status = %q, want degraded", status)
	}
}

func TestStatusCheckUsesFailureCode(t *testing.T) {
	got := statusCheck("auth", "connection", true, false, "auth.connected", "auth.not_connected", "missing", &action{ID: "login"})
	if got.Code != "auth.not_connected" {
		t.Fatalf("code = %q", got.Code)
	}
}
