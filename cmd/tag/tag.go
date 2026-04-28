package tag

import (
	"encoding/json"
	"fmt"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/ui"
	"github.com/spf13/cobra"
)

// NewTagCmd returns the tag parent command with subcommands.
func NewTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "管理笔记标签 / Manage note tags",
	}
	cmd.AddCommand(newTagAddCmd())
	cmd.AddCommand(newTagRemoveCmd())
	cmd.AddCommand(newTagListCmd())
	return cmd
}

func newTagAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "add <note_id> <tag>",
		Short:   "为笔记添加标签 / Add a tag to a note",
		Args:    cobra.ExactArgs(2),
		Example: `  getnote tag add 1896830231705320746 工作`,
		RunE: func(cmd *cobra.Command, args []string) error {
			noteID := args[0]
			tagName := args[1]
			c := client.New("")

			resp, err := c.NoteTagsAdd(noteID, []string{tagName})
			if err != nil {
				return ui.FriendlyError(err)
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			if !resp.Success {
				msg := "unknown error"
				if resp.Error != nil {
					msg = resp.Error.Message
				}
				return fmt.Errorf("API error: %s", msg)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Tag added. Note %s now has %d tag(s):\n", resp.Data.NoteID, len(resp.Data.Tags))
			printTags(cmd, resp.Data.Tags)
			return nil
		},
	}
}

func newTagRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <note_id> <tag_id>",
		Short: "删除笔记标签 / Remove a tag from a note by tag ID",
		Long: `Remove a tag from a note. Provide the tag ID (not name).
Use "getnote tag list <note_id>" to find tag IDs.
System tags cannot be deleted.`,
		Args:    cobra.ExactArgs(2),
		Example: `  getnote tag remove 1896830231705320746 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			noteID := args[0]
			tagID := args[1]
			c := client.New("")

			resp, err := c.NoteTagsDelete(noteID, tagID)
			if err != nil {
				return ui.FriendlyError(err)
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			if !resp.Success {
				msg := "unknown error"
				if resp.Error != nil {
					msg = resp.Error.Message
				}
				return fmt.Errorf("API error: %s", msg)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Tag removed.")
			return nil
		},
	}
}

func newTagListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list <note_id>",
		Short:   "查看笔记标签 / List tags on a note",
		Args:    cobra.ExactArgs(1),
		Example: `  getnote tag list 1896830231705320746`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")

			noteResp, err := c.NoteGet(args[0])
			if err != nil {
				return ui.FriendlyError(err)
			}
			note := noteResp.Data.Note

			// Parse tags as NoteTag objects (detail endpoint returns full objects)
			var tags []client.NoteTag
			for _, raw := range note.Tags {
				var t client.NoteTag
				if json.Unmarshal(raw, &t) == nil && t.ID != "" {
					tags = append(tags, t)
				}
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]interface{}{
					"note_id": note.NoteID.String(),
					"tags":    tags,
				})
			}

			if len(tags) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No tags.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Tags for note %s:\n", note.NoteID.String())
			printTags(cmd, tags)
			return nil
		},
	}
}

var tagCols = []ui.ColSpec{
	{Value: "ID", Width: 12},
	{Value: "Name", Width: 30},
	{Value: "Type", Width: 10},
}

const sep = "  "

func printTags(cmd *cobra.Command, tags []client.NoteTag) {
	out := cmd.OutOrStdout()
	fmt.Fprint(out, ui.PrintHeader(tagCols, sep))
	fmt.Fprint(out, ui.DividerLine(tagCols, sep))
	for _, t := range tags {
		row := []ui.ColSpec{
			{Value: t.ID, Width: tagCols[0].Width},
			{Value: t.Name, Width: tagCols[1].Width},
			{Value: t.Type, Width: tagCols[2].Width},
		}
		fmt.Fprint(out, ui.PrintRow(row, sep))
	}
	fmt.Fprintf(out, "\n(%d tag(s))\n", len(tags))
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}

