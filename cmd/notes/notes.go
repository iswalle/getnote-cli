package notes

import (
	"encoding/json"
	"fmt"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewNotesCmd returns the top-level notes (list) command.
func NewNotesCmd() *cobra.Command {
	var limit int
	var sinceID string
	var all bool

	cmd := &cobra.Command{
		Use:   "notes",
		Short: "List recent notes",
		Example: `  getnote notes
  getnote notes --limit 5
  getnote notes --all
  getnote notes --since-id 1234567890`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))

			if all {
				return listAll(cmd, c)
			}

			resp, err := c.NoteList(client.NoteListParams{Limit: limit, SinceID: sinceID})
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"ID", "Title", "Type", "Created"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			for _, n := range resp.Data.Notes {
				id := n.NoteID.String()
				if id == "" || id == "0" {
					id = n.ID.String()
				}
				table.Append([]string{id, n.Title, n.NoteType, n.CreatedAt})
			}
			table.Render()

			if resp.Data.HasMore {
				fmt.Fprintf(cmd.OutOrStdout(),
					"\n(showing %d notes, use --since-id %s for next page, or --all for everything)\n",
					len(resp.Data.Notes), resp.Data.NextCursor.String())
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Number of notes per page")
	cmd.Flags().StringVar(&sinceID, "since-id", "", "Pagination cursor (note ID)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all notes (auto-paginate)")
	return cmd
}

func listAll(cmd *cobra.Command, c *client.Client) error {
	table := tablewriter.NewWriter(cmd.OutOrStdout())
	table.SetHeader([]string{"ID", "Title", "Type", "Created"})
	table.SetBorder(false)
	table.SetAutoWrapText(false)

	sinceID := "0"
	total := 0
	for {
		resp, err := c.NoteList(client.NoteListParams{Limit: 20, SinceID: sinceID})
		if err != nil {
			return err
		}
		for _, n := range resp.Data.Notes {
			id := n.NoteID.String()
			if id == "" || id == "0" {
				id = n.ID.String()
			}
			table.Append([]string{id, n.Title, n.NoteType, n.CreatedAt})
			total++
		}
		if !resp.Data.HasMore {
			break
		}
		sinceID = resp.Data.NextCursor.String()
	}
	table.Render()
	fmt.Fprintf(cmd.OutOrStdout(), "\n(%d notes total)\n", total)
	return nil
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}

func envTarget(cmd *cobra.Command) string {
	e, _ := cmd.Root().PersistentFlags().GetString("env")
	return e
}
