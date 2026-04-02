package kb

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/ui"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewKbCmd returns the kb command (notes in KB + create + add + remove).
func NewKbCmd() *cobra.Command {
	var limit int
	var all bool

	cmd := &cobra.Command{
		Use:   "kb <topic_id>",
		Short: "List notes in a knowledge base",
		Args:  cobra.ExactArgs(1),
		Example: `  getnote kb vnrOAaGY
  getnote kb vnrOAaGY --limit 5
  getnote kb vnrOAaGY --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))

			if all {
				return streamAllKBNotes(cmd, c, args[0])
			}

			resp, err := c.KBNotes(client.KBNotesParams{TopicID: args[0], Limit: limit})
			if err != nil {
				return ui.FriendlyError(err)
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
			renderNoteRows(table, resp.Data)
			table.Render()

			if resp.Data.HasMore {
				fmt.Fprintf(cmd.OutOrStdout(),
					"\n(showing %d of more notes — use --all for everything)\n",
					len(resp.Data.Notes))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "\n(%d notes)\n", len(resp.Data.Notes))
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Number of notes per page")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all notes (auto-paginate)")

	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	return cmd
}

var noteCols = []ui.ColSpec{
	{Value: "ID", Width: 20},
	{Value: "Title", Width: 52},
	{Value: "Type", Width: 14},
	{Value: "Created", Width: 19},
}

const colSep = "  "

func streamAllKBNotes(cmd *cobra.Command, c *client.Client, topicID string) error {
	isJSON := outputFormat(cmd) == "json"

	if isJSON {
		var allNotes []client.Note
		page := 1
		for {
			resp, err := c.KBNotes(client.KBNotesParams{TopicID: topicID, Limit: 20, Page: page})
			if err != nil {
				return ui.FriendlyError(err)
			}
			allNotes = append(allNotes, resp.Data.Notes...)
			if !resp.Data.HasMore {
				break
			}
			page++
			time.Sleep(500 * time.Millisecond)
		}
		result := &client.KBNotesResponse{
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

	fmt.Fprint(cmd.OutOrStdout(), ui.PrintHeader(noteCols, colSep))
	fmt.Fprint(cmd.OutOrStdout(), ui.DividerLine(noteCols, colSep))

	page := 1
	total := 0
	for {
		resp, err := c.KBNotes(client.KBNotesParams{TopicID: topicID, Limit: 20, Page: page})
		if err != nil {
			return ui.FriendlyError(err)
		}
		for _, n := range resp.Data.Notes {
			row := []ui.ColSpec{
				{Value: ui.NoteID(n.NoteID, n.ID), Width: noteCols[0].Width},
				{Value: n.Title, Width: noteCols[1].Width},
				{Value: n.NoteType, Width: noteCols[2].Width},
				{Value: n.CreatedAt, Width: noteCols[3].Width},
			}
			fmt.Fprint(cmd.OutOrStdout(), ui.PrintRow(row, colSep))
			total++
		}
		if !resp.Data.HasMore {
			break
		}
		page++
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n(%d notes total)\n", total)
	return nil
}

func renderNoteRows(table *tablewriter.Table, data client.NoteListData) {
	for _, n := range data.Notes {
		table.Append([]string{
			ui.NoteID(n.NoteID, n.ID),
			ui.Truncate(n.Title, 40),
			n.NoteType,
			n.CreatedAt,
		})
	}
}

func newCreateCmd() *cobra.Command {
	var desc string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new knowledge base",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))
			resp, err := c.KBCreate(client.KBCreateRequest{Name: args[0], Description: desc})
			if err != nil {
				return err
			}
			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Knowledge base created.")
			return nil
		},
	}
	cmd.Flags().StringVar(&desc, "desc", "", "Description")
	return cmd
}

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <topic_id> <note_id> [note_id...]",
		Short: "Add notes to a knowledge base",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))
			resp, err := c.KBNotesAdd(args[0], args[1:])
			if err != nil {
				return err
			}
			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Added %d note(s) to %s.\n", len(args[1:]), args[0])
			return nil
		},
	}
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <topic_id> <note_id> [note_id...]",
		Short: "Remove notes from a knowledge base",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(envTarget(cmd))
			resp, err := c.KBNotesRemove(args[0], args[1:])
			if err != nil {
				return err
			}
			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Removed %d note(s) from %s.\n", len(args[1:]), args[0])
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
