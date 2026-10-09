package kbssub

import (
	"encoding/json"
	"fmt"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/discoveryflags"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewKbsSubCmd returns the kbs-sub command for listing subscribed knowledge bases.
func NewKbsSubCmd() *cobra.Command {
	var page int
	var scope string
	var options client.KnowledgeLookupOptions
	var all bool

	cmd := &cobra.Command{
		Use:     "kbs-sub",
		Args:    cobra.NoArgs,
		Aliases: []string{"subscribed-kbs"},
		Short:   "列出订阅的知识库 / List subscribed knowledge bases",
		Example: `  getnote kbs-sub
  getnote kbs-sub --scope BOOKSPACE
  getnote kbs-sub --scope DEFAULT --page 2 -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
			options.Page, options.Scope = page, scope
			if err := discoveryflags.ValidateLookupFlags(cmd, options); err != nil {
				return err
			}
			var resp *client.KBListResponse
			var err error
			if all {
				resp, err = c.AllKnowledge(options, true)
			} else if discoveryflags.LookupRequested(cmd) {
				resp, err = c.KBListWithOptions(options, true)
			} else {
				resp, err = c.KBSubscribedList(page, scope)
			}
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

	discoveryflags.BindLookup(cmd, &options, &all)
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().StringVar(&scope, "scope", "DEFAULT", "Knowledge base scope: DEFAULT, CUSTOMER, BOOKSPACE, TEAMSPACE")
	return cmd
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}
