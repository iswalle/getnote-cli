package setup

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

const defaultCLIPackage = "@getnote/cli@latest"

var workBuddySkillNames = []string{
	"getnote-auth",
	"getnote-kb",
	"getnote-note",
	"getnote-search",
	"getnote-tag",
}

type result struct {
	Success         bool     `json:"success"`
	Targets         []string `json:"targets"`
	InstalledCLI    bool     `json:"installed_cli"`
	InstalledSkills bool     `json:"installed_skills"`
	RestartRequired []string `json:"restart_required,omitempty"`
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
  getnote setup --target workbuddy
  getnote setup --target codex --target claude-code
  getnote setup --dry-run -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if scope != "global" && scope != "project" {
				return fmt.Errorf("不支持的安装范围: %s", scope)
			}
			resolved, err := resolveTargets(targets)
			if err != nil {
				return err
			}
			if len(resolved) == 0 {
				return fmt.Errorf("未检测到可自动配置的平台；支持 workbuddy、codex、claude-code、cursor")
			}

			out, _ := cmd.Root().PersistentFlags().GetString("output")
			if dryRun {
				return writeResult(cmd, out, result{
					Success:       true,
					Targets:       resolved,
					Authenticated: config.Get().IsLoggedIn(),
					Next:          setupPlan(resolved, scope),
				})
			}

			// `npx @getnote/cli setup` runs from a disposable npm cache. Install the
			// real package first so getnote/gnote both resolve from a stable path.
			installCLI := exec.Command("npm", "install", "-g", cliPackage())
			installCLI.Stdin = cmd.InOrStdin()
			configureInstallProcess(installCLI, out, cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err := installCLI.Run(); err != nil {
				return fmt.Errorf("安装 GetNote CLI 失败: %w", err)
			}

			if source == "" {
				var err error
				source, err = globalPackageDir()
				if err != nil {
					return err
				}
			}

			agentTargets := standardAgentTargets(resolved)
			if len(agentTargets) > 0 {
				installArgs := []string{"-y", "skills", "add", source, "-y"}
				if scope == "global" {
					installArgs = append(installArgs, "-g")
				}
				installArgs = append(installArgs, "--agent")
				installArgs = append(installArgs, agentTargets...)
				install := exec.Command("npx", installArgs...)
				install.Stdin = cmd.InOrStdin()
				configureInstallProcess(install, out, cmd.OutOrStdout(), cmd.ErrOrStderr())
				if err := install.Run(); err != nil {
					return fmt.Errorf("安装 GetNote Skills 失败: %w", err)
				}
			}

			restartRequired := []string{}
			if contains(resolved, "workbuddy") {
				if err := installWorkBuddySkills(filepath.Join(source, "skills"), workBuddySkillsDir()); err != nil {
					return fmt.Errorf("安装 WorkBuddy Skills 失败: %w", err)
				}
				restartRequired = append(restartRequired, "workbuddy")
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
			return writeResult(cmd, out, result{Success: true, Targets: resolved, InstalledCLI: true, InstalledSkills: true, RestartRequired: restartRequired, Authenticated: authed, Next: next})
		},
	}
	cmd.Flags().StringSliceVar(&targets, "target", nil, "目标平台，可重复或用逗号分隔: workbuddy,codex,claude-code,cursor")
	cmd.Flags().StringVar(&scope, "scope", "global", "Skill 安装范围: global 或 project")
	cmd.Flags().StringVar(&source, "skill-source", "", "Skill 来源；默认使用刚安装的全局 CLI 内置 Skills，本地验收可传仓库目录")
	cmd.Flags().BoolVar(&skipAuth, "skip-auth", false, "跳过首次授权")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "仅输出将执行的操作")
	return cmd
}

func setupPlan(targets []string, scope string) string {
	steps := []string{"npm install -g " + cliPackage()}
	agents := standardAgentTargets(targets)
	if len(agents) > 0 {
		args := []string{"npx -y skills add <全局 @getnote/cli 目录> -y"}
		if scope == "global" {
			args = append(args, "-g")
		}
		steps = append(steps, strings.Join(args, " ")+" --agent "+strings.Join(agents, " "))
	}
	if contains(targets, "workbuddy") {
		steps = append(steps, "复制 5 个 GetNote Skills 到 "+workBuddySkillsDir()+" 并重启 WorkBuddy")
	}
	return strings.Join(steps, " && ")
}

func cliPackage() string {
	// Internal override for isolated release validation. End users always get latest.
	if configured := strings.TrimSpace(os.Getenv("GETNOTE_CLI_PACKAGE")); configured != "" {
		return configured
	}
	return defaultCLIPackage
}

func standardAgentTargets(targets []string) []string {
	result := make([]string, 0, len(targets))
	for _, target := range targets {
		if agent, ok := agentNames[target]; ok {
			result = append(result, agent)
		}
	}
	return result
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func globalPackageDir() (string, error) {
	out, err := exec.Command("npm", "root", "-g").Output()
	if err != nil {
		return "", fmt.Errorf("无法定位全局 npm 目录: %w", err)
	}
	dir := filepath.Join(strings.TrimSpace(string(out)), "@getnote", "cli")
	if info, err := os.Stat(filepath.Join(dir, "skills")); err != nil || !info.IsDir() {
		return "", fmt.Errorf("全局 GetNote CLI 缺少内置 Skills: %s", dir)
	}
	return dir, nil
}

func workBuddySkillsDir() string {
	if configured := strings.TrimSpace(os.Getenv("GETNOTE_WORKBUDDY_SKILLS_DIR")); configured != "" {
		return configured
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".workbuddy", "skills")
	}
	return filepath.Join(home, ".workbuddy", "skills")
}

func installWorkBuddySkills(sourceRoot, targetRoot string) error {
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return err
	}
	stagingRoot, err := os.MkdirTemp(targetRoot, ".getnote-skills-install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stagingRoot)

	for _, name := range workBuddySkillNames {
		source := filepath.Join(sourceRoot, name)
		if _, err := os.Stat(filepath.Join(source, "SKILL.md")); err != nil {
			return fmt.Errorf("%s 缺少 SKILL.md", source)
		}
		if err := copyDir(source, filepath.Join(stagingRoot, name)); err != nil {
			return err
		}
	}

	backupRoot, err := os.MkdirTemp(targetRoot, ".getnote-skills-backup-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(backupRoot)
	committed := []string{}
	rollback := func() {
		for index := len(committed) - 1; index >= 0; index-- {
			name := committed[index]
			target := filepath.Join(targetRoot, name)
			_ = os.RemoveAll(target)
			backup := filepath.Join(backupRoot, name)
			if _, err := os.Stat(backup); err == nil {
				_ = os.Rename(backup, target)
			}
		}
	}
	for _, name := range workBuddySkillNames {
		target := filepath.Join(targetRoot, name)
		backup := filepath.Join(backupRoot, name)
		if _, err := os.Stat(target); err == nil {
			if err := os.Rename(target, backup); err != nil {
				rollback()
				return err
			}
		} else if !os.IsNotExist(err) {
			rollback()
			return err
		}
		if err := os.Rename(filepath.Join(stagingRoot, name), target); err != nil {
			if _, backupErr := os.Stat(backup); backupErr == nil {
				_ = os.Rename(backup, target)
			}
			rollback()
			return err
		}
		committed = append(committed, name)
	}
	return nil
}

func copyDir(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if info.IsDir() {
			return os.MkdirAll(destination, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("不支持的 Skill 文件类型: %s", path)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputCloseErr := input.Close()
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputCloseErr != nil {
			return inputCloseErr
		}
		return closeErr
	})
}

func resolveTargets(values []string) ([]string, error) {
	set := map[string]bool{}
	for _, value := range values {
		for _, id := range strings.Split(value, ",") {
			id = strings.ToLower(strings.TrimSpace(id))
			if id == "" {
				continue
			}
			if _, ok := agentNames[id]; !ok && id != "workbuddy" {
				return nil, fmt.Errorf("不支持自动配置的平台: %s", id)
			}
			set[id] = true
		}
	}
	if len(set) == 0 {
		for _, item := range platform.Detect() {
			if item.Detected {
				if _, ok := agentNames[item.ID]; ok || item.ID == "workbuddy" {
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
	if contains(data.RestartRequired, "workbuddy") {
		fmt.Fprintln(cmd.OutOrStdout(), "请完全退出并重新打开 WorkBuddy，使新安装的 Skills 生效")
	}
	if data.Next != "" {
		fmt.Fprintln(cmd.OutOrStdout(), data.Next)
	}
	return nil
}
