package setup

import (
	"bytes"
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
	"codex":          "codex",
	"claude-code":    "claude-code",
	"cursor":         "cursor",
	"gemini-cli":     "gemini-cli",
	"github-copilot": "github-copilot",
	"windsurf":       "windsurf",
	"opencode":       "opencode",
	"cline":          "cline",
	"continue":       "continue",
	"roo":            "roo",
	"kilo":           "kilo",
	"trae":           "trae",
	"trae-cn":        "trae-cn",
	"qoder":          "qoder",
	"qoder-cn":       "qoder-cn",
	"qwen-code":      "qwen-code",
	"kimi-code-cli":  "kimi-code-cli",
	"goose":          "goose",
	"zed":            "zed",
	"warp":           "warp",
	"amp":            "amp",
	"augment":        "augment",
	"droid":          "droid",
}

var platformNames = map[string]string{
	"workbuddy": "WorkBuddy", "codex": "Codex", "claude-code": "Claude Code", "cursor": "Cursor",
	"qclaw": "QClaw", "openclaw": "OpenClaw",
	"gemini-cli": "Gemini CLI", "github-copilot": "GitHub Copilot",
	"windsurf": "Windsurf", "opencode": "OpenCode", "cline": "Cline", "continue": "Continue",
	"roo": "Roo Code", "kilo": "Kilo Code", "trae": "Trae", "trae-cn": "Trae CN",
	"qoder": "Qoder", "qoder-cn": "Qoder CN", "qwen-code": "Qwen Code", "kimi-code-cli": "Kimi Code CLI",
	"goose": "Goose", "zed": "Zed", "warp": "Warp", "amp": "Amp", "augment": "Augment", "droid": "Droid",
}

var marketplaceTargets = map[string]bool{"qclaw": true, "openclaw": true}

const clawHubURL = "https://clawhub.ai/iswalle/getnote"

const setupBanner = `
 ███████╗██╗  ██╗██╗██╗     ██╗     ███████╗
 ██╔════╝██║ ██╔╝██║██║     ██║     ██╔════╝
 ███████╗█████╔╝ ██║██║     ██║     ███████╗
 ╚════██║██╔═██╗ ██║██║     ██║     ╚════██║
 ███████║██║  ██╗██║███████╗███████╗███████║
 ╚══════╝╚═╝  ╚═╝╚═╝╚══════╝╚══════╝╚══════╝`

const defaultCLIPackage = "@getnote/cli@latest"

var workBuddySkillNames = []string{
	"getnote-auth",
	"getnote-kb",
	"getnote-note",
	"getnote-search",
	"getnote-tag",
}

type result struct {
	Success         bool             `json:"success"`
	Targets         []string         `json:"targets"`
	InstalledCLI    bool             `json:"installed_cli"`
	InstalledSkills bool             `json:"installed_skills"`
	RestartRequired []string         `json:"restart_required,omitempty"`
	Authenticated   bool             `json:"authenticated"`
	Platforms       []platformResult `json:"platforms"`
	NextActions     []nextAction     `json:"next_actions"`
	Next            string           `json:"next,omitempty"`
}

type platformResult struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	SkillsInstalled bool   `json:"skills_installed"`
	RestartRequired bool   `json:"restart_required"`
	Message         string `json:"message"`
}

type nextAction struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	URL         string `json:"url,omitempty"`
}

func configureInstallProcess(command *exec.Cmd, output string, stdout, stderr interface{ Write([]byte) (int, error) }) {
	command.Stdout = stdout
	command.Stderr = stderr
	if output == "json" {
		// Keep stdout machine-readable; dependency installer progress is diagnostic output.
		command.Stdout = stderr
	}
}

func runInstaller(command *exec.Cmd) (string, error) {
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	return strings.TrimSpace(output.String()), err
}

func installerError(label string, err error, details string) error {
	if details == "" {
		return fmt.Errorf("%s失败: %w", label, err)
	}
	return fmt.Errorf("%s失败: %w\n%s", label, err, details)
}

func writeProgress(cmd *cobra.Command, output, message string) {
	if output == "table" {
		fmt.Fprintln(cmd.OutOrStdout(), message)
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
			out, _ := cmd.Root().PersistentFlags().GetString("output")
			if dryRun {
				platforms, actions := setupPlatformResults(resolved, false)
				return writeResult(cmd, out, result{
					Success:       true,
					Targets:       resolved,
					Authenticated: config.Get().IsLoggedIn(),
					Platforms:     platforms,
					NextActions:   actions,
					Next:          setupPlan(resolved, scope),
				})
			}
			writeProgress(cmd, out, setupBanner)
			writeProgress(cmd, out, "\n正在安装得到大脑，请稍候…")

			// `npx @getnote/cli setup` runs from a disposable npm cache. Install the
			// real package first so getnote/gnote both resolve from a stable path.
			installCLI := exec.Command("npm", "install", "-g", cliPackage())
			installCLI.Stdin = cmd.InOrStdin()
			details, installErr := runInstaller(installCLI)
			if installErr != nil {
				return installerError("安装命令行工具", installErr, details)
			}

			localTargets := locallyManagedTargets(resolved)
			if source == "" && len(localTargets) > 0 {
				var err error
				source, err = globalPackageDir()
				if err != nil {
					return err
				}
			}

			agentTargets := standardAgentTargets(resolved)
			if len(agentTargets) > 0 {
				writeProgress(cmd, out, "正在为 "+displayNames(resolved, false)+" 安装 5 个 Skills…")
				installArgs := []string{"-y", "skills", "add", source, "-y"}
				if scope == "global" {
					installArgs = append(installArgs, "-g")
				}
				installArgs = append(installArgs, "--agent")
				installArgs = append(installArgs, agentTargets...)
				install := exec.Command("npx", installArgs...)
				install.Stdin = cmd.InOrStdin()
				details, installErr := runInstaller(install)
				if installErr != nil {
					return installerError("安装 AI Skills", installErr, details)
				}
			}

			restartRequired := []string{}
			if contains(resolved, "workbuddy") {
				writeProgress(cmd, out, "正在为 WorkBuddy 安装 5 个 Skills…")
				if err := installWorkBuddySkills(filepath.Join(source, "skills"), workBuddySkillsDir()); err != nil {
					return fmt.Errorf("安装 WorkBuddy Skills 失败: %w", err)
				}
				restartRequired = append(restartRequired, "workbuddy")
			}

			authed := config.Get().IsLoggedIn()
			if !skipAuth && !authed {
				writeProgress(cmd, out, "接下来请在浏览器中确认得到大脑授权…")
				login := exec.Command(os.Args[0], "auth", "login")
				login.Stdin = cmd.InOrStdin()
				configureInstallProcess(login, out, cmd.OutOrStdout(), cmd.ErrOrStderr())
				if err := login.Run(); err != nil {
					return fmt.Errorf("得到大脑授权失败: %w", err)
				}
				authed = true
			}
			platforms, actions := setupPlatformResults(resolved, true)
			next := "说“帮我保存一条测试笔记”完成验证"
			if !authed {
				next = "运行 getnote auth login 完成授权"
			}
			return writeResult(cmd, out, result{Success: true, Targets: resolved, InstalledCLI: true, InstalledSkills: len(localTargets) > 0, RestartRequired: restartRequired, Authenticated: authed, Platforms: platforms, NextActions: actions, Next: next})
		},
	}
	cmd.Flags().StringSliceVar(&targets, "target", nil, "通常无需填写；仅用于调试时指定平台 ID，可重复或用逗号分隔")
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
	for _, target := range targets {
		if marketplaceTargets[target] {
			steps = append(steps, "在 "+platformNames[target]+" 内确认 ClawHub 的得到大脑 Skill 已启用")
		}
	}
	return strings.Join(steps, " && ")
}

func locallyManagedTargets(targets []string) []string {
	result := []string{}
	for _, target := range targets {
		if target == "workbuddy" || agentNames[target] != "" {
			result = append(result, target)
		}
	}
	return result
}

func setupPlatformResults(targets []string, installed bool) ([]platformResult, []nextAction) {
	platforms := []platformResult{}
	actions := []nextAction{}
	for _, target := range targets {
		name := platformNames[target]
		switch {
		case marketplaceTargets[target]:
			platforms = append(platforms, platformResult{ID: target, Name: name, Status: "verify_in_platform", Message: "由平台管理 Skill，请在技能市场确认“得到大脑”已安装并启用"})
			actions = append(actions, nextAction{ID: "verify_" + target + "_skill", Description: "在 " + name + " 内确认“得到大脑”Skill 已安装并启用", URL: clawHubURL})
		case target == "workbuddy":
			status, message := "planned", "将安装 5 个 Skills；完成后需要重启 WorkBuddy"
			if installed {
				status, message = "installed", "5 个 Skills 已安装，重启 WorkBuddy 后生效"
			}
			platforms = append(platforms, platformResult{ID: target, Name: name, Status: status, SkillsInstalled: installed, RestartRequired: true, Message: message})
		default:
			status, message := "planned", "将安装 5 个 Skills"
			if installed {
				status, message = "installed", "5 个 Skills 已安装"
			}
			platforms = append(platforms, platformResult{ID: target, Name: name, Status: status, SkillsInstalled: installed, Message: message})
		}
	}
	if len(targets) == 0 {
		actions = append(actions, nextAction{ID: "open_supported_ai", Description: "打开支持的 AI 应用后重新运行 getnote setup"})
	}
	return platforms, actions
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

func displayNames(targets []string, includeMarketplace bool) string {
	names := []string{}
	for _, target := range targets {
		if target == "workbuddy" || (!includeMarketplace && marketplaceTargets[target]) {
			continue
		}
		names = append(names, platformNames[target])
	}
	return strings.Join(names, "、")
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
			if _, ok := platformNames[id]; !ok {
				return nil, fmt.Errorf("不支持的平台: %s", id)
			}
			set[id] = true
		}
	}
	if len(set) == 0 {
		for _, item := range platform.Detect() {
			if item.Detected {
				if _, ok := platformNames[item.ID]; ok {
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
	fmt.Fprintln(cmd.OutOrStdout(), "\n安装完成")
	if data.InstalledCLI {
		fmt.Fprintln(cmd.OutOrStdout(), "✓ 得到大脑命令行工具")
	}
	if data.Authenticated {
		fmt.Fprintln(cmd.OutOrStdout(), "✓ 得到大脑账号已连接")
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "→ 得到大脑账号尚未连接")
	}
	if len(data.Platforms) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "✓ 命令行工具和得到大脑账号已准备好")
		fmt.Fprintln(cmd.OutOrStdout(), "未检测到正在使用的 AI 应用；打开 AI 应用后再次运行这条安装命令即可")
	}
	for _, item := range data.Platforms {
		if item.Status == "installed" {
			fmt.Fprintf(cmd.OutOrStdout(), "✓ %s：%s\n", item.Name, item.Message)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "→ %s：%s\n", item.Name, item.Message)
		}
	}
	if len(data.NextActions) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\n还需完成：")
		for _, action := range data.NextActions {
			fmt.Fprintf(cmd.OutOrStdout(), "- %s", action.Description)
			if action.URL != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "：%s", action.URL)
			}
			fmt.Fprintln(cmd.OutOrStdout())
		}
	}
	if data.Next != "" {
		fmt.Fprintln(cmd.OutOrStdout(), data.Next)
	}
	return nil
}
