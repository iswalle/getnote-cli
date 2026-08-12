package kb

import (
	"encoding/json"
	"fmt"
	"strings"
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
	var noContent bool

	cmd := &cobra.Command{
		Use:   "kb <topic_id>",
		Short: "列出知识库内的笔记 / List notes in a knowledge base",
		Args:  cobra.ExactArgs(1),
		Example: `  getnote kb vnrOAaGY
  getnote kb vnrOAaGY --limit 5
  getnote kb vnrOAaGY --all
  getnote kb vnrOAaGY -o json --no-content`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")

			if all {
				return streamAllKBNotes(cmd, c, args[0], noContent)
			}

			resp, err := c.KBNotes(client.KBNotesParams{TopicID: args[0], Limit: limit})
			if err != nil {
				return ui.FriendlyError(err)
			}

			if outputFormat(cmd) == "json" {
				if noContent {
					for i := range resp.Data.Notes {
						resp.Data.Notes[i].Content = ""
					}
				}
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
	cmd.Flags().BoolVar(&noContent, "no-content", false, "Omit the content field in JSON output (saves tokens for AI agents)")

	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newDirectoriesCmd())
	cmd.AddCommand(newDirectoryCreateCmd())
	cmd.AddCommand(newDirectoryUpdateCmd())
	cmd.AddCommand(newDirectoryDeleteCmd())
	cmd.AddCommand(newBloggersCmd())
	cmd.AddCommand(newBloggerFollowCmd())
	cmd.AddCommand(newBloggerContentsCmd())
	cmd.AddCommand(newBloggerContentCmd())
	cmd.AddCommand(newLivesCmd())
	cmd.AddCommand(newLiveCmd())
	cmd.AddCommand(newLiveFollowCmd())
	return cmd
}

func newLiveFollowCmd() *cobra.Command {
	var platform string
	cmd := &cobra.Command{
		Use:     "live-follow <topic_id> <link>",
		Short:   "订阅直播到知识库 / Follow a live into a knowledge base",
		Example: `  getnote kb live-follow vnrOAaGY https://www.dedao.cn/live/example`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := client.New("").KBLiveFollow(args[0], args[1], platform)
			if err != nil {
				return ui.FriendlyError(err)
			}
			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			followID := resp.Data.FollowIDStr
			if followID == "" {
				followID = resp.Data.FollowID.String()
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Live followed (follow_id: %s).\n", followID)
			return nil
		},
	}
	cmd.Flags().StringVar(&platform, "platform", "", "Platform override (normally auto-detected)")
	return cmd
}

var noteCols = []ui.ColSpec{
	{Value: "ID", Width: 20},
	{Value: "Title", Width: 52},
	{Value: "Type", Width: 14},
	{Value: "Created", Width: 19},
}

const colSep = "  "

func streamAllKBNotes(cmd *cobra.Command, c *client.Client, topicID string, noContent bool) error {
	isJSON := outputFormat(cmd) == "json"

	if isJSON {
		var allNotes []client.KBNote
		page := 1
		for {
			resp, err := c.KBNotes(client.KBNotesParams{TopicID: topicID, Limit: 20, Page: page})
			if err != nil {
				return ui.FriendlyError(err)
			}
			if noContent {
				for i := range resp.Data.Notes {
					resp.Data.Notes[i].Content = ""
				}
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
			Data: client.KBNoteListData{
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
				{Value: n.NoteID, Width: noteCols[0].Width},
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

func renderNoteRows(table *tablewriter.Table, data client.KBNoteListData) {
	for _, n := range data.Notes {
		table.Append([]string{
			n.NoteID,
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
		Example: `  getnote kb create "产品研究"
  getnote kb create "产品研究" --desc "产品资料与用户反馈"`,
		Args: cobra.ExactArgs(1),
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
			if !resp.Success {
				msg := "unknown error"
				if resp.Error != nil {
					msg = resp.Error.Message
				}
				return fmt.Errorf("API error: %s", msg)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Knowledge base created.")
			return nil
		},
	}
	cmd.Flags().StringVar(&desc, "desc", "", "Description")
	return cmd
}

func newAddCmd() *cobra.Command {
	var directoryID string
	cmd := &cobra.Command{
		Use:   "add <topic_id> <note_id> [note_id...]",
		Short: "添加笔记到知识库 / Add notes to a knowledge base",
		Long:  "将笔记加入自有知识库。每次最多处理 20 条；订阅知识库只读。",
		Example: `  getnote kb add vnrOAaGY 1896830231705320746
  getnote kb add vnrOAaGY 1896830231705320746 --directory-id 7123456789012345678
  getnote kb add vnrOAaGY 1896830231705320746 1896830231705320747`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateNoteBatch(args[1:]); err != nil {
				return err
			}
			c := client.New("")
			resp, err := c.KBNotesAdd(args[0], directoryID, args[1:])
			if err != nil {
				return err
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
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Added %d note(s) to %s.\n", len(args[1:]), args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&directoryID, "directory-id", "", "Target directory ID; omit for the knowledge-base root")
	return cmd
}

func newDirectoriesCmd() *cobra.Command {
	var directoryID string
	cmd := &cobra.Command{
		Use:     "directories <topic_id>",
		Aliases: []string{"dir", "dirs"},
		Short:   "浏览知识库目录和资源 / Browse knowledge-base folders and resources",
		Example: "  getnote kb directories vnrOAaGY\n  getnote kb directories vnrOAaGY --directory-id 7123456789012345678",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := client.New("").KBDirectoryList(args[0], directoryID)
			if err != nil {
				return ui.FriendlyError(err)
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		},
	}
	cmd.Flags().StringVar(&directoryID, "directory-id", "", "Directory ID to browse; omit for root")
	return cmd
}

func newDirectoryCreateCmd() *cobra.Command {
	var parentID, nameFlag string
	cmd := &cobra.Command{
		Use:     "directory-create <topic_id> [name]",
		Aliases: []string{"mkdir"},
		Short:   "创建知识库目录 / Create a knowledge-base folder",
		Long:    "目录名称既可作为第二个位置参数，也可通过 --name 指定；两种写法等价且不能同时使用。",
		Example: "  getnote kb directory-create vnrOAaGY 产品资料\n  getnote kb directory-create vnrOAaGY --name 用户研究 --parent-id 7123456789012345678",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := resolveDirectoryCreateName(args, nameFlag)
			if err != nil {
				return err
			}
			resp, err := client.New("").KBDirectoryCreate(args[0], parentID, name)
			if err != nil {
				return ui.FriendlyError(err)
			}
			return writeJSON(cmd, resp)
		},
	}
	cmd.Flags().StringVar(&nameFlag, "name", "", "Directory name (alternative to the positional name argument)")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Parent directory ID; omit for root")
	return cmd
}

func resolveDirectoryCreateName(args []string, nameFlag string) (string, error) {
	positionalName := ""
	if len(args) > 1 {
		positionalName = strings.TrimSpace(args[1])
	}
	nameFlag = strings.TrimSpace(nameFlag)
	if positionalName != "" && nameFlag != "" {
		return "", fmt.Errorf("目录名称不能同时使用位置参数和 --name 指定")
	}
	if positionalName != "" {
		return positionalName, nil
	}
	if nameFlag != "" {
		return nameFlag, nil
	}
	return "", fmt.Errorf("必须提供目录名称：使用 <name> 或 --name")
}

func newDirectoryUpdateCmd() *cobra.Command {
	var parentID, name string
	cmd := &cobra.Command{
		Use:     "directory-update <topic_id> <directory_id>",
		Aliases: []string{"mvdir"},
		Short:   "重命名或移动知识库目录 / Rename or move a knowledge-base folder",
		Long:    "至少提供 --name 或 --parent-id。只改名称时目录位置保持不变，只移动时目录名称保持不变。",
		Example: "  getnote kb directory-update vnrOAaGY 7123456789012345678 --name 新名称\n  getnote kb directory-update vnrOAaGY 7123456789012345678 --parent-id 7234567890123456789",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" && parentID == "" {
				return fmt.Errorf("at least one of --name or --parent-id is required")
			}
			resp, err := client.New("").KBDirectoryUpdate(args[0], args[1], parentID, name)
			if err != nil {
				return ui.FriendlyError(err)
			}
			return writeJSON(cmd, resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New directory name; omit to keep the current name")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "New parent directory ID; omit to keep the current parent")
	return cmd
}

func newDirectoryDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "directory-delete <topic_id> <directory_id>",
		Aliases: []string{"rmdir"},
		Short:   "删除空知识库目录 / Delete an empty knowledge-base folder",
		Example: "  getnote kb directory-delete vnrOAaGY 7123456789012345678 --yes",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			approved, err := ui.ConfirmDestructive(cmd, yes, "Delete this knowledge-base directory?")
			if err != nil || !approved {
				return err
			}
			resp, err := client.New("").KBDirectoryDelete(args[0], args[1])
			if err != nil {
				return ui.FriendlyError(err)
			}
			return writeJSON(cmd, resp)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func newBloggerFollowCmd() *cobra.Command {
	var platform string
	cmd := &cobra.Command{
		Use:     "blogger-follow <topic_id> <link>",
		Short:   "订阅抖音博主到知识库 / Follow a Douyin creator into a knowledge base",
		Example: "  getnote kb blogger-follow vnrOAaGY https://www.douyin.com/user/example",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := client.New("").KBBloggerFollow(args[0], args[1], platform)
			if err != nil {
				return ui.FriendlyError(err)
			}
			return writeJSON(cmd, resp)
		},
	}
	cmd.Flags().StringVar(&platform, "platform", "douyin", "Creator platform")
	return cmd
}

func writeJSON(cmd *cobra.Command, value interface{}) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func newRemoveCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "remove <topic_id> <note_id> [note_id...]",
		Short: "从知识库移除笔记 / Remove notes from a knowledge base",
		Long:  "从自有知识库移出笔记。每次最多处理 20 条，执行前必须确认。",
		Example: `  getnote kb remove vnrOAaGY 1896830231705320746
  getnote kb remove vnrOAaGY 1896830231705320746 --yes`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateNoteBatch(args[1:]); err != nil {
				return err
			}
			approved, err := ui.ConfirmDestructive(cmd, yes, fmt.Sprintf("Remove %d note(s) from knowledge base %s?", len(args[1:]), args[0]))
			if err != nil {
				return err
			}
			if !approved {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
			}
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
			if !resp.Success {
				msg := "unknown error"
				if resp.Error != nil {
					msg = resp.Error.Message
				}
				return fmt.Errorf("API error: %s", msg)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Removed %d note(s) from %s.\n", len(args[1:]), args[0])
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm removing notes without prompting")
	return cmd
}

func validateNoteBatch(noteIDs []string) error {
	if len(noteIDs) > 20 {
		return fmt.Errorf("每批最多处理 20 条笔记，当前为 %d 条", len(noteIDs))
	}
	return nil
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
				followID := b.FollowIDStr
				if followID == "" {
					followID = b.FollowID.String()
				}
				table.Append([]string{followID, b.AccountName, b.Platform, b.FollowTime})
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
		Use:     "blogger-content <topic_id> <post_id>",
		Short:   "查看博主内容详情（含原文）/ Show blogger content detail",
		Args:    cobra.ExactArgs(2),
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
			d := resp.Data
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Field", "Value"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			name := d.PostName
			if d.PostTitle != "" {
				name = d.PostTitle
			}
			table.Append([]string{"Name", name})
			if d.PostSubtitle != "" {
				table.Append([]string{"Subtitle", d.PostSubtitle})
			}
			table.Append([]string{"Published", d.PublishTime})
			if d.PostSummary != "" {
				table.Append([]string{"Summary", ui.Truncate(d.PostSummary, 200)})
			}
			if d.PostMediaText != "" {
				table.Append([]string{"Content", ui.Truncate(d.PostMediaText, 500)})
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
		Use:     "live <topic_id> <live_id>",
		Short:   "查看直播详情（含 AI 摘要和原文）/ Show live detail",
		Args:    cobra.ExactArgs(2),
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
			d := resp.Data
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Field", "Value"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			name := d.PostName
			if d.PostTitle != "" {
				name = d.PostTitle
			}
			table.Append([]string{"Name", name})
			if d.PostSubtitle != "" {
				table.Append([]string{"Subtitle", d.PostSubtitle})
			}
			table.Append([]string{"Published", d.PublishTime})
			if d.PostSummary != "" {
				table.Append([]string{"Summary", ui.Truncate(d.PostSummary, 200)})
			}
			if d.PostMediaText != "" {
				table.Append([]string{"Transcript", ui.Truncate(d.PostMediaText, 500)})
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
