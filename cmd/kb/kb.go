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
		Short: "列出知识库内的笔记 / List notes in a knowledge base",
		Args:  cobra.ExactArgs(1),
		Example: `  getnote kb vnrOAaGY
  getnote kb vnrOAaGY --limit 5
  getnote kb vnrOAaGY --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")

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
	cmd.AddCommand(newBloggersCmd())
	cmd.AddCommand(newBloggerContentsCmd())
	cmd.AddCommand(newBloggerContentCmd())
	cmd.AddCommand(newLivesCmd())
	cmd.AddCommand(newLiveCmd())
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
		Short: "创建知识库 / Create a new knowledge base",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
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
		Short: "添加笔记到知识库 / Add notes to a knowledge base",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
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
		Short: "从知识库移除笔记 / Remove notes from a knowledge base",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
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

func newBloggersCmd() *cobra.Command {
	var page int
	cmd := &cobra.Command{
		Use:   "bloggers <topic_id>",
		Short: "列出知识库订阅的博主 / List bloggers in a knowledge base",
		Args:  cobra.ExactArgs(1),
		Example: `  getnote kb bloggers vnrOAaGY
  getnote kb bloggers vnrOAaGY --page 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
			resp, err := c.KBBloggerList(args[0], page)
			if err != nil {
				return err
			}
			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Follow ID", "Name", "Platform", "Followed At"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			for _, b := range resp.Data.Bloggers {
				table.Append([]string{b.FollowID.String(), b.AccountName, b.Platform, b.FollowTime})
			}
			table.Render()
			fmt.Fprintf(cmd.OutOrStdout(), "\n(%d bloggers)\n", len(resp.Data.Bloggers))
			return nil
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "页码 / Page number")
	return cmd
}

func newBloggerContentsCmd() *cobra.Command {
	var page int
	cmd := &cobra.Command{
		Use:   "blogger-contents <topic_id> <follow_id>",
		Short: "列出博主内容 / List blogger contents",
		Args:  cobra.ExactArgs(2),
		Example: `  getnote kb blogger-contents vnrOAaGY follow123
  getnote kb blogger-contents vnrOAaGY follow123 --page 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
			resp, err := c.KBBloggerContentList(args[0], args[1], page)
			if err != nil {
				return err
			}
			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Post ID", "Title", "Type", "Published"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			for _, c := range resp.Data.Contents {
				table.Append([]string{c.PostIDAlias, ui.Truncate(c.PostTitle, 50), c.PostType, c.PublishTime})
			}
			table.Render()
			fmt.Fprintf(cmd.OutOrStdout(), "\n(%d contents)\n", len(resp.Data.Contents))
			return nil
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "页码 / Page number")
	return cmd
}

func newBloggerContentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "blogger-content <topic_id> <post_id>",
		Short: "查看博主内容详情（含原文）/ Show blogger content detail",
		Args:  cobra.ExactArgs(2),
		Example: `  getnote kb blogger-content vnrOAaGY post_abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
			resp, err := c.KBBloggerContentGet(args[0], args[1])
			if err != nil {
				return err
			}
			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			d := resp.Data.Content
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Field", "Value"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			table.Append([]string{"ID", d.PostIDAlias})
			table.Append([]string{"Title", d.PostTitle})
			table.Append([]string{"Author", d.AccountName})
			table.Append([]string{"Type", d.PostType})
			table.Append([]string{"Published", d.PublishTime})
			if d.PostSummary != "" {
				table.Append([]string{"Summary", ui.Truncate(d.PostSummary, 200)})
			}
			if d.PostMediaText != "" {
				table.Append([]string{"Content", ui.Truncate(d.PostMediaText, 300)})
			}
			table.Render()
			return nil
		},
	}
}

func newLivesCmd() *cobra.Command {
	var page int
	cmd := &cobra.Command{
		Use:   "lives <topic_id>",
		Short: "列出知识库已完成的直播 / List completed lives in a knowledge base",
		Args:  cobra.ExactArgs(1),
		Example: `  getnote kb lives vnrOAaGY
  getnote kb lives vnrOAaGY --page 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
			resp, err := c.KBLiveList(args[0], page)
			if err != nil {
				return err
			}
			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Live ID", "Name", "Status"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			for _, l := range resp.Data.Lives {
				table.Append([]string{l.LiveID, ui.Truncate(l.Name, 50), l.Status})
			}
			table.Render()
			fmt.Fprintf(cmd.OutOrStdout(), "\n(%d lives)\n", len(resp.Data.Lives))
			return nil
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "页码 / Page number")
	return cmd
}

func newLiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "live <topic_id> <live_id>",
		Short: "查看直播详情（含 AI 摘要和原文）/ Show live detail",
		Args:  cobra.ExactArgs(2),
		Example: `  getnote kb live vnrOAaGY live_abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
			resp, err := c.KBLiveGet(args[0], args[1])
			if err != nil {
				return err
			}
			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			d := resp.Data.Live
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Field", "Value"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			table.Append([]string{"ID", d.LiveID})
			table.Append([]string{"Name", d.Name})
			table.Append([]string{"Status", d.Status})
			if d.PostSummary != "" {
				table.Append([]string{"Summary", ui.Truncate(d.PostSummary, 200)})
			}
			if d.PostMediaText != "" {
				table.Append([]string{"Transcript", ui.Truncate(d.PostMediaText, 300)})
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

