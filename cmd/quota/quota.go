package quota

import (
	"encoding/json"
	"fmt"

	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewQuotaCmd returns the quota command.
func NewQuotaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "quota",
		Short: "查看 API 配额使用情况 / Show API quota usage",
		Example: `  getnote quota
  getnote quota -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New("")
			resp, err := c.QuotaGet()
			if err != nil {
				return err
			}

			if outputFormat(cmd) == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Name", "Used", "Total", "Reset At"})
			table.SetBorder(false)
			table.SetAutoWrapText(false)
			for _, q := range resp.Data.Quotas {
				table.Append([]string{
					q.Name,
					fmt.Sprintf("%d", q.Used),
					fmt.Sprintf("%d", q.Total),
					q.ResetTime,
				})
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
