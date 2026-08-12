package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestAllCommandsProvideCompleteHelp prevents a release from adding a command
// that cannot explain its usage to users and AI agents through -h/--help.
func TestAllCommandsProvideCompleteHelp(t *testing.T) {
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if command != rootCmd && !command.Hidden {
			if strings.TrimSpace(command.Short) == "" {
				t.Errorf("%s: missing Short description", command.CommandPath())
			}
			if command.Runnable() && strings.TrimSpace(command.Example) == "" {
				t.Errorf("%s: runnable command missing Example", command.CommandPath())
			}

			var output bytes.Buffer
			command.InitDefaultHelpFlag()
			command.SetOut(&output)
			command.SetErr(&output)
			if err := command.Help(); err != nil {
				t.Errorf("%s: help failed: %v", command.CommandPath(), err)
			} else {
				help := output.String()
				if !strings.Contains(help, "Usage:") {
					t.Errorf("%s: help missing Usage", command.CommandPath())
				}
				if !strings.Contains(help, "-h, --help") {
					t.Errorf("%s: help missing -h/--help flag", command.CommandPath())
				}
			}
		}
		for _, child := range command.Commands() {
			visit(child)
		}
	}

	visit(rootCmd)
}

// TestBundledSkillsOnlyReferenceRealCommands keeps the five bundled Skills
// aligned with the Cobra command tree instead of maintaining a second command
// specification in Markdown.
func TestBundledSkillsOnlyReferenceRealCommands(t *testing.T) {
	available := make(map[string]struct{})
	aliases := make(map[string]string)
	var collect func(*cobra.Command)
	collect = func(command *cobra.Command) {
		if command != rootCmd && !command.Hidden {
			path := strings.TrimPrefix(command.CommandPath(), rootCmd.Name()+" ")
			available[path] = struct{}{}
			parent := strings.TrimSuffix(path, command.Name())
			for _, alias := range command.Aliases {
				aliases[strings.TrimSpace(parent+alias)] = path
			}
		}
		for _, child := range command.Commands() {
			collect(child)
		}
	}
	collect(rootCmd)

	files, err := filepath.Glob(filepath.Join("..", "skills", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no bundled Skills found")
	}

	commandPattern := regexp.MustCompile("`(?:getnote|gnote)\\s+([^`]+)`")
	for _, file := range files {
		content, readErr := os.ReadFile(file)
		if readErr != nil {
			t.Errorf("%s: %v", file, readErr)
			continue
		}

		matches := commandPattern.FindAllStringSubmatch(string(content), -1)
		if len(matches) == 0 {
			t.Errorf("%s: no getnote command navigation found", file)
			continue
		}

		for _, match := range matches {
			fields := strings.Fields(match[1])
			found := false
			for length := min(3, len(fields)); length > 0; length-- {
				candidate := strings.Join(fields[:length], " ")
				if _, ok := available[candidate]; ok {
					found = true
					break
				}
				if canonical, ok := aliases[candidate]; ok {
					if _, exists := available[canonical]; exists {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("%s: documented command does not exist: getnote %s", file, match[1])
			}
		}
	}
}

func TestCompactCommandAliases(t *testing.T) {
	want := map[string][]string{
		"kb directories":      {"dir", "dirs"},
		"kb directory-create": {"mkdir"},
		"kb directory-update": {"mvdir"},
		"kb directory-delete": {"rmdir"},
		"note quick-note":     {"quick"},
	}
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		path := strings.TrimPrefix(command.CommandPath(), rootCmd.Name()+" ")
		if aliases, ok := want[path]; ok {
			for _, alias := range aliases {
				if !contains(command.Aliases, alias) {
					t.Errorf("%s: missing alias %s", path, alias)
				}
			}
			delete(want, path)
		}
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	visit(rootCmd)
	for path := range want {
		t.Errorf("compact alias command not found: %s", path)
	}
}

func TestBundledSkillsCoverAllCapabilities(t *testing.T) {
	available := make(map[string]struct{})
	aliases := make(map[string]string)
	var collect func(*cobra.Command)
	collect = func(command *cobra.Command) {
		if command != rootCmd && !command.Hidden && command.Runnable() {
			path := strings.TrimPrefix(command.CommandPath(), rootCmd.Name()+" ")
			available[path] = struct{}{}
			parent := strings.TrimSuffix(path, command.Name())
			for _, alias := range command.Aliases {
				aliases[strings.TrimSpace(parent+alias)] = path
			}
		}
		for _, child := range command.Commands() {
			collect(child)
		}
	}
	collect(rootCmd)

	mentioned := make(map[string]struct{})
	files, err := filepath.Glob(filepath.Join("..", "skills", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	commandPattern := regexp.MustCompile("`(?:getnote|gnote)\\s+([^`]+)`")
	for _, file := range files {
		content, readErr := os.ReadFile(file)
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, match := range commandPattern.FindAllStringSubmatch(string(content), -1) {
			fields := strings.Fields(match[1])
			for length := min(3, len(fields)); length > 0; length-- {
				candidate := strings.Join(fields[:length], " ")
				if _, ok := available[candidate]; ok {
					mentioned[candidate] = struct{}{}
					break
				}
				if canonical, ok := aliases[candidate]; ok {
					mentioned[canonical] = struct{}{}
					break
				}
			}
		}
	}
	for command := range available {
		if _, ok := mentioned[command]; ok {
			continue
		}
		t.Errorf("CLI capability is not routed by a bundled Skill: %s", command)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// TestHistoricalSafetyFlags keeps high-risk operations guarded in the CLI even
// when a host ignores or truncates Skill instructions.
func TestHistoricalSafetyFlags(t *testing.T) {
	want := map[string]string{
		"note update": "yes",
		"note delete": "yes",
		"note share":  "yes",
		"kb remove":   "yes",
	}
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		path := strings.TrimPrefix(command.CommandPath(), rootCmd.Name()+" ")
		if flag, ok := want[path]; ok {
			if command.Flags().Lookup(flag) == nil {
				t.Errorf("%s: missing --%s safety flag", path, flag)
			}
			delete(want, path)
		}
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	visit(rootCmd)
	for path := range want {
		t.Errorf("safety command not found: %s", path)
	}
}
