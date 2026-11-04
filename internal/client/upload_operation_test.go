package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCredentialRecoveryOriginalOperation(t *testing.T) {
	issued, lookups := 0, 0
	var original string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open/api/v1/resource/image/config":
			fmt.Fprint(w, `{"success":true,"data":{"runtime":{"governance_enabled":true,"operation_status_endpoint":"/open/api/v1/resource/image/upload_operation","operation_replay":true,"idempotency_header":"Idempotency-Key"}}}`)
		case "/open/api/v1/resource/image/upload_token":
			issued++
			original = r.Header.Get("Idempotency-Key")
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
		case "/open/api/v1/resource/image/upload_operation":
			lookups++
			if r.URL.Query().Get("operation_id") != original {
				t.Error("changed operation")
			}
			fmt.Fprintf(w, `{"success":true,"data":{"operation_id":%q,"state":"issued","resume_until":%q,"credential":{"access_url":"https://example.com/original"}}}`, original, time.Now().Add(time.Hour).Format(time.RFC3339))
		default:
			t.Error(r.URL.Path)
		}
	}))
	defer s.Close()
	c := &Client{baseURL: s.URL, httpClient: s.Client()}
	got, err := c.ImageGetUploadToken("png")
	if err != nil || got.Data.AccessURL != "https://example.com/original" || issued != 1 || lookups != 1 || original == "" {
		t.Fatalf("issued=%d lookup=%d err=%v result=%+v", issued, lookups, err, got)
	}
}
func TestCredentialExplicitIDValidatedBeforeRequest(t *testing.T) {
	c := &Client{}
	if _, err := c.ImageToken("png", "bad id"); err == nil {
		t.Fatal("invalid operation accepted")
	}
}

func TestCredentialLegacyAndPendingNeverRemint(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			issued := 0
			id := ""
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/open/api/v1/resource/image/config":
					fmt.Fprintf(w, `{"success":true,"data":{"runtime":{"governance_enabled":%t,"operation_replay":%t,"idempotency_header":"Idempotency-Key","operation_status_endpoint":"/open/api/v1/resource/image/upload_operation"}}}`, enabled, enabled)
				case "/open/api/v1/resource/image/upload_token":
					issued++
					id = r.Header.Get("Idempotency-Key")
					if !enabled {
						fmt.Fprint(w, `{"success":true,"data":{"access_url":"https://example.com/legacy"}}`)
						return
					}
					conn, _, _ := w.(http.Hijacker).Hijack()
					conn.Close()
				case "/open/api/v1/resource/image/upload_operation":
					fmt.Fprintf(w, `{"success":true,"data":{"operation_id":%q,"state":"needs_reconciliation"}}`, id)
				}
			}))
			defer s.Close()
			c := &Client{baseURL: s.URL, httpClient: s.Client()}
			_, err := c.ImageGetUploadToken("png")
			if issued != 1 {
				t.Fatalf("reminted %d times", issued)
			}
			if !enabled && (err != nil || id != "") {
				t.Fatalf("legacy header=%q err=%v", id, err)
			}
			if enabled {
				var recovery *RecoveryError
				if !errors.As(err, &recovery) || recovery.OperationID != id || recovery.State != "needs_reconciliation" {
					t.Fatalf("lost original operation: %v", err)
				}
			}
		})
	}
}

func TestCredentialRedirectNeverIssuesSecondOperation(t *testing.T) {
	redirected := 0
	issued := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open/api/v1/resource/image/config":
			fmt.Fprint(w, `{"success":true,"data":{}}`)
		case "/open/api/v1/resource/image/upload_token":
			issued++
			http.Redirect(w, r, "/second", http.StatusFound)
		case "/second":
			redirected++
			fmt.Fprint(w, `{"success":true,"data":{}}`)
		}
	}))
	defer s.Close()
	c := &Client{baseURL: s.URL, httpClient: s.Client()}
	_, err := c.ImageGetUploadToken("png")
	if err == nil || issued != 1 || redirected != 0 {
		t.Fatalf("redirect issuance=%d second=%d err=%v", issued, redirected, err)
	}
}
func TestCredentialDeadlinesAndOperationIdentity(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)
	for _, tt := range []struct {
		name, raw string
		valid     bool
	}{
		{"expired", fmt.Sprintf(`{"operation_id":"same","resume_until":%q}`, past.Format(time.RFC3339)), false},
		{"mismatch", fmt.Sprintf(`{"operation_id":"other","resume_until":%q}`, future.Format(time.RFC3339)), false},
		{"outer_deadline", `{}`, true},
		{"original_earlier", fmt.Sprintf(`{"operation_id":"same","resume_until":%q,"future_hint":12}`, future.Format(time.RFC3339)), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			outer := time.Now().Add(time.Minute)
			raw, err := validateCredential(json.RawMessage(tt.raw), "same", outer)
			if (err == nil) != tt.valid {
				t.Fatalf("raw=%s err=%v", raw, err)
			}
			if tt.valid {
				var data map[string]any
				json.Unmarshal(raw, &data)
				until, _ := time.Parse(time.RFC3339, data["resume_until"].(string))
				if until.After(outer) || (tt.name == "original_earlier" && data["future_hint"] == nil) {
					t.Fatalf("deadline extended or metadata lost: %s", raw)
				}
			}
		})
	}
}
