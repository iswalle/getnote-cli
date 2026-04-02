package search

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/iswalle/getnote-cli/internal/ui"
	"github.com/spf13/cobra"
)

var cols = []ui.ColSpec{
	{Value: "ID", Width: 20},
	{Value: "Title", Width: 40},
	{Value: "Type", Width: 10},
	{Value: "Created", Width: 19},
	{Value: "Content", Width: 50},
}

const sep = "  "

// NewSearchCmd returns the search command.
func NewSearchCmd() *cobra.Command {
	var limit int
	var kb string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Semantic search across notes",
		Args:  cobra.MinimumNArgs(1),
		Example: `  getnote search "大模型 API"
  getnote search "RAG" --kb qnNX75j0
  getnote search "机器学习" --limit 5 -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			c := client.New(envTarget(cmd))

			var resp *client.NoteSearchResponse
			var err error

			if kb != "" {
				resp, err = c.KBSearch(kb, query, limit)
			} else {
				resp, err = c.NoteSearch(query, limit)
			}
			if err != nil {
				return ui.FriendlyError(err)
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			out := cmd.OutOrStdout()
			results := resp.Data.Results
			if len(results) == 0 {
				results = resp.Results // fallback
			}
			if len(results) == 0 {
				fmt.Fprintln(out, "No results found.")
				return nil
			}

			fmt.Fprint(out, ui.PrintHeader(cols, sep))
			fmt.Fprint(out, ui.DividerLine(cols, sep))
			for _, r := range results {
				row := []ui.ColSpec{
					{Value: r.NoteID, Width: cols[0].Width},
					{Value: r.Title, Width: cols[1].Width},
					{Value: r.NoteType, Width: cols[2].Width},
					{Value: r.CreatedAt, Width: cols[3].Width},
					{Value: ui.Truncate(r.Content, cols[4].Width), Width: cols[4].Width},
				}
				fmt.Fprint(out, ui.PrintRow(row, sep))
			}
			fmt.Fprintf(out, "\n(%d results)\n", len(results))
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Max results to return (max 10)")
	cmd.Flags().StringVar(&kb, "kb", "", "Limit search to a knowledge base (topic_id)")
	return cmd
}

func outputFormat(cmd *cobra.Command) string {
	f, _ := cmd.Root().PersistentFlags().GetString("output")
	return f
}

func envTarget(cmd *cobra.Command) string {
	e, _ := cmd.Root().PersistentFlags().GetString("env")
	return e
}
