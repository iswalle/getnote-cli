package notes

import (
	"encoding/json"
	"fmt"
	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/spf13/cobra"
)

func newBatchCmd() *cobra.Command {
	var fields []string
	var budget int
	cmd := &cobra.Command{Use: "batch <note-id> [note-id...]", Short: "按预算批量读取个人笔记正文 / Read up to 20 distinct personal notes", Args: cobra.RangeArgs(1, 200), RunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("max-total-chars") && budget < 1 {
			return fmt.Errorf("--max-total-chars must be 1..64000")
		}
		r, err := client.New("").NotesBatch(client.BatchDetailOptions{IDs: args, Fields: fields, MaxTotalChars: budget})
		if err != nil {
			return err
		}
		if outputFormat(cmd) == "json" {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(r)
		}
		for _, item := range r.Data.Items {
			fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", item.NoteID, item.Status)
			if len(item.Note) > 0 {
				var note struct {
					Content  *string `json:"content"`
					Quote    *string `json:"ref_content"`
					Original *struct {
						Content string `json:"content"`
						Source  string `json:"source"`
					} `json:"original"`
				}
				if err = json.Unmarshal(item.Note, &note); err != nil {
					return err
				}
				if note.Content != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Content / Comment:\n%s\n", *note.Content)
				}
				if note.Quote != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Quote:\n%s\n", *note.Quote)
				}
				if note.Original != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Original (%s):\n%s\n", note.Original.Source, note.Original.Content)
				}
			}
		}
		if len(r.Data.PendingIDs) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Pending IDs: %v. Read individual full notes with getnote note <id>.\n", r.Data.PendingIDs)
		}
		return nil
	}}
	cmd.Example = "  getnote notes batch '100001' '100002' --fields original -o json\n  getnote notes batch '100001' --max-total-chars 32000 -o json"
	cmd.Flags().StringSliceVar(&fields, "fields", nil, "Requested fields: content,ref_content,reference,original; originals are opt-in")
	cmd.Flags().IntVar(&budget, "max-total-chars", 0, "Unicode body budget (1..64000, server default 32000); whole items may be deferred")
	return cmd
}
