package client

import (
	"context"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func mergeRetryAfter(apiErr *APIError, header string) {
	if apiErr.RetryAfterSeconds != nil && *apiErr.RetryAfterSeconds < 0 {
		apiErr.RetryAfterSeconds = nil
	}
	header = strings.TrimSpace(header)
	seconds, err := strconv.ParseInt(header, 10, 64)
	if err != nil {
		deadline, parseErr := http.ParseTime(header)
		if parseErr != nil {
			return
		}
		seconds = int64(math.Ceil(time.Until(deadline).Seconds()))
		if seconds < 0 {
			seconds = 0
		}
	} else if seconds < 0 {
		return
	}
	if apiErr.RetryAfterSeconds == nil || seconds > *apiErr.RetryAfterSeconds {
		apiErr.RetryAfterSeconds = &seconds
	}
}

func readRetryDelay(req *http.Request, apiErr *RequestError, retry int, remaining time.Duration) (time.Duration, bool) {
	if retry >= 2 || remaining <= 0 || apiErr.StatusCode != http.StatusTooManyRequests || !apiErr.Retryable {
		return 0, false
	}
	qpsLimit := apiErr.Code == 10202 && (apiErr.Reason == "qps_exceeded" || apiErr.Reason == "qps_global_exceeded" || apiErr.Reason == "qps_bucket_exceeded")
	if !qpsLimit && !(apiErr.Code == 10005 && apiErr.Reason == "rate_limited") {
		return 0, false
	}
	readOnly := isReadOnlyGET(req) || (req.Method == http.MethodPost &&
		(req.URL.Path == "/open/api/v1/resource/recall" || req.URL.Path == "/open/api/v1/resource/recall/knowledge" || req.URL.Path == "/open/api/v1/resource/note/batch-detail"))
	if !readOnly || (req.Body != nil && req.Body != http.NoBody && req.GetBody == nil) {
		return 0, false
	}
	delay := (200 * time.Millisecond) << retry
	if apiErr.RetryAfterSeconds != nil {
		// Compare before converting to Duration: large valid waits must not overflow.
		if *apiErr.RetryAfterSeconds > int64(remaining/time.Second) {
			return 0, false
		}
		delay = time.Duration(*apiErr.RetryAfterSeconds) * time.Second
	}
	delay += time.Duration(1+rand.Intn(50)) * time.Millisecond
	return delay, delay <= remaining
}

func isReadOnlyGET(req *http.Request) bool {
	if req.Method != http.MethodGet {
		return false
	}
	// Credential issuance also uses GET. Retry only operations whose read
	// semantics are known; future routes remain one-shot until classified.
	switch req.URL.Path {
	case "/open/api/v1/resource/note/list", "/open/api/v1/resource/note/detail",
		"/open/api/v1/resource/note/marks", "/open/api/v1/resource/note/sprouts", "/open/api/v1/resource/note/sprout",
		"/open/api/v1/resource/reference/list", "/open/api/v1/resource/knowledge/list", "/open/api/v1/resource/knowledge/subscribe/list",
		"/open/api/v1/resource/knowledge/notes", "/open/api/v1/resource/knowledge/directories", "/open/api/v1/resource/knowledge/file/capabilities",
		"/open/api/v1/resource/knowledge/bloggers", "/open/api/v1/resource/knowledge/blogger/contents", "/open/api/v1/resource/knowledge/blogger/content/detail",
		"/open/api/v1/resource/knowledge/lives", "/open/api/v1/resource/knowledge/live/detail", "/open/api/v1/resource/rate-limit/quota":
		return true
	}
	return false
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
