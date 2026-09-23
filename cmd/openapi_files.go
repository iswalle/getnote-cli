package cmd

import (
	"encoding/json"
	"errors"
	"github.com/iswalle/getnote-cli/internal/client"
	oss "github.com/iswalle/getnote-cli/internal/upload"
	"github.com/spf13/cobra"
	"io"
	"os"
)

func addFileAndReportCommands() {
	for _, spec := range []struct {
		use, description, example string
		argc                      int
		run                       func(*client.Client, []string) (*client.ResourceResponse, error)
	}{
		{"marks <note_id>", "读取标记；与 Timeline 独立", "getnote marks 1922071641760757698 -o json", 1, func(c *client.Client, a []string) (*client.ResourceResponse, error) { return c.NoteMarks(a[0]) }},
		{"sprouts <YYYY-MM>", "列出指定月份的发芽报告", "getnote sprouts 2026-09 -o json", 1, func(c *client.Client, a []string) (*client.ResourceResponse, error) { return c.Sprouts(a[0]) }},
		{"sprout <id>", "读取发芽报告原文", "getnote sprout report-alias -o json", 1, func(c *client.Client, a []string) (*client.ResourceResponse, error) { return c.Sprout(a[0]) }},
		{"file-capabilities", "查询知识库文件格式及大小页数限制", "getnote file-capabilities -o json", 0, func(c *client.Client, a []string) (*client.ResourceResponse, error) { return c.FileCapabilities() }},
		{"file-token <extension>", "获取临时OSS文件token；仅供机器传递，不贴到聊天", "getnote file-token HTML -o json > upload-token.json", 1, func(c *client.Client, a []string) (*client.ResourceResponse, error) { return c.FileToken(a[0]) }},
	} {
		spec := spec
		command := &cobra.Command{Use: spec.use, Short: spec.description, Example: "  " + spec.example, Args: cobra.ExactArgs(spec.argc), RunE: func(cmd *cobra.Command, args []string) error {
			var result *client.ResourceResponse
			var err error
			if cmd.Name() == "sprouts" {
				sinceID, _ := cmd.Flags().GetString("since-id")
				limit, _ := cmd.Flags().GetInt("limit")
				if limit < 1 || limit > 20 {
					return errors.New("limit must be between 1 and 20")
				}
				result, err = client.New("").SproutsPage(args[0], sinceID, limit)
			} else {
				result, err = spec.run(client.New(""), args)
			}
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}}
		if command.Name() == "sprouts" {
			command.Flags().String("since-id", "", "上一页返回的游标（字符串）")
			command.Flags().Int("limit", 20, "每页条数（1–20）")
		}
		rootCmd.AddCommand(command)
	}
	var metadataFile string
	cmd := &cobra.Command{Use: "file-add <topic_id> <directory_id>", Short: "将OSS上传结果提交知识库；随后查询目录确认最终状态", Example: "  getnote file-add topic-alias 14 --metadata-file upload-result.json -o json", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		var r io.Reader = cmd.InOrStdin()
		if metadataFile != "-" {
			f, err := os.Open(metadataFile)
			if err != nil {
				return err
			}
			defer f.Close()
			r = f
		}
		var metadata oss.Result
		if err := json.NewDecoder(io.LimitReader(r, 65536)).Decode(&metadata); err != nil {
			return errors.New("invalid OSS result JSON")
		}
		if metadata.Stage != "oss_uploaded" || metadata.URL == "" || metadata.MD5 == "" {
			return errors.New("expected successful getnote upload result")
		}
		result, err := client.New("").FileAdd(map[string]string{"topic_id": args[0], "directory_id": args[1], "file_name": metadata.FileName, "file_type": metadata.FileType, "md5": metadata.MD5, "url": metadata.URL})
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
	cmd.Flags().StringVar(&metadataFile, "metadata-file", "-", "getnote upload 的JSON输出文件，-从stdin读取")
	rootCmd.AddCommand(cmd)
}
