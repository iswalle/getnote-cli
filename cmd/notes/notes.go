package notes

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/ui"
	"github.com/spf13/cobra"
)

const sep = "  "

var cols = []ui.ColSpec{
	{Value: "ID", Width: 20},
	{Value: "Title", Width: 52},
	{Value: "Type", Width: 14},
	{Value: "Created", Width: 19},
}

// NewNotesCmd returns the top-level notes (list) command.
func NewNotesCmd() *cobra.Command {
	var limit int
	var sinceID string
	var all bool

	cmd := &cobra.Command{
		Use:   "notes",
		Short: "列出最近笔记 / List recent notes",
		Example: `  getnote notes
  getnote notes --limit 5
  getnote notes --all
  getnote notes --since-id 1234567890`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")

			if all {
				return streamAll(cmd, c)
			}

			resp, err := c.NoteList(client.NoteListParams{Limit: limit, SinceID: sinceID})
			if err != nil {
				return ui.FriendlyError(err)
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			printHeader(cmd)
			shown := resp.Data.Notes
			if limit > 0 && len(shown) > limit {
				shown = shown[:limit]
			}
			for _, n := range shown {
				printRow(cmd, n)
			}

			hasMore := resp.Data.HasMore || len(resp.Data.Notes) > len(shown)
			if hasMore {
				fmt.Fprintf(cmd.OutOrStdout(),
					"\n(showing %d of more notes — use --since-id %s for next page, or --all)\n",
					len(shown), resp.Data.NextCursor.String())
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "\n(%d notes)\n", len(shown))
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Number of notes per page")
	cmd.Flags().StringVar(&sinceID, "since-id", "", "Pagination cursor (note ID)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all notes (auto-paginate, streams output)")
	return cmd
}

// streamAll fetches all notes page by page, printing each row immediately.
func streamAll(cmd *cobra.Command, c *client.Client) error {
	isJSON := outputFormat(cmd) == "json"

	if isJSON {
		var allNotes []client.Note
		sinceID := "0"
		for {
			resp, err := c.NoteList(client.NoteListParams{Limit: 20, SinceID: sinceID})
			if err != nil {
				return ui.FriendlyError(err)
			}
			allNotes = append(allNotes, resp.Data.Notes...)
			if !resp.Data.HasMore {
				break
			}
			sinceID = resp.Data.NextCursor.String()
			time.Sleep(500 * time.Millisecond)
		}
		result := &client.NoteListResponse{
			Success: true,
			Data: client.NoteListData{
				Notes:   allNotes,
				HasMore: false,
				Total:   len(allNotes),
			},
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	printHeader(cmd)
	sinceID := "0"
	total := 0
	for {
		resp, err := c.NoteList(client.NoteListParams{Limit: 20, SinceID: sinceID})
		if err != nil {
			return ui.FriendlyError(err)
		}
		for _, n := range resp.Data.Notes {
			printRow(cmd, n)
			total++
		}
		if !resp.Data.HasMore {
			break
		}
		sinceID = resp.Data.NextCursor.String()
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n(%d notes total)\n", total)
	return nil
}

func printHeader(cmd *cobra.Command) {
	fmt.Fprint(cmd.OutOrStdout(), ui.PrintHeader(cols, sep))
	fmt.Fprint(cmd.OutOrStdout(), ui.DividerLine(cols, sep))
}

func printRow(cmd *cobra.Command, n client.Note) {
	row := []ui.ColSpec{
		{Value: ui.NoteID(n.NoteID, n.ID), Width: cols[0].Width},
		{Value: n.Title, Width: cols[1].Width},
		{Value: n.NoteType, Width: cols[2].Width},
		{Value: n.CreatedAt, Width: cols[3].Width},
	}
	fmt.Fprint(cmd.OutOrStdout(), ui.PrintRow(row, sep))
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}

