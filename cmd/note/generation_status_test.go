package note

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func runNote(t *testing.T, note string, args ...string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"success":true,"data":{"note":%s}}`, note)
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	t.Setenv("GETNOTE_API_KEY", "test-key")
	root := &cobra.Command{Use: "getnote"}
	root.PersistentFlags().StringP("output", "o", "table", "")
	root.AddCommand(NewNoteCmd())
	root.SetArgs(append([]string{"note", "21"}, args...))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestNoteDetailGenerationStatus(t *testing.T) {
	generating := `{"note_id":"21","title":"链接","note_type":"link","content":"","generation_status":"generating"}`
	table := runNote(t, generating)
	if !strings.Contains(table, "[生成中] 链接") || !strings.Contains(table, "generating（生成中") {
		t.Fatalf("missing generation hint: %s", table)
	}
	if js := runNote(t, generating, "-o", "json"); !strings.Contains(js, `"generation_status": "generating"`) {
		t.Fatalf("json lost generation_status: %s", js)
	}

	legacy := `{"note_id":"21","title":"链接","note_type":"link","content":"正文"}`
	table = runNote(t, legacy)
	if strings.Contains(table, "Generation") || strings.Contains(table, "[") {
		t.Fatalf("legacy note must not show generation hint: %s", table)
	}
	if js := runNote(t, legacy, "-o", "json"); strings.Contains(js, "generation_status") {
		t.Fatalf("legacy json must not invent generation_status: %s", js)
	}
}
