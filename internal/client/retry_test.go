package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestExperienceRateLimitDoesNotRepeatWritesQuotaOrLongWaits(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, reason, retryAfter string
		code                                   int
	}{
		{"write", "POST", "/open/api/v1/resource/note/save", "qps_exceeded", "0", 10202},
		{"credential issuance", "GET", "/open/api/v1/resource/image/upload_token", "qps_exceeded", "0", 10202},
		{"unknown route", "GET", "/open/api/v1/resource/future-action", "qps_exceeded", "0", 10202},
		{"quota", "GET", "/open/api/v1/resource/note/detail", "quota_exhausted", "0", 10203},
		{"long wait", "GET", "/open/api/v1/resource/note/detail", "qps_exceeded", "60", 10202},
		{"unknown reason", "GET", "/open/api/v1/resource/note/detail", "future_limit", "0", 10202},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) > 1 {
					_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
					return
				}
				w.Header().Set("Retry-After", tc.retryAfter)
				w.WriteHeader(429)
				_, _ = fmt.Fprintf(w, `{"success":false,"error":{"code":%d,"reason":%q,"retryable":true},"request_id":"req-limit"}`, tc.code, tc.reason)
			}))
			defer s.Close()
			t.Setenv("GETNOTE_API_URL", s.URL)
			req, _ := http.NewRequest(tc.method, s.URL+tc.path, bytes.NewReader([]byte(`{"content":"once"}`)))
			_, err := doRequest[map[string]any](New(""), req)
			var apiErr *RequestError
			if !errors.As(err, &apiErr) || calls.Load() != 1 || apiErr.RequestID != "req-limit" {
				t.Fatalf("calls=%d err=%v; expected one request and original error", calls.Load(), err)
			}
		})
	}
}

func TestExperienceBatchReadRetryReplaysIdenticalBody(t *testing.T) {
	var bodies []string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body bytes.Buffer
		body.ReadFrom(r.Body)
		bodies = append(bodies, body.String())
		if len(bodies) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429)
			fmt.Fprint(w, `{"success":false,"error":{"code":10202,"reason":"qps_exceeded","retryable":true}}`)
			return
		}
		fmt.Fprint(w, `{"success":true,"meta":{"capabilities":["note.batch_detail.v1"]},"data":{"items":[{"note_id":"42","status":"unavailable"}]}}`)
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	_, err := New("").NotesBatch(BatchDetailOptions{IDs: []string{"42"}})
	if err != nil || len(bodies) != 2 || bodies[0] != bodies[1] {
		t.Fatalf("batch read retry changed body: count=%d err=%v", len(bodies), err)
	}
}

func TestExperienceRetryStopsAfterTwoAdditionalReads(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":10202,"reason":"qps_exceeded","retryable":true},"request_id":"req-last"}`))
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	_, err := New("").NoteGet("123")
	var apiErr *RequestError
	if !errors.As(err, &apiErr) || calls.Load() != 3 || apiErr.RequestID != "req-last" {
		t.Fatalf("calls=%d err=%v, expected three attempts preserving the last error", calls.Load(), err)
	}
}

func TestExperienceRetryWaitHonorsCancellationAndPreservesError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":10202,"reason":"qps_exceeded","retryable":true},"request_id":"req-cancel"}`))
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", s.URL+"/open/api/v1/resource/note/detail?id=123", nil)
	start := time.Now()
	_, err := doRequest[map[string]any](New(""), req)
	var apiErr *RequestError
	if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &apiErr) || apiErr.RequestID != "req-cancel" || time.Since(start) > 400*time.Millisecond {
		t.Fatalf("elapsed=%s err=%v, expected prompt cancellation with last API error", time.Since(start), err)
	}
}

func TestExperienceErrorAdviceSurvivesBodyAndHeader(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "9")
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":10000,"reason":"invalid_request","field":"source","retryable":false,"supported_values":["dedao","manual"],"retry_after_seconds":12,"future_field":true},"request_id":"req-advice"}`))
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	_, err := New("").NoteGet("123")
	var apiErr *RequestError
	if !errors.As(err, &apiErr) {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(apiErr)
	var data map[string]any
	_ = json.Unmarshal(raw, &data)
	if data["retry_after_seconds"] != float64(12) || fmt.Sprint(data["supported_values"]) != "[dedao manual]" || apiErr.Field != "source" {
		t.Fatalf("advice lost: %s", raw)
	}
}

func TestExperienceReadOnlySearchReplaysOriginalBody(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["query"] != "课程" {
			t.Error("search body changed", body)
		}
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429)
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":10202,"reason":"qps_exceeded","retryable":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"results":[]}}`))
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	_, err := New("").NoteSearch("课程", 3)
	if err != nil || calls.Load() != 2 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
}

func TestExperienceRetryBudgetNeverShortensServerWait(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.invalid/open/api/v1/resource/note/detail", nil)
	for _, tc := range []struct {
		name      string
		seconds   int64
		remaining time.Duration
		want      bool
	}{
		{"fits", 4, 5 * time.Second, true},
		{"no room for positive jitter", 5, 5 * time.Second, false},
		{"already spent budget", 1, 900 * time.Millisecond, false},
		{"overflow cannot wrap to short delay", 9223372036854775807, 5 * time.Second, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := &RequestError{APIError: APIError{Code: 10202, Reason: "qps_exceeded", Retryable: true, RetryAfterSeconds: &tc.seconds}, StatusCode: 429}
			delay, ok := readRetryDelay(req, err, 0, tc.remaining)
			if ok != tc.want || (ok && (delay < time.Duration(tc.seconds)*time.Second || delay > tc.remaining)) {
				t.Fatalf("delay=%s ok=%v", delay, ok)
			}
		})
	}
}
