package notes

import (
	"encoding/json"
	"fmt"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/discoveryflags"
	"github.com/iswalle/getnote-cli/internal/ui"
	"github.com/spf13/cobra"
)

const sep = "  "

var cols = []ui.ColSpec{
	{Value: "ID", Width: 20},
	{Value: "Title", Width: 38},
	{Value: "Type", Width: 14},
	{Value: "Created", Width: 19},
	{Value: "Note URL", Width: 44},
}

// NewNotesCmd returns the top-level notes (list) command.
func NewNotesCmd() *cobra.Command {
	var limit int
	var cursor string
	var all bool
	var options client.DiscoveryOptions

	cmd := &cobra.Command{
		Use:   "notes",
		Args:  cobra.NoArgs,
		Short: "列出最近笔记 / List recent notes",
		Example: `  getnote notes
  getnote notes --limit 5
  getnote notes --all
  getnote notes --cursor 1914025811879486080`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")

			options.Cursor = cursor
			resolved, err := discoveryflags.ResolveNotes(cmd, options, limit)
			if err != nil {
				return err
			}
			var resp *client.NoteListResponse
			if all {
				if resolved.Active() {
					resp, err = c.AllNotes(resolved)
				} else {
					resp, err = c.LegacyAllNotes(cursor)
				}
			} else if resolved.Active() {
				resp, err = c.NoteListWithOptions(resolved)
			} else {
				resp, err = c.NoteList(client.NoteListParams{Limit: limit, Cursor: cursor})
				if err == nil {
					applyLimit(resp, limit)
				}
			}
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
			for _, n := range shown {
				printRow(cmd, n)
			}

			if resp.Data.HasMore {
				fmt.Fprintf(cmd.OutOrStdout(),
					"\n(showing %d of more notes — use --cursor %s for next page, or --all)\n",
					len(shown), resp.Data.Cursor)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "\n(%d notes returned in this traversal; no snapshot guarantee)\n", len(shown))
			}
			return nil
		},
	}

	discoveryflags.BindNotes(cmd, &options, true)
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of notes per page")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor (from a previous page's cursor)")
	cmd.Flags().BoolVar(&all, "all", false, "Traverse all matching pages; report incomplete on errors, no snapshot guarantee")
	cmd.AddCommand(newBatchCmd())
	return cmd
}

func applyLimit(resp *client.NoteListResponse, limit int) {
	if resp == nil || limit <= 0 || len(resp.Data.Notes) <= limit {
		return
	}
	resp.Data.Notes = resp.Data.Notes[:limit]
	last := resp.Data.Notes[len(resp.Data.Notes)-1]
	cursor := ui.NoteID(last.NoteID, last.ID)
	resp.Data.Cursor = cursor
	resp.Data.NextCursor = client.StringID(cursor)
	resp.Data.HasMore = true
}

func printHeader(cmd *cobra.Command) {
	fmt.Fprint(cmd.OutOrStdout(), ui.PrintHeader(cols, sep))
	fmt.Fprint(cmd.OutOrStdout(), ui.DividerLine(cols, sep))
}

func printRow(cmd *cobra.Command, n client.Note) {
	row := []ui.ColSpec{
		{Value: ui.NoteID(n.NoteID, n.ID), Width: cols[0].Width},
		{Value: n.DisplayName(), Width: cols[1].Width},
		{Value: n.NoteType, Width: cols[2].Width},
		{Value: n.CreatedAt, Width: cols[3].Width},
		{Value: n.NoteURL, Width: cols[4].Width},
	}
	fmt.Fprint(cmd.OutOrStdout(), ui.PrintRow(row, sep))
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}
