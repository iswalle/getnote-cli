package capabilities

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/iswalle/getnote-cli/internal/platform"
	"github.com/iswalle/getnote-cli/internal/version"
	"github.com/spf13/cobra"
)

type response struct {
	Success         bool                `json:"success"`
	CLIVersion      string              `json:"cli_version"`
	ContractVersion string              `json:"contract_version"`
	Architecture    string              `json:"architecture"`
	Commands        map[string][]string `json:"commands"`
	NoteTypes       []string            `json:"note_types"`
	Platforms       []platform.Info     `json:"platforms"`
	Install         map[string]string   `json:"install"`
	Upgrade         map[string]string   `json:"upgrade"`
}

// NewCapabilitiesCmd reports the stable execution surface exposed to an AI host.
func NewCapabilitiesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "capabilities",
		Short: "查看 CLI 能力与可接入平台 / Show CLI capabilities",
		Example: `  getnote capabilities
  getnote capabilities -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			data := response{
				Success:         true,
				CLIVersion:      version.String(),
				ContractVersion: "2.0",
				Architecture:    "Skill navigates intent; CLI performs deterministic operations",
				Commands: map[string][]string{
					"connection":     {"doctor", "capabilities", "auth", "auth login", "auth status", "auth logout", "setup"},
					"notes":          {"save", "task", "notes", "note", "note update", "note delete", "note share"},
					"search":         {"search"},
					"knowledge_base": {"kbs", "kbs-sub", "kb", "kb create", "kb add", "kb remove", "kb bloggers", "kb blogger-contents", "kb blogger-content", "kb lives", "kb live", "kb live-follow"},
					"tags":           {"tag", "tag list", "tag add", "tag remove"},
					"account":        {"quota", "version", "update"},
				},
				NoteTypes: []string{"plain_text", "link", "img_text"},
				Platforms: platform.Detect(),
				Install: map[string]string{
					"simple":   "Choose a supported AI and install the GetNote skill",
					"terminal": "npx -y @getnote/cli@latest setup",
					"fallback": "npx -y @getnote/mcp",
				},
				Upgrade: map[string]string{
					"check": "getnote update --check",
					"cli":   "getnote update",
					"npm":   "npm install -g @getnote/cli@latest",
				},
			}
			out, _ := cmd.Root().PersistentFlags().GetString("output")
			if out == "json" {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(data)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "GetNote CLI %s\n", data.CLIVersion)
			groups := make([]string, 0, len(data.Commands))
			for group := range data.Commands {
				groups = append(groups, group)
			}
			sort.Strings(groups)
			for _, group := range groups {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", group, strings.Join(data.Commands[group], ", "))
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Upgrade: getnote update")
			fmt.Fprintln(cmd.OutOrStdout(), "Detected AI platforms:")
			for _, item := range data.Platforms {
				mark := "-"
				if item.Detected {
					mark = "✓"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", mark, item.Name)
			}
			return nil
		},
	}
}
