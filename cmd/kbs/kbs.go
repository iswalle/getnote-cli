package kbs

import (
	"encoding/json"
	"fmt"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewKbsCmd returns the top-level kbs (list) command.
func NewKbsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "kbs",
		Short: "List all knowledge bases",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))
			resp, err := c.KBList()
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"ID", "Name", "Description", "Notes"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			for _, t := range resp.Data.Topics {
				table.Append([]string{
					t.TopicID,
					t.Name,
					t.Description,
					fmt.Sprintf("%d", t.Stats.NoteCount),
				})
			}
			table.Render()
			return nil
		},
	}
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}

func envTarget(cmd *cobra.Command) string {
	e, _ := cmd.Root().PersistentFlags().GetString("env")
	return e
}
