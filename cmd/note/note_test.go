package note

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
		})
	}
}
