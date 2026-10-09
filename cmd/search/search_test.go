package search

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsSearchTimeout(t *testing.T) {
	t.Parallel()
	if !isSearchTimeout(context.DeadlineExceeded) {
		t.Fatal("deadline exceeded must be recognized as timeout")
	}
	if isSearchTimeout(errors.New("search failed")) {
		t.Fatal("ordinary error must not be recognized as timeout")
	}
}

func TestPartialEmptySearchIsNotReportedAsNoMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"data":{"results":[],"partial":true,"warnings":[{"code":"file_identity_unavailable","resource_type":"file","count":1}]}}`)
	}))
	defer server.Close()
	t.Setenv("GETNOTE_API_URL", server.URL)
	var out bytes.Buffer
	cmd := NewSearchCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"query"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Partial search") || !strings.Contains(out.String(), "file_identity_unavailable") || strings.Contains(out.String(), "No results found.") {
		t.Fatalf("partial search was hidden: %s", out.String())
	}
}
