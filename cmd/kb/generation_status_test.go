package kb

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestKBNotesGenerationStatus(t *testing.T) {
	body := `{"success":true,"data":{"notes":[{"note_id":"31","title":"附件笔记","note_type":"link","generation_status":"attachment_processing"},{"note_id":"32","title":"旧服务","note_type":"link"}],"has_more":false}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	t.Setenv("GETNOTE_API_KEY", "test-key")
	run := func(args ...string) string {
		root := &cobra.Command{Use: "getnote"}
		root.PersistentFlags().StringP("output", "o", "table", "")
		root.AddCommand(NewKbCmd())
		root.SetArgs(append([]string{"kb", "topic1"}, args...))
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	table := run()
	if !strings.Contains(table, "[附件处理中] 附件笔记") || strings.Contains(table, "] 旧服务") {
		t.Fatalf("unexpected table: %s", table)
	}
	js := run("-o", "json")
	if strings.Count(js, `"generation_status"`) != 1 || !strings.Contains(js, `"generation_status": "attachment_processing"`) {
		t.Fatalf("unexpected json: %s", js)
	}
}
