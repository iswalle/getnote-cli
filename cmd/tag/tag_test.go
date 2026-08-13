package tag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
)

func TestTagListJSONUsesCommonEnvelope(t *testing.T) {
	t.Setenv("GETNOTE_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"data":{"note":{"id":"1917808813705036914","note_id":"1917808813705036914","title":"测试","tags":[{"id":"1001","name":"工作","type":"manual"}]}}}`)
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)

	root := &cobra.Command{Use: "getnote"}
	root.PersistentFlags().StringP("output", "o", "table", "")
	root.AddCommand(NewTagCmd())
	root.SetArgs([]string{"tag", "list", "1917808813705036914", "-o", "json"})
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Success bool `json:"success"`
		Data    struct {
			NoteID string        `json:"note_id"`
			Tags   []interface{} `json:"tags"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("decode output: %v\n%s", err, output.String())
	}
	if !got.Success || got.Data.NoteID != "1917808813705036914" || len(got.Data.Tags) != 1 {
		t.Fatalf("unexpected output: %#v", got)
	}
}
