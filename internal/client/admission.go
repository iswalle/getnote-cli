package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

type admissionDecision struct {
	RetryAfterSeconds int64   `json:"retry_after_seconds"`
	Retryable         bool    `json:"retryable"`
	Dimension         string  `json:"dimension"`
	Bucket            string  `json:"bucket"`
	RequestExecution  string  `json:"request_execution"`
	RetryAction       string  `json:"retry_action"`
	RetryAfterMS      int64   `json:"retry_after_ms"`
	Rate              float64 `json:"sustained_rate_per_second"`
}

// Queues are shared by client instances in this process. Different processes,
// keys, and tools still share the authoritative server budget; the Agent must
// coordinate those callers instead of creating independent retry loops.
type admissionQueue struct {
	gate      chan struct{}
	refs      int
	lastUsed  time.Time
	intervals map[string]time.Duration
	next      map[string]time.Time
}

var admissionQueues = struct {
	sync.Mutex
	items map[string]*admissionQueue
}{items: make(map[string]*admissionQueue)}

func acquireAdmissionQueue(ctx context.Context, endpoint, key, clientID string) (*admissionQueue, func(), error) {
	hash := sha256.Sum256([]byte(endpoint + "\x00" + key + "\x00" + clientID))
	identity := hex.EncodeToString(hash[:])
	admissionQueues.Lock()
	now := time.Now()
	for id, item := range admissionQueues.items {
		if item.refs == 0 && now.Sub(item.lastUsed) > 5*time.Minute {
			delete(admissionQueues.items, id)
		}
	}
	q := admissionQueues.items[identity]
	if q == nil {
		q = &admissionQueue{gate: make(chan struct{}, 1), intervals: map[string]time.Duration{}, next: map[string]time.Time{}}
		q.gate <- struct{}{}
		admissionQueues.items[identity] = q
	}
	q.refs++
	admissionQueues.Unlock()
	deref := func() {
		admissionQueues.Lock()
		q.refs--
		q.lastUsed = time.Now()
		admissionQueues.Unlock()
	}
	select {
	case <-ctx.Done():
		deref()
		return nil, nil, ctx.Err()
	case <-q.gate:
		return q, func() { q.gate <- struct{}{}; deref() }, nil
	}
}

func requestBucket(req *http.Request) string {
	path := req.URL.Path
	if req.Method == http.MethodGet && (strings.HasSuffix(path, "/image/upload_token") || strings.HasSuffix(path, "/knowledge/file/upload_token")) {
		return "upload_credential"
	}
	if req.Method == http.MethodPost && strings.HasSuffix(path, "/note/save") {
		return "write_note"
	}
	if req.Method == http.MethodGet || strings.HasSuffix(path, "/note/task/progress") || strings.HasSuffix(path, "/note/batch-detail") || strings.HasSuffix(path, "/recall") || strings.HasSuffix(path, "/recall/knowledge") {
		return "read"
	}
	return "write"
}

func knownWrite(req *http.Request) bool {
	if req.Method != http.MethodPost {
		return false
	}
	switch req.URL.Path {
	case "/open/api/v1/resource/note/save", "/open/api/v1/resource/note/update", "/open/api/v1/resource/note/delete", "/open/api/v1/resource/note/sharing", "/open/api/v1/resource/note/tags/add", "/open/api/v1/resource/note/tags/delete", "/open/api/v1/resource/knowledge/file/upload", "/open/api/v1/resource/knowledge/create", "/open/api/v1/resource/knowledge/directory/create", "/open/api/v1/resource/knowledge/directory/update", "/open/api/v1/resource/knowledge/directory/delete", "/open/api/v1/resource/knowledge/blogger/follow":
		return true
	}
	return false
}

func (q *admissionQueue) wait(ctx context.Context, bucket string) error {
	now := time.Now()
	ready := q.next["global"]
	if q.next[bucket].After(ready) {
		ready = q.next[bucket]
	}
	if ready.After(now) {
		if err := waitForRetry(ctx, ready.Sub(now)); err != nil {
			return err
		}
	}
	now = time.Now()
	q.next["global"] = now.Add(q.intervals["global"])
	q.next[bucket] = now.Add(q.intervals[bucket])
	return nil
}

func validRate(rate float64) bool {
	return rate > 0 && rate <= 10000 && !math.IsNaN(rate) && !math.IsInf(rate, 0)
}

func (q *admissionQueue) observeDecision(raw json.RawMessage) {
	var decision admissionDecision
	if json.Unmarshal(raw, &decision) != nil || decision.Dimension != "qps" {
		return
	}
	if decision.Bucket != "global" && decision.Bucket != "read" && decision.Bucket != "write" && decision.Bucket != "write_note" && decision.Bucket != "upload_credential" {
		return
	}
	if validRate(decision.Rate) {
		q.intervals[decision.Bucket] = time.Duration(math.Ceil(float64(time.Second) / decision.Rate))
	}
	delay := q.intervals[decision.Bucket]
	if decision.RetryAfterMS > 0 && decision.RetryAfterMS <= int64((24*time.Hour)/time.Millisecond) && time.Duration(decision.RetryAfterMS)*time.Millisecond > delay {
		delay = time.Duration(decision.RetryAfterMS) * time.Millisecond
	}
	q.next[decision.Bucket] = time.Now().Add(delay)
}

func (q *admissionQueue) observePolicy(body []byte) {
	var envelope struct {
		Data struct {
			Policy *struct {
				Rates map[string]struct {
					Rate float64 `json:"sustained_rate_per_second"`
				} `json:"rates"`
			} `json:"rate_limit_policy"`
			Runtime struct {
				Policy *struct {
					Rates map[string]struct {
						Rate float64 `json:"sustained_rate_per_second"`
					} `json:"rates"`
				} `json:"rate_limit_policy"`
			} `json:"runtime"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return
	}
	policy := envelope.Data.Policy
	if policy == nil {
		policy = envelope.Data.Runtime.Policy
	}
	if policy == nil {
		return
	}
	for _, bucket := range []string{"global", "read", "write", "write_note", "upload_credential"} {
		rate, ok := policy.Rates[bucket]
		if ok && validRate(rate.Rate) {
			q.intervals[bucket] = time.Duration(math.Ceil(float64(time.Second) / rate.Rate))
		}
	}
}

func (q *admissionQueue) remainingWait(bucket string) time.Duration {
	ready := q.next["global"]
	if q.next[bucket].After(ready) {
		ready = q.next[bucket]
	}
	if delay := time.Until(ready); delay > 0 {
		return delay
	}
	return 0
}

func (q *admissionQueue) observeCooldown(bucket string, e APIError) {
	var d admissionDecision
	if json.Unmarshal(e.RateLimitDetail, &d) != nil || d.Dimension != "qps" {
		return
	}
	if d.Bucket == "global" || d.Bucket == "read" || d.Bucket == "write" || d.Bucket == "write_note" || d.Bucket == "upload_credential" {
		bucket = d.Bucket
	}
	delay := q.remainingWait(bucket)
	for _, seconds := range []*int64{e.RetryAfterSeconds, &d.RetryAfterSeconds} {
		if seconds != nil && *seconds > 0 && *seconds <= 86400 {
			if wait := time.Duration(*seconds) * time.Second; wait > delay {
				delay = wait
			}
		}
	}
	if until := time.Now().Add(delay); until.After(q.next[bucket]) {
		q.next[bucket] = until
	}
}
