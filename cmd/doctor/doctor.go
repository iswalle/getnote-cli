package doctor

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

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
