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
	if !((apiErr.Code == 10202 && apiErr.Reason == "qps_exceeded") || (apiErr.Code == 10005 && apiErr.Reason == "rate_limited")) {
		return 0, false
	}
	readOnly := req.Method == http.MethodGet || (req.Method == http.MethodPost &&
		(req.URL.Path == "/open/api/v1/resource/recall" || req.URL.Path == "/open/api/v1/resource/recall/knowledge"))
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
