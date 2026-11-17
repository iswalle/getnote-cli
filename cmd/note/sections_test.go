package note

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProgressiveCommandContractAndNoFullReadFallback(t *testing.T) {
	var paths []string
	mode := "ok"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Query().Get("id") != "9007199254740993" {
			t.Error("ID precision lost")
		}
		if r.URL.Query().Get("keyword") != "" && r.URL.Query().Has("limit") {
			t.Error("omitted keyword window changed")
		}
		if mode == "version" {
			fmt.Fprint(w, `{"success":false,"error":{"code":10002,"reason":"note_version_changed","message":"refresh outline","retryable":false}}`)
			return
		}
		caps := []string{"note.progressive_read.v1"}
		if mode == "old" {
			caps = nil
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true, "meta": map[string]any{"capabilities": caps}, "data": map[string]any{"id": "9007199254740993", "text": "节选😀\r\n", "total": 12345, "version": "hash-12345", "offset_unit": "utf16"}})
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	t.Setenv("GETNOTE_API_KEY", "synthetic")
	run := func(args ...string) (string, error) {
		root := &cobra.Command{Use: "getnote", SilenceErrors: true, SilenceUsage: true}
		root.PersistentFlags().StringP("output", "o", "table", "")
		root.AddCommand(NewNoteCmd())
		root.SetArgs(append([]string{"note"}, args...))
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		err := root.Execute()
		return out.String(), err
	}
	for _, args := range [][]string{{"outline", "9007199254740993", "-o", "json"}, {"section", "9007199254740993", "--offset", "8000", "--version", "hash-12345", "-o", "json"}, {"section", "9007199254740993", "--keyword", "绩效|晋升", "-o", "json"}} {
		out, err := run(args...)
		if err != nil {
			t.Fatal(err)
		}
		var envelope map[string]any
		if json.Unmarshal([]byte(out), &envelope) != nil || envelope["data"].(map[string]any)["text"] != "节选😀\r\n" {
			t.Fatal("response changed")
		}
	}
	for _, m := range []string{"old", "version"} {
		mode = m
		before := len(paths)
		if _, err := run("section", "9007199254740993"); err == nil {
			t.Fatal("unsupported/stale response accepted")
		}
		if len(paths) != before+1 {
			t.Fatal("automatic full-content fallback or retry")
		}
	}
	before := len(paths)
	if _, err := run("section", "9007199254740993", "--limit", "0"); err == nil {
		t.Fatal("bad limit accepted")
	}
	if len(paths) != before {
		t.Fatal("invalid request sent")
	}
	for _, path := range paths {
		if strings.HasSuffix(path, "/detail") {
			t.Fatal("unexpected full read")
		}
	}
}
