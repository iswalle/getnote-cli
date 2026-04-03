package quota

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewQuotaCmd returns the quota command.
func NewQuotaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "quota",
		Short: "查看 API 配额用量 / Show API quota usage",
		Example: `  getnote quota
  getnote quota -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
			resp, err := c.QuotaGet()
			if err != nil {
				return err
			}
			out, _ := cmd.Root().PersistentFlags().GetString("output")
			if out == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			d := resp.Data
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Type", "Period", "Used", "Limit", "Remaining", "Reset At"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)

			fmtTime := func(ts int64) string {
				return time.Unix(ts, 0).Format("2006-01-02")
			}

			rows := []struct {
				name   string
				period string
				limit  int
				used   int
				remain int
				reset  int64
			}{
				{"read", "daily", d.Read.Daily.Limit, d.Read.Daily.Used, d.Read.Daily.Remaining, d.Read.Daily.ResetAt},
				{"read", "monthly", d.Read.Monthly.Limit, d.Read.Monthly.Used, d.Read.Monthly.Remaining, d.Read.Monthly.ResetAt},
				{"write", "daily", d.Write.Daily.Limit, d.Write.Daily.Used, d.Write.Daily.Remaining, d.Write.Daily.ResetAt},
				{"write", "monthly", d.Write.Monthly.Limit, d.Write.Monthly.Used, d.Write.Monthly.Remaining, d.Write.Monthly.ResetAt},
				{"write_note", "daily", d.WriteNote.Daily.Limit, d.WriteNote.Daily.Used, d.WriteNote.Daily.Remaining, d.WriteNote.Daily.ResetAt},
				{"write_note", "monthly", d.WriteNote.Monthly.Limit, d.WriteNote.Monthly.Used, d.WriteNote.Monthly.Remaining, d.WriteNote.Monthly.ResetAt},
			}
			for _, r := range rows {
				table.Append([]string{
					r.name, r.period,
					fmt.Sprintf("%d", r.used),
					fmt.Sprintf("%d", r.limit),
					fmt.Sprintf("%d", r.remain),
					fmtTime(r.reset),
				})
			}
			table.Render()
			return nil
		},
	}
}
