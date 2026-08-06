package capabilities

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iswalle/getnote-cli/internal/platform"
	"github.com/iswalle/getnote-cli/internal/version"
	"github.com/spf13/cobra"
)

type response struct {
	Success      bool              `json:"success"`
	CLIVersion   string            `json:"cli_version"`
	Architecture string            `json:"architecture"`
	Commands     []string          `json:"commands"`
	NoteTypes    []string          `json:"note_types"`
	Platforms    []platform.Info   `json:"platforms"`
	Install      map[string]string `json:"install"`
}

// NewCapabilitiesCmd reports the stable execution surface exposed to an AI host.
func NewCapabilitiesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "capabilities",
		Short: "查看 CLI 能力与可接入平台 / Show CLI capabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			data := response{
				Success:      true,
				CLIVersion:   version.String(),
				Architecture: "Skill understands intent; CLI performs deterministic operations",
				Commands:     []string{"auth", "save", "notes", "note", "search", "kbs", "kb", "tag", "quota", "task"},
				NoteTypes:    []string{"plain_text", "link", "img_text"},
				Platforms:    platform.Detect(),
				Install: map[string]string{
					"simple":   "Choose a supported AI and install the GetNote skill",
					"terminal": "npx -y @getnote/cli@latest setup",
					"fallback": "npx -y @getnote/mcp",
				},
			}
			out, _ := cmd.Root().PersistentFlags().GetString("output")
			if out == "json" {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(data)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "GetNote CLI %s\n", data.CLIVersion)
			fmt.Fprintln(cmd.OutOrStdout(), "Commands: "+strings.Join(data.Commands, ", "))
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
