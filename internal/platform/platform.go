package platform

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Info describes whether a supported AI host is available on this machine.
type Info struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Detected   bool   `json:"detected"`
	Executable string `json:"executable,omitempty"`
	AppPath    string `json:"app_path,omitempty"`
	SkillPath  string `json:"skill_path,omitempty"`
}

type definition struct {
	id       string
	name     string
	commands []string
	apps     []string
	paths    []string
	skillDir string
}

var definitions = []definition{
	{id: "workbuddy", name: "WorkBuddy", apps: []string{"WorkBuddy.app"}},
	{id: "openclaw", name: "OpenClaw（小龙虾）", commands: []string{"openclaw"}, apps: []string{"OpenClaw.app"}},
	{id: "qclaw", name: "QClaw", apps: []string{"QClaw.app"}},
	{id: "codex", name: "Codex", commands: []string{"codex"}, apps: []string{"Codex.app"}, skillDir: ".agents/skills"},
	{id: "claude-code", name: "Claude Code", commands: []string{"claude"}, apps: []string{"Claude.app"}, skillDir: ".claude/skills"},
	{id: "cursor", name: "Cursor", commands: []string{"cursor"}, apps: []string{"Cursor.app"}, skillDir: ".agents/skills"},
	{id: "gemini-cli", name: "Gemini CLI", commands: []string{"gemini"}, paths: []string{".gemini"}, skillDir: ".agents/skills"},
	{id: "github-copilot", name: "GitHub Copilot", commands: []string{"copilot"}, paths: []string{".copilot"}, skillDir: ".agents/skills"},
	{id: "windsurf", name: "Windsurf", commands: []string{"windsurf"}, apps: []string{"Windsurf.app"}, paths: []string{".codeium/windsurf"}, skillDir: ".codeium/windsurf/skills"},
	{id: "opencode", name: "OpenCode", commands: []string{"opencode"}, paths: []string{".config/opencode"}, skillDir: ".agents/skills"},
	{id: "cline", name: "Cline", apps: []string{"Cline.app"}, paths: []string{".cline"}, skillDir: ".agents/skills"},
}

// Detect returns every supported host, including hosts that are not installed.
func Detect() []Info {
	home, _ := os.UserHomeDir()
	result := make([]Info, 0, len(definitions))
	for _, item := range definitions {
		info := Info{ID: item.id, Name: item.name}
		if item.skillDir != "" {
			info.SkillPath = filepath.Join(home, filepath.FromSlash(item.skillDir))
		}
		for _, command := range item.commands {
			if path, err := exec.LookPath(command); err == nil {
				if commandWorks(path) {
					info.Detected = true
					info.Executable = path
					break
				}
			}
		}
		if runtime.GOOS == "darwin" {
			for _, app := range item.apps {
				path := filepath.Join("/Applications", app)
				if _, err := os.Stat(path); err == nil {
					info.Detected = true
					info.AppPath = path
					break
				}
			}
		}
		for _, relative := range item.paths {
			if _, err := os.Stat(filepath.Join(home, filepath.FromSlash(relative))); err == nil {
				info.Detected = true
				break
			}
		}
		result = append(result, info)
	}
	return result
}

func commandWorks(path string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// Find returns platform information by its stable ID.
func Find(id string) (Info, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, info := range Detect() {
		if info.ID == id {
			return info, true
		}
	}
	return Info{}, false
}
