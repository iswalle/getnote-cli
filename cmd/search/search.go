package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/ui"
	"github.com/spf13/cobra"
)

var cols = []ui.ColSpec{
	{Value: "ID", Width: 20},
	{Value: "Title", Width: 40},
	{Value: "Type", Width: 10},
	{Value: "Score", Width: 6},
	{Value: "Created", Width: 19},
	{Value: "Content", Width: 50},
	{Value: "Note URL", Width: 44},
}

const sep = "  "

// NewSearchCmd returns the search command.
func NewSearchCmd() *cobra.Command {
	var limit int
	var kb string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "语义搜索笔记 / Search notes by semantic query",
		Args:  cobra.MinimumNArgs(1),
		Example: `  getnote search "大模型 API"
  getnote search "RAG" --kb qnNX75j0
  getnote search "机器学习" --limit 5 -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit < 1 || limit > 10 {
				return fmt.Errorf("--limit 必须在 1 到 10 之间")
			}
			query := strings.Join(args, " ")
			c := client.New("")

			var resp *client.NoteSearchResponse
			var err error

			if kb != "" {
				resp, err = c.KBSearch(kb, query, limit)
			} else {
				resp, err = c.NoteSearch(query, limit)
			}
			if err != nil {
				if isSearchTimeout(err) {
					return fmt.Errorf("搜索服务响应超时，请稍后重试；也可以缩小关键词或指定知识库后再试: %w", err)
				}
				return ui.FriendlyError(err)
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			out := cmd.OutOrStdout()
			partial := (resp.Data.Partial != nil && *resp.Data.Partial) || len(resp.Data.Warnings) > 0
			if partial {
				fmt.Fprintln(out, "Partial search: some results could not be verified or read.")
				for _, warning := range resp.Data.Warnings {
					fmt.Fprintf(out, "Warning: %s\n", warning)
				}
			}
			results := resp.Data.Results
			if len(results) == 0 {
				if partial {
					fmt.Fprintln(out, "No verified results available; this does not establish that no matches exist.")
					return nil
				}
				fmt.Fprintln(out, "No results found.")
				return nil
			}

			fmt.Fprint(out, ui.PrintHeader(cols, sep))
			fmt.Fprint(out, ui.DividerLine(cols, sep))
			for _, r := range results {
				title, noteType := r.Title, r.NoteType
				if r.Note != nil {
					title = r.Note.DisplayName()
					noteType = r.Note.NoteType
				}
				row := []ui.ColSpec{
					{Value: r.NoteID, Width: cols[0].Width},
					{Value: title, Width: cols[1].Width},
					{Value: noteType, Width: cols[2].Width},
					{Value: scoreText(r.Score), Width: cols[3].Width},
					{Value: r.CreatedAt, Width: cols[4].Width},
					{Value: ui.Truncate(r.Content, cols[5].Width), Width: cols[5].Width},
					{Value: r.NoteURL, Width: cols[6].Width},
				}
				fmt.Fprint(out, ui.PrintRow(row, sep))
			}
			fmt.Fprintf(out, "\n(%d Top-K hits; may contain multiple fragments of one note, not an exhaustive list)\n", len(results))
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Max results to return (max 10)")
	cmd.Flags().StringVar(&kb, "kb", "", "Limit search to a knowledge base (topic_id)")
	return cmd
}

func isSearchTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}

func scoreText(score *float64) string {
	if score == nil {
		return "—"
	}
	return fmt.Sprintf("%.2f", *score)
}
