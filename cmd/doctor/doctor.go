package doctor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/config"
	"github.com/iswalle/getnote-cli/internal/platform"
	"github.com/iswalle/getnote-cli/internal/version"
	"github.com/spf13/cobra"
)

const diagnosticSchemaVersion = "2.0"

var requiredSkillNames = []string{"getnote-auth", "getnote-kb", "getnote-note", "getnote-search", "getnote-tag"}

type action struct {
	ID                   string `json:"id"`
	Description          string `json:"description"`
	Command              string `json:"command,omitempty"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
}

type check struct {
	Name     string         `json:"name"`
	OK       bool           `json:"ok"`
	Message  string         `json:"message"`
	Category string         `json:"category"`
	Severity string         `json:"severity"`
	Required bool           `json:"required"`
	Code     string         `json:"code"`
	Duration int64          `json:"duration_ms,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
	Fix      *action        `json:"fix,omitempty"`
}

type issue struct {
	Code     string  `json:"code"`
	Severity string  `json:"severity"`
	Blocking bool    `json:"blocking"`
	Message  string  `json:"message"`
	Fix      *action `json:"fix,omitempty"`
}

type updateInfo struct {
	Checked         bool    `json:"checked"`
	Current         string  `json:"current"`
	Latest          string  `json:"latest,omitempty"`
	UpdateAvailable *bool   `json:"update_available,omitempty"`
	Message         string  `json:"message"`
	Fix             *action `json:"fix,omitempty"`
}

type integration struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Detected        bool           `json:"detected"`
	Executable      string         `json:"executable,omitempty"`
	AppPath         string         `json:"app_path,omitempty"`
	DetectionSource []string       `json:"detection_sources,omitempty"`
	SkillStatus     string         `json:"skill_status"`
	Ready           *bool          `json:"ready,omitempty"`
	InstalledSkills []string       `json:"installed_skills,omitempty"`
	MissingSkills   []string       `json:"missing_skills,omitempty"`
	SkillPath       string         `json:"skill_path,omitempty"`
	Message         string         `json:"message"`
	Fix             *action        `json:"fix,omitempty"`
	Details         map[string]any `json:"details,omitempty"`
}

type response struct {
	Success              bool            `json:"success"`
	DiagnosticsCompleted bool            `json:"diagnostics_completed"`
	Ready                *bool           `json:"ready"`
	LocalReady           bool            `json:"local_ready"`
	Status               string          `json:"status"`
	Summary              string          `json:"summary"`
	SchemaVersion        string          `json:"schema_version"`
	Mode                 string          `json:"mode"`
	CLIVersion           string          `json:"cli_version"`
	OS                   string          `json:"os"`
	Arch                 string          `json:"arch"`
	Checks               []check         `json:"checks"`
	Issues               []issue         `json:"issues"`
	NextActions          []action        `json:"next_actions"`
	Update               updateInfo      `json:"update"`
	Integrations         []integration   `json:"integrations"`
	Platforms            []platform.Info `json:"platforms"`
}

// NewDoctorCmd validates installation, login state, API connectivity and AI host integration state.
// It never prints credentials or note contents.
func NewDoctorCmd() *cobra.Command {
	var offline, skipUpdate, allPlatforms bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "检查安装、登录、服务连通性和 AI 接入状态 / Diagnose the complete local setup",
		Example: `  getnote doctor
  getnote doctor --offline
  getnote doctor --all-platforms -o json
  getnote doctor -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			data := collectDiagnostics(offline, skipUpdate || offline, allPlatforms)
			out, _ := cmd.Root().PersistentFlags().GetString("output")
			if out == "json" {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(data); err != nil {
					return err
				}
				if data.Ready != nil && !*data.Ready {
					return doctorExitError{}
				}
				return nil
			}
			writeHuman(cmd, data)
			if data.Ready != nil && !*data.Ready {
				return fmt.Errorf("environment is not ready; follow the remediation steps above")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&offline, "offline", false, "跳过联网检查；结果只能证明本地环境 / Skip network checks")
	cmd.Flags().BoolVar(&skipUpdate, "skip-update", false, "跳过 GitHub 版本检查 / Skip the GitHub release check")
	cmd.Flags().BoolVar(&allPlatforms, "all-platforms", false, "显示未检测到的平台 / Include undetected platforms")
	return cmd
}

type doctorExitError struct{}

func (doctorExitError) Error() string  { return "environment is not ready" }
func (doctorExitError) Rendered() bool { return true }

func collectDiagnostics(offline, skipUpdate, allPlatforms bool) response {
	cfg := config.Get()
	checks := []check{
		baseCheck("cli", "runtime", true, version.String() != "", "cli.available", "CLI is executable", nil),
		commandPathCheck("getnote", true),
		commandPathCheck("gnote", false),
		npmPackageVersionCheck(),
		commandAvailabilityCheck("node", "Node.js is available for dependency installation"),
		commandAvailabilityCheck("npx", "npx is available for one-command setup"),
		statusCheck("auth", "connection", true, cfg.IsLoggedIn(), "auth.connected", "auth.not_connected", authMessage(cfg.IsLoggedIn()), &action{ID: "login", Description: "Open the browser and authorize this GetNote account", Command: "getnote auth login", RequiresConfirmation: true}),
	}
	if offline {
		checks = append(checks, check{Name: "api", OK: false, Message: "Skipped by --offline; remote readiness is unknown", Category: "connection", Severity: "warning", Required: true, Code: "api.skipped"})
	} else if cfg.IsLoggedIn() {
		started := time.Now()
		_, err := client.New("").NoteList(client.NoteListParams{Limit: 1})
		apiCheck := baseCheck("api", "connection", true, err == nil, "api.reachable", apiMessage(err), &action{ID: "retry_diagnostics", Description: "Retry the authenticated OpenAPI check", Command: "getnote doctor -o json", RequiresConfirmation: false})
		apiCheck.Duration = time.Since(started).Milliseconds()
		if err != nil {
			apiCheck.Code = "api.request_failed"
			var requestErr *client.RequestError
			if errors.As(err, &requestErr) {
				apiCheck.Details = map[string]any{"error_code": requestErr.Code, "reason": requestErr.Reason, "retryable": requestErr.Retryable, "request_id": requestErr.RequestID, "http_status": requestErr.StatusCode}
			}
		}
		checks = append(checks, apiCheck)
	} else {
		checks = append(checks, check{Name: "api", OK: false, Message: "Not checked because no GetNote account is connected", Category: "connection", Severity: "error", Required: true, Code: "api.blocked_by_auth", Fix: &action{ID: "login", Description: "Authorize before checking the API", Command: "getnote auth login", RequiresConfirmation: true}})
	}

	localReady := requiredChecksReadyExcept(checks, "api")
	remoteReady := requiredChecksReady(checks) && !offline
	issues, actions := issuesAndActions(checks)
	update := collectUpdate(skipUpdate)
	if update.UpdateAvailable != nil && *update.UpdateAvailable {
		issues = append(issues, issue{Code: "update.available", Severity: "warning", Blocking: false, Message: update.Message, Fix: update.Fix})
		actions = appendUniqueAction(actions, update.Fix)
	}
	rawPlatforms := platform.Detect()
	integrations := diagnoseIntegrations(rawPlatforms, allPlatforms)
	for _, item := range integrations {
		if item.Detected && item.SkillStatus == "missing" {
			issues = append(issues, issue{Code: "integration.skills_missing." + item.ID, Severity: "warning", Blocking: false, Message: item.Message, Fix: item.Fix})
			actions = appendUniqueAction(actions, item.Fix)
		} else if item.Detected && item.SkillStatus == "unverified" {
			issues = append(issues, issue{Code: "integration.skills_unverified." + item.ID, Severity: "warning", Blocking: false, Message: item.Message, Fix: item.Fix})
			actions = appendUniqueAction(actions, item.Fix)
		}
	}
	status, summary := readinessSummary(remoteReady, offline, issues)
	mode := "online"
	if offline {
		mode = "offline"
	}
	responseValue := response{
		Success: localReady && (offline || remoteReady), DiagnosticsCompleted: true, LocalReady: localReady, Status: status, Summary: summary,
		SchemaVersion: diagnosticSchemaVersion, Mode: mode, CLIVersion: version.String(), OS: runtime.GOOS, Arch: runtime.GOARCH,
		Checks: checks, Issues: issues, NextActions: actions, Update: update, Integrations: integrations, Platforms: rawPlatforms,
	}
	if !offline {
		result := remoteReady
		responseValue.Ready = &result
	}
	return responseValue
}

func statusCheck(name, category string, required, ok bool, successCode, failureCode, message string, fix *action) check {
	code := successCode
	if !ok {
		code = failureCode
	}
	return baseCheck(name, category, required, ok, code, message, fix)
}

func baseCheck(name, category string, required, ok bool, code, message string, fix *action) check {
	severity := "info"
	if !ok && required {
		severity = "error"
	} else if !ok {
		severity = "warning"
	}
	if ok {
		fix = nil
	}
	return check{Name: name, OK: ok, Message: message, Category: category, Severity: severity, Required: required, Code: code, Fix: fix}
}

func commandAvailabilityCheck(name, message string) check {
	_, err := exec.LookPath(name)
	if err == nil {
		return baseCheck(name, "installation", false, true, name+".available", message, nil)
	}
	return baseCheck(name, "installation", false, false, name+".missing", name+" is unavailable; existing CLI commands still work, but automated setup cannot repair or install Skills", &action{ID: "install_node", Description: "Install Node.js 20+ including npm and npx", Command: "https://nodejs.org/", RequiresConfirmation: true})
}

func requiredChecksReady(checks []check) bool {
	for _, item := range checks {
		if item.Required && !item.OK {
			return false
		}
	}
	return true
}

func requiredChecksReadyExcept(checks []check, excludedName string) bool {
	for _, item := range checks {
		if item.Name != excludedName && item.Required && !item.OK {
			return false
		}
	}
	return true
}

func readinessSummary(ready, offline bool, issues []issue) (string, string) {
	if offline {
		return "partial", "Local diagnostics completed, but API connectivity was skipped; remote readiness is unknown"
	}
	if !ready {
		for _, item := range issues {
			if item.Blocking {
				return "not_ready", "GetNote is not ready: " + item.Message
			}
		}
		return "not_ready", "GetNote is not ready"
	}
	if len(issues) > 0 {
		return "degraded", "Core GetNote access is ready, with non-blocking issues that should be repaired"
	}
	return "ready", "GetNote CLI, account authorization, OpenAPI connectivity and detected integrations are ready"
}

func issuesAndActions(checks []check) ([]issue, []action) {
	issues := []issue{}
	actions := []action{}
	for _, item := range checks {
		if item.OK {
			continue
		}
		issues = append(issues, issue{Code: item.Code, Severity: item.Severity, Blocking: item.Required, Message: item.Message, Fix: item.Fix})
		actions = appendUniqueAction(actions, item.Fix)
	}
	return issues, actions
}

func appendUniqueAction(actions []action, candidate *action) []action {
	if candidate == nil {
		return actions
	}
	for _, existing := range actions {
		if existing.ID == candidate.ID {
			return actions
		}
	}
	return append(actions, *candidate)
}

func collectUpdate(skip bool) updateInfo {
	result := updateInfo{Current: version.String()}
	if skip || version.Version == "dev" {
		result.Message = "Version check skipped"
		return result
	}
	result.Checked = true
	latest := version.LatestRelease()
	if latest == "" {
		result.Message = "Could not query the latest GitHub Release; this does not block GetNote"
		return result
	}
	latest = strings.TrimPrefix(latest, "v")
	result.Latest = latest
	available := version.Compare(latest, version.String()) > 0
	result.UpdateAvailable = &available
	if available {
		result.Message = fmt.Sprintf("CLI update available: %s -> %s", version.String(), latest)
		result.Fix = &action{ID: "update_cli", Description: "Upgrade the CLI and rerun diagnostics", Command: "getnote update && getnote doctor -o json", RequiresConfirmation: true}
	} else {
		result.Message = "CLI is up to date"
	}
	return result
}

func diagnoseIntegrations(raw []platform.Info, includeAll bool) []integration {
	home, _ := os.UserHomeDir()
	result := []integration{}
	for _, item := range raw {
		if !item.Detected && !includeAll {
			continue
		}
		entry := integration{ID: item.ID, Name: item.Name, Detected: item.Detected, Executable: item.Executable, AppPath: item.AppPath, Details: map[string]any{"detection_only": true}}
		if item.Executable != "" {
			entry.DetectionSource = append(entry.DetectionSource, "executable")
		}
		if item.AppPath != "" {
			entry.DetectionSource = append(entry.DetectionSource, "application")
		}
		if !item.Detected {
			entry.SkillStatus = "not_checked"
			entry.Message = "Platform was not detected on this machine"
			result = append(result, entry)
			continue
		}
		switch item.ID {
		case "workbuddy":
			entry = applySkillDirectoryState(entry, filepath.Join(home, ".workbuddy", "skills"))
			if entry.SkillStatus == "missing" {
				entry.Fix = setupAction(item.ID)
			}
		case "codex", "claude-code", "cursor":
			entry = applySkillDirectoryState(entry, filepath.Join(home, ".agents", "skills"))
			if entry.SkillStatus == "missing" {
				entry.Fix = setupAction(item.ID)
			}
		case "qclaw", "openclaw":
			entry.SkillStatus = "unverified"
			entry.Message = "Platform detected, but this CLI cannot safely inspect its platform-managed Skill registry; verify the GetNote Skill inside the platform"
			entry.Fix = &action{ID: "verify_platform_skill_" + item.ID, Description: "Open the platform Skill manager and verify that GetNote is installed and enabled", RequiresConfirmation: false}
		default:
			entry.SkillStatus = "unverified"
			entry.Message = "Platform detected; Skill installation state is unknown"
		}
		result = append(result, entry)
	}
	return result
}

func applySkillDirectoryState(entry integration, root string) integration {
	entry.SkillPath = root
	ready := true
	for _, name := range requiredSkillNames {
		if _, err := os.Stat(filepath.Join(root, name, "SKILL.md")); err == nil {
			entry.InstalledSkills = append(entry.InstalledSkills, name)
		} else {
			entry.MissingSkills = append(entry.MissingSkills, name)
			ready = false
		}
	}
	entry.Ready = &ready
	if ready {
		entry.SkillStatus = "installed"
		entry.Message = fmt.Sprintf("All %d GetNote Skills are installed", len(requiredSkillNames))
		entry.Fix = nil
	} else {
		entry.SkillStatus = "missing"
		entry.Message = fmt.Sprintf("Platform detected, but %d of %d GetNote Skills are missing", len(entry.MissingSkills), len(requiredSkillNames))
	}
	return entry
}

func setupAction(target string) *action {
	return &action{ID: "setup_" + target, Description: "Install or repair the CLI and GetNote Skills, then restart the AI host", Command: "npx -y @getnote/cli@latest setup", RequiresConfirmation: true}
}

func writeHuman(cmd *cobra.Command, data response) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", strings.ToUpper(data.Status), data.Summary)
	for _, item := range data.Checks {
		mark := "✓"
		if !item.OK {
			mark = "!"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s %-12s [%s] %s\n", mark, item.Name, item.Severity, item.Message)
	}
	if len(data.Integrations) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "AI integrations:")
		for _, item := range data.Integrations {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %-12s %-10s %s\n", item.Name, item.SkillStatus, item.Message)
		}
	}
	if len(data.NextActions) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Next actions:")
		for _, next := range data.NextActions {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s", next.Description)
			if next.Command != "" {
				fmt.Fprintf(cmd.OutOrStdout(), ": %s", next.Command)
			}
			fmt.Fprintln(cmd.OutOrStdout())
		}
	}
}

type npmPackage struct {
	Version string `json:"version"`
}

func commandPathCheck(name string, required bool) check {
	path, err := exec.LookPath(name)
	fix := &action{ID: "repair_cli", Description: "Install or repair the stable GetNote CLI", Command: "npx -y @getnote/cli@latest setup", RequiresConfirmation: true}
	if err != nil {
		if !required {
			return baseCheck(name, "runtime", false, false, "cli.alias_missing", name+" alias is missing; use getnote or rerun setup to restore the alias", fix)
		}
		return baseCheck(name, "runtime", true, false, "cli.command_missing", name+" command is missing", fix)
	}
	resolved := path
	if realPath, err := filepath.EvalSymlinks(path); err == nil {
		resolved = realPath
	}
	if isTemporaryNpxPath(path) || isTemporaryNpxPath(resolved) {
		return baseCheck(name, "runtime", required, false, "cli.disposable_path", name+" points to a disposable npx cache", fix)
	}
	item := baseCheck(name, "runtime", required, true, "cli.stable_path", path, nil)
	item.Details = map[string]any{"path": path, "resolved_path": resolved}
	return item
}

func isTemporaryNpxPath(path string) bool {
	normalized := filepath.ToSlash(path)
	return strings.Contains(normalized, "/.npm/_npx/") || strings.Contains(normalized, "/_npx/")
}

func npmGlobalPackageDir() string {
	out, err := exec.Command("npm", "root", "-g").Output()
	if err != nil {
		return ""
	}
	return filepath.Join(strings.TrimSpace(string(out)), "@getnote", "cli")
}

func npmPackageVersionCheck() check {
	path := filepath.Join(npmGlobalPackageDir(), "package.json")
	fix := &action{ID: "repair_npm_package", Description: "Reinstall the matching global npm package and Skills", Command: "npx -y @getnote/cli@latest setup", RequiresConfirmation: true}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return baseCheck("npm_package", "installation", false, true, "npm.optional", "not installed through npm (optional)", nil)
	}
	if err != nil {
		return baseCheck("npm_package", "installation", false, false, "npm.metadata_unreadable", "cannot read global package metadata", fix)
	}
	var pkg npmPackage
	if err := json.Unmarshal(raw, &pkg); err != nil || strings.TrimSpace(pkg.Version) == "" {
		return baseCheck("npm_package", "installation", false, false, "npm.metadata_invalid", "invalid global package metadata", fix)
	}
	want := strings.TrimPrefix(version.String(), "v")
	if want != "dev" && pkg.Version != want {
		item := baseCheck("npm_package", "installation", false, false, "npm.version_mismatch", fmt.Sprintf("global npm package is v%s but running CLI is v%s", pkg.Version, want), fix)
		item.Details = map[string]any{"package_version": pkg.Version, "cli_version": want, "package_path": path}
		return item
	}
	item := baseCheck("npm_package", "installation", false, true, "npm.aligned", "global npm package v"+pkg.Version, nil)
	item.Details = map[string]any{"package_version": pkg.Version, "package_path": path}
	return item
}

func authMessage(ok bool) string {
	if ok {
		return "GetNote account is connected"
	}
	return "No GetNote account is connected"
}

func apiMessage(err error) string {
	if err == nil {
		return "OpenAPI request succeeded"
	}
	return "OpenAPI request failed: " + err.Error()
}
