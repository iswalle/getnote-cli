package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/config"
	"github.com/iswalle/getnote-cli/internal/platform"
	"github.com/iswalle/getnote-cli/internal/version"
	"github.com/spf13/cobra"
)

type check struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type response struct {
	Success    bool            `json:"success"`
	CLIVersion string          `json:"cli_version"`
	OS         string          `json:"os"`
	Arch       string          `json:"arch"`
	Checks     []check         `json:"checks"`
	Platforms  []platform.Info `json:"platforms"`
}

// NewDoctorCmd validates installation, login state and API connectivity without printing credentials or note data.
func NewDoctorCmd() *cobra.Command {
	var offline bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "检查安装、登录和服务连通性 / Diagnose the local setup",
		Example: `  getnote doctor
  getnote doctor --offline
  getnote doctor -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Get()
			checks := []check{
				{Name: "cli", OK: version.String() != "", Message: "CLI is executable"},
				commandPathCheck("getnote", true),
				commandPathCheck("gnote", npmGlobalPackageInstalled()),
				npmPackageVersionCheck(),
				{Name: "node", OK: hasCommand("node"), Message: "Node.js is available for skill installation"},
				{Name: "npx", OK: hasCommand("npx"), Message: "npx is available for skill installation"},
				{Name: "auth", OK: cfg.IsLoggedIn(), Message: authMessage(cfg.IsLoggedIn())},
			}
			if !offline && cfg.IsLoggedIn() {
				_, err := client.New("").NoteList(client.NoteListParams{Limit: 1})
				checks = append(checks, check{Name: "api", OK: err == nil, Message: apiMessage(err)})
			}
			ok := true
			for _, item := range checks {
				if !item.OK && item.Name != "node" && item.Name != "npx" {
					ok = false
				}
			}
			data := response{Success: ok, CLIVersion: version.String(), OS: runtime.GOOS, Arch: runtime.GOARCH, Checks: checks, Platforms: platform.Detect()}
			out, _ := cmd.Root().PersistentFlags().GetString("output")
			if out == "json" {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(data)
			}
			for _, item := range checks {
				mark := "✗"
				if item.OK {
					mark = "✓"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %-8s %s\n", mark, item.Name, item.Message)
			}
			if !ok {
				return fmt.Errorf("environment check failed")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&offline, "offline", false, "跳过 API 连通性检查 / Skip API connectivity check")
	return cmd
}

func hasCommand(name string) bool { _, err := exec.LookPath(name); return err == nil }

type npmPackage struct {
	Version string `json:"version"`
}

func commandPathCheck(name string, required bool) check {
	path, err := exec.LookPath(name)
	if err != nil {
		if required {
			return check{Name: name, OK: false, Message: name + " command is missing; run: npm install -g @getnote/cli@latest"}
		}
		return check{Name: name, OK: true, Message: name + " alias is optional for direct binary installation"}
	}
	resolved := path
	if realPath, err := filepath.EvalSymlinks(path); err == nil {
		resolved = realPath
	}
	if isTemporaryNpxPath(path) || isTemporaryNpxPath(resolved) {
		return check{Name: name, OK: false, Message: name + " points to a disposable npx cache; rerun: npx -y @getnote/cli@latest setup"}
	}
	return check{Name: name, OK: true, Message: path}
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

func npmGlobalPackageInstalled() bool {
	_, err := os.Stat(filepath.Join(npmGlobalPackageDir(), "package.json"))
	return err == nil
}

func npmPackageVersionCheck() check {
	path := filepath.Join(npmGlobalPackageDir(), "package.json")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return check{Name: "npm_package", OK: true, Message: "not installed through npm (optional)"}
	}
	if err != nil {
		return check{Name: "npm_package", OK: false, Message: "cannot read global package metadata"}
	}
	var pkg npmPackage
	if err := json.Unmarshal(raw, &pkg); err != nil || strings.TrimSpace(pkg.Version) == "" {
		return check{Name: "npm_package", OK: false, Message: "invalid global package metadata"}
	}
	want := strings.TrimPrefix(version.String(), "v")
	if want != "dev" && pkg.Version != want {
		return check{Name: "npm_package", OK: false, Message: fmt.Sprintf("global npm package is v%s but running CLI is v%s; run setup again", pkg.Version, want)}
	}
	return check{Name: "npm_package", OK: true, Message: "global npm package v" + pkg.Version}
}
func authMessage(ok bool) string {
	if ok {
		return "GetNote account is connected"
	}
	return "Run: getnote auth login"
}
func apiMessage(err error) string {
	if err == nil {
		return "OpenAPI request succeeded"
	}
	return "OpenAPI request failed: " + err.Error()
}
