package notes

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExperienceBatchCommandUsesBackendAndReportsPending(t *testing.T) {
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/open/api/v1/resource/note/batch-detail" || r.Method != "POST" {
			t.Error("wrong batch route")
		}
		fmt.Fprint(w, `{"success":true,"meta":{"capabilities":["note.batch_detail.v1"]},"data":{"items":[{"note_id":"9007199254740993","status":"deferred_budget"}],"pending_ids":["9007199254740993"],"partial":true}}`)
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	root := &cobra.Command{Use: "getnote"}
	root.PersistentFlags().StringP("output", "o", "json", "")
	root.AddCommand(NewNotesCmd())
	root.SetArgs([]string{"notes", "batch", "9007199254740993", "--fields", "original"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !strings.Contains(out.String(), `"pending_ids"`) || !strings.Contains(out.String(), `"deferred_budget"`) {
		t.Fatalf("lost pending result calls=%d: %s", calls, out.String())
	}
}
