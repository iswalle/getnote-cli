package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAdmissionErrorPreservesDecisionAndFutureFields(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		fmt.Fprint(w, `{"success":false,"error":{"code":10203,"reason":"upload_quota_rolling_exceeded","retryable":false,"rate_limit_detail":{"dimension":"upload_rolling","request_execution":"not_started","retry_action":"complete_existing","limit":100,"future_hint":{"enabled":true}}},"request_id":"decision-1"}`)
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	_, err := New("").NoteGet("1")
	var requestErr *RequestError
	if !errors.As(err, &requestErr) {
		t.Fatalf("expected structured refusal: %v", err)
	}
	raw, _ := json.Marshal(requestErr.APIError)
	var envelope map[string]json.RawMessage
	json.Unmarshal(raw, &envelope)
	var detail map[string]any
	json.Unmarshal(envelope["rate_limit_detail"], &detail)
	if detail["retry_action"] != "complete_existing" || detail["future_hint"] == nil || requestErr.RequestID != "decision-1" {
		t.Fatalf("decision was lost: %s", raw)
	}
}

func TestAdmissionRetriesOnlyExplicitUnstartedWriteWithIdenticalPayload(t *testing.T) {
	var bodies []string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		if len(bodies) == 1 {
			w.WriteHeader(429)
			fmt.Fprint(w, `{"success":false,"error":{"code":10202,"reason":"qps_bucket_exceeded","retryable":true,"retry_after_seconds":0,"rate_limit_detail":{"dimension":"qps","retryable":true,"request_execution":"not_started","retry_action":"retry_same_operation","bucket":"write_note"}}}`)
			return
		}
		fmt.Fprint(w, `{"success":true,"data":{"note_id":"123"}}`)
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	_, err := New("").NoteSave(NoteSaveRequest{NoteType: "img_text", ImageURLs: []string{"https://storage.example/original.jpg"}, ClientRequestID: "same-note-operation"})
	if err != nil || len(bodies) != 2 || bodies[0] != bodies[1] {
		t.Fatalf("unstarted write not safely replayed: calls=%d err=%v", len(bodies), err)
	}
}

func TestAdmissionNeverRetriesReadWhenDecisionSaysExecutionStarted(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(429)
		fmt.Fprint(w, `{"success":false,"error":{"code":10202,"reason":"qps_bucket_exceeded","retryable":true,"retry_after_seconds":0,"rate_limit_detail":{"dimension":"qps","request_execution":"admitted","retry_action":"retry_same_operation"}},"request_id":"already-started"}`)
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	_, err := New("").NoteGet("1")
	if err == nil || calls.Load() != 1 {
		t.Fatalf("started decision was retried: calls=%d err=%v", calls.Load(), err)
	}
}

func TestAdmissionSerializesSameIdentityWithoutBlockingAnotherAccount(t *testing.T) {
	first := make(chan struct{})
	release := make(chan struct{})
	second := make(chan struct{}, 1)
	independent := make(chan struct{}, 1)
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	defer finish()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("id") {
		case "first":
			close(first)
			<-release
		case "second":
			second <- struct{}{}
		case "independent":
			independent <- struct{}{}
		}
		fmt.Fprint(w, `{"success":true,"data":{"note":{"note_id":"1"}}}`)
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	a, a2, b := New(""), New(""), New("")
	a.apiKey, a2.apiKey, b.apiKey = "account-a", "account-a", "account-b"
	go a.NoteGet("first")
	<-first
	go a2.NoteGet("second")
	go b.NoteGet("independent")
	select {
	case <-independent:
	case <-time.After(time.Second):
		finish()
		t.Fatal("another identity was blocked")
	}
	select {
	case <-second:
		finish()
		t.Fatal("same identity requests ran concurrently")
	case <-time.After(40 * time.Millisecond):
	}
	finish()
	select {
	case <-second:
	case <-time.After(time.Second):
		t.Fatal("queued request never resumed")
	}
}

func TestAdmissionLowRateDoesNotExceedRetryBudget(t *testing.T) {
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(429)
		fmt.Fprint(w, `{"success":false,"error":{"code":10202,"reason":"qps_bucket_exceeded","retryable":true,"retry_after_seconds":0,"rate_limit_detail":{"dimension":"qps","bucket":"read","retryable":true,"request_execution":"not_started","retry_action":"retry_same_operation","sustained_rate_per_second":0.1}}}`)
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	start := time.Now()
	_, err := New("").NoteGet("1")
	if err == nil || calls != 1 || time.Since(start) > time.Second {
		t.Fatalf("exceeded budget calls=%d elapsed=%v err=%v", calls, time.Since(start), err)
	}
}
func TestAdmissionRequiresDetailRetryableAndNonnegativeWait(t *testing.T) {
	for _, detail := range []string{
		`{"dimension":"qps","request_execution":"not_started","retry_action":"retry_same_operation","retryable":false}`,
		`{"dimension":"qps","request_execution":"not_started","retry_action":"retry_same_operation"}`,
		`{"dimension":"qps","request_execution":"not_started","retry_action":"retry_same_operation","retryable":true,"retry_after_ms":-1}`,
	} {
		calls := 0
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(429)
			fmt.Fprintf(w, `{"success":false,"error":{"code":10202,"reason":"qps_bucket_exceeded","retryable":true,"rate_limit_detail":%s}}`, detail)
		}))
		t.Setenv("GETNOTE_API_URL", s.URL)
		_, err := New("").NoteSave(NoteSaveRequest{NoteType: "plain_text", Content: "exact"})
		s.Close()
		if err == nil || calls != 1 {
			t.Fatalf("unsafe retry calls=%d err=%v", calls, err)
		}
	}
}
func TestAdmissionHonorsDetailMilliseconds(t *testing.T) {
	calls := 0
	var first time.Time
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			first = time.Now()
			w.WriteHeader(429)
			fmt.Fprint(w, `{"success":false,"error":{"code":10202,"reason":"qps_bucket_exceeded","retryable":true,"retry_after_seconds":0,"rate_limit_detail":{"dimension":"qps","bucket":"read","retryable":true,"request_execution":"not_started","retry_action":"retry_same_operation","retry_after_ms":120}}}`)
			return
		}
		if time.Since(first) < 120*time.Millisecond {
			t.Error("retry too early")
		}
		fmt.Fprint(w, `{"success":true,"data":{}}`)
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	_, err := New("").NoteGet("1")
	if err != nil || calls != 2 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestAdmissionGlobalCooldownAppliesAcrossBuckets(t *testing.T) {
	var refused time.Time
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/open/api/v1/resource/note/save" {
			refused = time.Now()
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(429)
			fmt.Fprint(w, `{"success":false,"error":{"code":10202,"reason":"qps_global_exceeded","retryable":false,"rate_limit_detail":{"dimension":"qps","bucket":"global","retryable":false,"request_execution":"not_started","retry_action":"pause","retry_after_ms":1}}}`)
			return
		}
		if time.Since(refused) < time.Second {
			t.Error("another bucket bypassed global cooldown")
		}
		fmt.Fprint(w, `{"success":true,"data":{}}`)
	}))
	defer s.Close()
	t.Setenv("GETNOTE_API_URL", s.URL)
	c := New("")
	if _, err := c.NoteSave(NoteSaveRequest{NoteType: "plain_text", Content: "exact"}); err == nil {
		t.Fatal("refusal lost")
	}
	if _, err := c.NoteGet("1"); err != nil {
		t.Fatal(err)
	}
}
