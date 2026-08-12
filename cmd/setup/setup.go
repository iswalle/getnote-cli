package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/iswalle/getnote-cli/internal/config"
	"github.com/iswalle/getnote-cli/internal/platform"
	"github.com/spf13/cobra"
)

var agentNames = map[string]string{
	"codex":       "codex",
	"claude-code": "claude-code",
	"cursor":      "cursor",
}

type result struct {
	Success         bool     `json:"success"`
	Targets         []string `json:"targets"`
	InstalledSkills bool     `json:"installed_skills"`
	Authenticated   bool     `json:"authenticated"`
	Next            string   `json:"next,omitempty"`
}

func configureInstallProcess(command *exec.Cmd, output string, stdout, stderr interface{ Write([]byte) (int, error) }) {
	command.Stdout = stdout
	command.Stderr = stderr
	if output == "json" {
		// Keep stdout machine-readable; dependency installer progress is diagnostic output.
		command.Stdout = stderr
	}
}

// NewSetupCmd installs the bundled atomic skills into supported local AI hosts, then starts authorization.
func NewSetupCmd() *cobra.Command {
	var targets []string
	var scope, source string
	var skipAuth, dryRun bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "一次连接本机 AI 与得到大脑 / Set up supported local AI hosts",
		Example: `  getnote setup
  getnote setup --target codex --target claude-code
  getnote setup --dry-run -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if scope != "global" && scope != "project" {
				return fmt.Errorf("不支持的安装范围: %s", scope)
			}
			resolved, err := resolveTargets(targets)
			if err != nil {
				return err
			}
			if len(resolved) == 0 {
				return fmt.Errorf("未检测到可自动配置的平台；支持 codex、claude-code、cursor")
			}
			if source == "" {
				source = "iswalle/getnote-cli"
			}
			installArgs := []string{"-y", "skills", "add", source, "-y"}
			if scope == "global" {
				installArgs = append(installArgs, "-g")
			}
			installArgs = append(installArgs, "--agent")
			for _, target := range resolved {
				installArgs = append(installArgs, agentNames[target])
			}

			out, _ := cmd.Root().PersistentFlags().GetString("output")
			if dryRun {
				return writeResult(cmd, out, result{Success: true, Targets: resolved, Authenticated: config.Get().IsLoggedIn(), Next: "npx " + strings.Join(installArgs, " ")})
			}

			install := exec.Command("npx", installArgs...)
			install.Stdin = cmd.InOrStdin()
			configureInstallProcess(install, out, cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err := install.Run(); err != nil {
				return fmt.Errorf("安装 GetNote Skills 失败: %w", err)
			}

			authed := config.Get().IsLoggedIn()
			if !skipAuth && !authed {
				login := exec.Command(os.Args[0], "auth", "login")
				login.Stdin = cmd.InOrStdin()
				configureInstallProcess(login, out, cmd.OutOrStdout(), cmd.ErrOrStderr())
				if err := login.Run(); err != nil {
					return fmt.Errorf("得到大脑授权失败: %w", err)
				}
				authed = true
			}
			next := "说“帮我保存一条测试笔记”完成验证"
			if !authed {
				next = "运行 getnote auth login 完成授权"
			}
			return writeResult(cmd, out, result{Success: true, Targets: resolved, InstalledSkills: true, Authenticated: authed, Next: next})
		},
	}
	cmd.Flags().StringSliceVar(&targets, "target", nil, "目标平台，可重复或用逗号分隔: codex,claude-code,cursor")
	cmd.Flags().StringVar(&scope, "scope", "global", "Skill 安装范围: global 或 project")
	cmd.Flags().StringVar(&source, "skill-source", "", "Skill 来源；默认 iswalle/getnote-cli，本地验收可传仓库目录")
	cmd.Flags().BoolVar(&skipAuth, "skip-auth", false, "跳过首次授权")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "仅输出将执行的操作")
	return cmd
}

func resolveTargets(values []string) ([]string, error) {
	set := map[string]bool{}
	for _, value := range values {
		for _, id := range strings.Split(value, ",") {
			id = strings.ToLower(strings.TrimSpace(id))
			if id == "" {
				continue
			}
			if _, ok := agentNames[id]; !ok {
				return nil, fmt.Errorf("不支持自动配置的平台: %s", id)
			}
			set[id] = true
		}
	}
	if len(set) == 0 {
		for _, item := range platform.Detect() {
			if item.Detected {
				if _, ok := agentNames[item.ID]; ok {
					set[item.ID] = true
				}
			}
		}
	}
	result := make([]string, 0, len(set))
	for id := range set {
		result = append(result, id)
	}
	sort.Strings(result)
	return result, nil
}

func writeResult(cmd *cobra.Command, output string, data result) error {
	if output == "json" {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n✓ 已为 %s 安装得到大脑能力\n", strings.Join(data.Targets, "、"))
	if data.Next != "" {
		fmt.Fprintln(cmd.OutOrStdout(), data.Next)
	}
	return nil
}
