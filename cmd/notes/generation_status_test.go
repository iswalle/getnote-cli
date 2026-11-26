package notes

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func runNotes(t *testing.T, body string, args ...string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	t.Setenv("GETNOTE_API_KEY", "test-key")
	root := &cobra.Command{Use: "getnote"}
	root.PersistentFlags().StringP("output", "o", "table", "")
	root.AddCommand(NewNotesCmd())
	root.SetArgs(append([]string{"notes"}, args...))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestNotesShowsGenerationStatus(t *testing.T) {
	body := `{"success":true,"data":{"notes":[{"note_id":"11","title":"写作中","note_type":"link","generation_status":"generating"},{"note_id":"12","title":"已完成","note_type":"link","generation_status":"completed"},{"note_id":"13","title":"旧服务","note_type":"link"}],"has_more":false}}`
	table := runNotes(t, body)
	if !strings.Contains(table, "[生成中] 写作中") {
		t.Fatalf("missing marker: %s", table)
	}
	if strings.Count(table, "[") != 1 {
		t.Fatalf("completed or legacy notes must not be marked: %s", table)
	}
	js := runNotes(t, body, "-o", "json")
	if strings.Count(js, `"generation_status"`) != 2 || !strings.Contains(js, `"generation_status": "generating"`) {
		t.Fatalf("json lost or invented generation_status: %s", js)
	}
}
