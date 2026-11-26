package client

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerationStatusRoundTrip(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantStatus string
		wantNotice string
		wantKey    bool
	}{
		{"generating", `{"note_id":"1","title":"t","content":"","generation_status":"generating"}`, "generating", "生成中", true},
		{"completed", `{"note_id":"1","title":"t","content":"x","generation_status":"completed"}`, "completed", "", true},
		{"failed", `{"note_id":"1","title":"t","generation_status":"failed"}`, "failed", "生成失败", true},
		{"future value", `{"note_id":"1","title":"t","generation_status":"queued_v2"}`, "queued_v2", "queued_v2", true},
		{"legacy server omits field", `{"note_id":"1","title":"t","content":"x"}`, "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var n Note
			if err := json.Unmarshal([]byte(tc.input), &n); err != nil {
				t.Fatal(err)
			}
			var kb KBNote
			if err := json.Unmarshal([]byte(tc.input), &kb); err != nil {
				t.Fatal(err)
			}
			for label, f := range map[string]DiscoveryNoteFields{"note": n.DiscoveryNoteFields, "kb": kb.DiscoveryNoteFields} {
				if f.GenerationStatus != tc.wantStatus || f.GenerationNotice() != tc.wantNotice {
					t.Fatalf("%s: status=%q notice=%q", label, f.GenerationStatus, f.GenerationNotice())
				}
			}
			for label, v := range map[string]any{"note": n, "kb": kb, "note_list": NoteListData{Notes: []Note{n}}, "detail": NoteGetData{Note: n}} {
				out, err := json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
				has := strings.Contains(string(out), `"generation_status"`)
				if has != tc.wantKey {
					t.Fatalf("%s: generation_status present=%v, want %v: %s", label, has, tc.wantKey, out)
				}
				if tc.wantKey && !strings.Contains(string(out), `"generation_status":"`+tc.wantStatus+`"`) {
					t.Fatalf("%s: status not preserved: %s", label, out)
				}
			}
		})
	}
}

func TestTitleWithGeneration(t *testing.T) {
	if got := (DiscoveryNoteFields{GenerationStatus: "generating"}).TitleWithGeneration("标题"); got != "[生成中] 标题" {
		t.Fatalf("got %q", got)
	}
	for _, status := range []string{"", "completed"} {
		if got := (DiscoveryNoteFields{GenerationStatus: status}).TitleWithGeneration("标题"); got != "标题" {
			t.Fatalf("status %q: got %q", status, got)
		}
	}
}

func TestOmitContentKeepsGenerationStatus(t *testing.T) {
	var kb KBNote
	if err := json.Unmarshal([]byte(`{"note_id":"1","content":"half","generation_status":"generating"}`), &kb); err != nil {
		t.Fatal(err)
	}
	kb.OmitContent()
	out, _ := json.Marshal(kb)
	if !strings.Contains(string(out), `"generation_status":"generating"`) || strings.Contains(string(out), `"half"`) {
		t.Fatalf("unexpected: %s", out)
	}
}
