package platform

import "testing"

func TestDetectReturnsStablePlatformIDs(t *testing.T) {
	got := Detect()
	want := map[string]bool{
		"workbuddy":   false,
		"qclaw":       false,
		"codex":       false,
		"claude-code": false,
		"cursor":      false,
		"openclaw":    false,
		"gemini-cli":  false, "github-copilot": false, "windsurf": false, "opencode": false,
		"cline": false, "continue": false, "roo": false, "kilo": false, "trae": false, "trae-cn": false,
		"qoder": false, "qoder-cn": false, "qwen-code": false, "kimi-code-cli": false,
		"goose": false, "zed": false, "warp": false, "amp": false, "augment": false, "droid": false,
	}
	for _, item := range got {
		if _, ok := want[item.ID]; !ok {
			t.Fatalf("unexpected platform id %q", item.ID)
		}
		want[item.ID] = true
	}
	for id, seen := range want {
		if !seen {
			t.Fatalf("missing platform id %q", id)
		}
	}
}

func TestAutomaticallyInstalledAgentsDeclareSkillPaths(t *testing.T) {
	marketplace := map[string]bool{"qclaw": true, "openclaw": true, "workbuddy": true}
	for _, item := range Detect() {
		if marketplace[item.ID] {
			continue
		}
		if item.SkillPath == "" {
			t.Errorf("%s is missing its inspectable Skill path", item.ID)
		}
	}
}
