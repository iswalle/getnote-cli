package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"regexp"

	"github.com/iswalle/getnote-cli/internal/client"
	oss "github.com/iswalle/getnote-cli/internal/upload"
	"github.com/spf13/cobra"
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
		metadata, err := readUploadMetadata(r)
		if err != nil {
			return err
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

// Accept exactly one bounded metadata object, never file bytes or trailing JSON.
func readUploadMetadata(r io.Reader) (oss.Result, error) {
	var result oss.Result
	b, err := io.ReadAll(io.LimitReader(r, 65537))
	if err != nil || len(b) > 65536 {
		return result, errors.New("OSS result must be at most 65536 bytes")
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, errors.New("invalid OSS result JSON")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return result, errors.New("expected exactly one OSS result")
	}
	u, err := url.Parse(result.URL)
	if result.Stage != "oss_uploaded" || result.FileName == "" || result.FileType == "" || result.Size <= 0 || !regexp.MustCompile(`^[a-fA-F0-9]{32}$`).MatchString(result.MD5) || err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return result, errors.New("expected successful getnote upload result")
	}
	return result, nil
}
