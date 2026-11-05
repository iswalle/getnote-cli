package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKBAddJSONRejections(t *testing.T) {
	for _, tc := range []struct {
		name   string
		count  int
		failed []string
	}{
		{"success", 2, []string{}}, {"partial", 1, []string{"2"}}, {"all rejected", 0, []string{"1", "2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/open/api/v1/resource/knowledge/note/batch-add" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"success_count": tc.count, "failed_note_ids": tc.failed, "topic_id": "test"}})
			}))
			defer server.Close()
			t.Setenv("GETNOTE_API_URL", server.URL)
			t.Setenv("GETNOTE_API_KEY", "test-key")
			root := discoveryRoot()
			root.SilenceErrors = true
			root.SilenceUsage = true
			root.SetArgs([]string{"kb", "add", "test", "1", "2", "-o", "json"})
			var out, errOut bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&errOut)
			err := root.Execute()
			rejected := len(tc.failed) > 0
			if (err != nil) != rejected {
				t.Fatalf("error=%v, rejected=%v", err, rejected)
			}
			payload := out.Bytes()
			if rejected {
				if out.Len() != 0 {
					t.Fatalf("failure printed success output: %s", out.String())
				}
				writeError(&errOut, err, "json")
				payload = errOut.Bytes()
			}
			var got struct {
				Success bool `json:"success"`
				Data    struct {
					Count  int      `json:"success_count"`
					Failed []string `json:"failed_note_ids"`
				} `json:"data"`
				Error struct {
					Reason string `json:"reason"`
				} `json:"error"`
			}
			if err := json.Unmarshal(payload, &got); err != nil {
				t.Fatal(err)
			}
			if got.Success == rejected || got.Data.Count != tc.count || len(got.Data.Failed) != len(tc.failed) {
				t.Fatalf("unexpected payload: %s", payload)
			}
			if rejected && got.Error.Reason != "knowledge_notes_rejected" {
				t.Fatalf("unexpected reason: %s", payload)
			}
		})
	}
}
