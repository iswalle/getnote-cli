package kbssub

import (
	"encoding/json"
	"fmt"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewKbsSubCmd returns the kbs-sub command for listing subscribed knowledge bases.
func NewKbsSubCmd() *cobra.Command {
	var page int

	cmd := &cobra.Command{
		Use:     "kbs-sub",
		Aliases: []string{"subscribed-kbs"},
		Short:   "列出订阅的知识库 / List subscribed knowledge bases",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
			resp, err := c.KBSubscribedList(page)
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

	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	return cmd
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}
