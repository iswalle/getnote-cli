package notes

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/ui"
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
			for _, n := range resp.Data.Notes {
				printRow(cmd, n)
			}

			if resp.Data.HasMore {
				fmt.Fprintf(cmd.OutOrStdout(),
					"\n(showing %d of more notes — use --since-id %s for next page, or --all)\n",
					len(resp.Data.Notes), resp.Data.NextCursor.String())
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "\n(%d notes)\n", len(resp.Data.Notes))
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
		time.Sleep(500 * time.Millisecond) // respect QPS limit
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n(%d notes total)\n", total)
	return nil
}

const colID = 20
const colTitle = 60
const colType = 16
const colCreated = 20

func printHeader(cmd *cobra.Command) {
	fmt.Fprintf(cmd.OutOrStdout(), "%-*s  %-*s  %-*s  %-*s\n",
		colID, "ID", colTitle, "Title", colType, "Type", colCreated, "Created")
	fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("-", colID+colTitle+colType+colCreated+6))
}

func printRow(cmd *cobra.Command, n client.Note) {
	title := ui.Truncate(n.Title, colTitle)
	fmt.Fprintf(cmd.OutOrStdout(), "%-*s  %-*s  %-*s  %-*s\n",
		colID, ui.NoteID(n.NoteID, n.ID), colTitle, title, colType, n.NoteType, colCreated, n.CreatedAt)
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}

func envTarget(cmd *cobra.Command) string {
	e, _ := cmd.Root().PersistentFlags().GetString("env")
	return e
}
