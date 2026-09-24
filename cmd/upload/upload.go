package upload

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	oss "github.com/iswalle/getnote-cli/internal/upload"
	"github.com/spf13/cobra"
)

func NewUploadCmd() *cobra.Command {
	var tokenFile string
	var maxBytes int64
	cmd := &cobra.Command{
		Use:     "upload <file>",
		Short:   "仅直传本地文件到 OSS，无需 CLI 登录；入库仍由已授权 MCP 完成",
		Example: "  getnote upload ./report.txt --token-file ./upload-token.json --max-size-bytes 10485760\n  getnote upload ./report.txt --max-size-bytes 10485760 < ./upload-token.json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var r io.Reader = cmd.InOrStdin()
			if tokenFile != "-" {
				f, err := os.Open(tokenFile)
				if err != nil {
					return err
				}
				defer f.Close()
				r = f
			}
			data, err := io.ReadAll(io.LimitReader(r, 65537))
			if err != nil || len(data) > 65536 {
				return errors.New("could not read upload token (maximum 64KB)")
			}
			var token oss.Token
			if err := json.Unmarshal(data, &token); err != nil {
				return errors.New("invalid upload token JSON")
			}
			if token.PutURL == "" {
				var envelope struct {
					Data oss.Token `json:"data"`
				}
				if json.Unmarshal(data, &envelope) == nil {
					token = envelope.Data
				}
			}
			result, err := oss.File(cmd.Context(), args[0], token, maxBytes)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	}
	cmd.Flags().StringVar(&tokenFile, "token-file", "-", "MCP 上传 token JSON 文件；- 从 stdin 读取，禁止把 token 放入命令参数")
	cmd.Flags().Int64Var(&maxBytes, "max-size-bytes", 0, "该格式能力接口返回的 max_size_bytes")
	return cmd
}
