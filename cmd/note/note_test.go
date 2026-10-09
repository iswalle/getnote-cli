package note

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestDetailViewsHaveStableJSONEnvelope(t *testing.T) {
	t.Setenv("GETNOTE_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open/api/v1/resource/note/detail" || r.URL.Query().Get("id") != "1917808813705036914" {
			t.Fatalf("unexpected request %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"success":true,
			"data":{"note":{
				"id":"1917808813705036914",
				"note_id":"1917808813705036914",
				"title":"会议记录",
				"note_type":"audio",
				"content":"文字正文",
				"audio":{"original":"完整转写"},
				"quick_note":"快捷笔记",
				"attachments":[{"type":"image","url":"https://example.com/a.png"}],
				"timeline":{"items":[]},
				"chapter_timeline":{"source":"summary_markdown_rules","items":[{"start_ms":1000,"title":"第一章"}]},
				"meeting_todos":{"source":"summary_section","items":[]}
			}}
		}`)
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)

	tests := map[string]string{
		"original":    "original",
		"transcript":  "transcript",
		"attachments": "attachments",
		"timeline":    "timeline",
		"chapters":    "chapter_timeline",
		"quick-note":  "quick_note",
		"todos":       "meeting_todos",
	}
	for command, field := range tests {
		t.Run(command, func(t *testing.T) {
			root := &cobra.Command{Use: "getnote"}
			root.PersistentFlags().StringP("output", "o", "table", "")
			root.AddCommand(NewNoteCmd())
			root.SetArgs([]string{"note", command, "1917808813705036914", "-o", "json"})
			var output bytes.Buffer
			root.SetOut(&output)
			root.SetErr(&output)
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			var got struct {
				Success bool                   `json:"success"`
				Data    map[string]interface{} `json:"data"`
			}
			if err := json.Unmarshal(output.Bytes(), &got); err != nil {
				t.Fatalf("decode output: %v\n%s", err, output.String())
			}
			if !got.Success || got.Data["note_id"] != "1917808813705036914" || got.Data["title"] != "会议记录" || got.Data[field] == nil {
				t.Fatalf("unexpected output: %#v", got)
			}
			if command == "chapters" {
				chapter := got.Data[field].(map[string]interface{})
				items := chapter["items"].([]interface{})
				if len(items) != 1 || items[0].(map[string]interface{})["start_ms"] != float64(1000) {
					t.Fatal("chapter data lost")
				}
			}
		})
	}
}

func TestOriginalSelectsOnlyTheTypeSpecificSource(t *testing.T) {
	cases := []struct {
		name, noteJSON, wantOriginal, wantSource, wantError string
	}{
		{"reference with comment", `{"note_type":"ref","content":"personal comment","ref_content":"quoted passage"}`, "quoted passage", "ref_content", ""},
		{"reference without comment", `{"note_type":"ref","content":"","ref_content":"quoted passage"}`, "quoted passage", "ref_content", ""},
		{"reference without quote", `{"note_type":"ref","content":"personal comment","ref_content":""}`, "", "", "not available"},
		{"plain text", `{"note_type":"plain_text","content":"typed text","ref_content":"other"}`, "typed text", "content", ""},
		{"link", `{"note_type":"link","content":"AI summary","web_page":{"content":"web article"}}`, "web article", "web_page.content", ""},
		{"audio", `{"note_type":"audio","content":"AI summary","audio":{"original":"raw transcript"}}`, "raw transcript", "audio.original", ""},
		{"local audio", `{"note_type":"local_audio","content":"AI summary","audio":{"original":"local transcript"}}`, "local transcript", "audio.original", ""},
		{"unknown", `{"note_type":"unknown","content":"AI summary"}`, "", "", "unsupported"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GETNOTE_API_KEY", "test-key")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"success":true,"data":{"note":{"id":"123","note_id":"123","title":"Synthetic",%s}}}`, tc.noteJSON[1:len(tc.noteJSON)-1])
			}))
			defer server.Close()
			t.Setenv("GETNOTE_API_URL", server.URL)
			root := &cobra.Command{Use: "getnote", SilenceUsage: true, SilenceErrors: true}
			root.PersistentFlags().StringP("output", "o", "table", "")
			root.AddCommand(NewNoteCmd())
			root.SetArgs([]string{"note", "original", "123", "-o", "json"})
			var output bytes.Buffer
			root.SetOut(&output)
			root.SetErr(&output)
			err := root.Execute()
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				Data map[string]any `json:"data"`
			}
			if err := json.Unmarshal(output.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Data["original"] != tc.wantOriginal || got.Data["original_source"] != tc.wantSource || got.Data["note_id"] != "123" || got.Data["title"] != "Synthetic" {
				t.Fatalf("data = %#v", got.Data)
			}
		})
	}
}
