package note

import (
	"encoding/json"
	"fmt"
	"github.com/iswalle/getnote-cli/internal/client"
	"github.com/spf13/cobra"
)

func newOutlineCmd() *cobra.Command {
	return &cobra.Command{Use: "outline <id>", Short: "读取长笔记目录与 UTF-16 位置 / Read note outline", Args: cobra.ExactArgs(1), Example: "  getnote note outline 123 -o json", RunE: func(cmd *cobra.Command, args []string) error {
		r, err := client.New("").NoteOutline(args[0])
		if err != nil {
			return err
		}
		return writeSection(cmd, r)
	}}
}
func newSectionCmd() *cobra.Command {
	var offset, limit int
	var keyword, version string
	cmd := &cobra.Command{Use: "section <id>", Short: "按目录位置或字面关键词读取节选 / Read note section", Long: "先 outline，再按返回的 offset 和 version 读取。偏移单位 UTF-16；普通读取默认最多8000单元。关键词不传 limit 时搜索到文末，传 limit 只搜索指定窗口。失败不自动读取全文。", Args: cobra.ExactArgs(1), Example: "  getnote note section 123 --keyword '绩效|晋升' -o json\n  getnote note section 123 --offset 8000 --limit 8000 --version '<返回的version>' -o json", RunE: func(cmd *cobra.Command, args []string) error {
		o := client.NoteSectionOptions{Keyword: keyword, Version: version}
		if cmd.Flags().Changed("offset") {
			o.Offset = &offset
		}
		if cmd.Flags().Changed("limit") {
			o.Limit = &limit
		}
		r, err := client.New("").NoteSection(args[0], o)
		if err != nil {
			return err
		}
		return writeSection(cmd, r)
	}}
	cmd.Flags().IntVar(&offset, "offset", 0, "返回的 UTF-16 offset；不要自行按字符数计算")
	cmd.Flags().IntVar(&limit, "limit", 8000, "UTF-16 读取上限 / 关键词搜索窗口，1..30000；不传保留服务端语义")
	cmd.Flags().StringVar(&keyword, "keyword", "", "字面关键词；用 | 分隔备选词，不是语义搜索")
	cmd.Flags().StringVar(&version, "version", "", "上次返回的版本；变化时重新读取目录")
	return cmd
}
func writeSection(cmd *cobra.Command, r *client.NoteSectionResponse) error {
	if outputFormat(cmd) == "json" {
		e := json.NewEncoder(cmd.OutOrStdout())
		e.SetIndent("", "  ")
		return e.Encode(r)
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), r.Data.Text)
	return err
}
