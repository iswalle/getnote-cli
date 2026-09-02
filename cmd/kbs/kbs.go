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
	var page int
	var scope string
	cmd := &cobra.Command{
		Use:   "kbs",
		Args:  cobra.NoArgs,
		Short: "列出所有知识库 / List all knowledge bases",
		Example: `  getnote kbs
  getnote kbs --scope BOOKSPACE
  getnote kbs --scope DEFAULT --page 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
			resp, err := c.KBList(page, scope)
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
	cmd.Flags().IntVar(&page, "page", 1, "页码 / Page number")
	cmd.Flags().StringVar(&scope, "scope", "DEFAULT", "知识库类型 / Scope: DEFAULT, CUSTOMER, BOOKSPACE, TEAMSPACE")
	return cmd
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}
