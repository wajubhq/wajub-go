package wajub

import (
	"context"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultMaxRetries = 2

var retryableStatus = map[int]bool{
	429: true,
	500: true,
	502: true,
	503: true,
	504: true,
}

func shouldRetryStatus(status int) bool {
	return retryableStatus[status]
}

func retryDelay(attempt int, res *http.Response) time.Duration {
	if res != nil {
		if delay := parseRetryAfter(res); delay > 0 {
			return delay
		}
	}

	base := time.Duration(500*(1<<(attempt-1))) * time.Millisecond
	jitter := 0.5 + rand.Float64()*0.5

	return time.Duration(float64(base) * jitter)
}

// waitToRetry pauses for d before the next attempt. It returns ctx's error as
// soon as ctx is cancelled or its deadline passes, rather than at the end of
// the pause.
func waitToRetry(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// connectionError is the error of a call that ends without a response it can
// return: the transport failed, or ctx ended while waiting to retry.
func connectionError(err error) *APIConnectionError {
	return &APIConnectionError{WajubError: WajubError{Message: err.Error(), Code: "network_error"}, Err: err}
}

func parseRetryAfter(res *http.Response) time.Duration {
	header := strings.TrimSpace(res.Header.Get("Retry-After"))
	if header == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(header); err == nil {
		return time.Duration(seconds) * time.Second
	}

	if t, err := http.ParseTime(header); err == nil {
		delay := time.Until(t)
		if delay < 0 {
			return 0
		}
		return delay
	}

	return 0
}

// isIdempotencyKeyInProgress reports whether the API answered the 409 it sends
// while a request with the same Idempotency-Key is still running
// (errors.idempotency_key). A call sent again after a timeout can arrive
// before the first one has finished: its answer is on the way, so the call is
// worth repeating with the same key. Every other 409 is a real refusal.
func isIdempotencyKeyInProgress(status int, body map[string]any) bool {
	if status != http.StatusConflict {
		return false
	}
	errs, ok := body["errors"].(map[string]any)
	if !ok {
		return false
	}
	_, present := errs["idempotency_key"]
	return present
}

// idempotencyConflictDelay is the pause before asking again about a key still
// being processed: 2, 4, then 8 seconds. A variable so tests need not wait.
var idempotencyConflictDelay = func(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	return time.Duration(1<<attempt) * time.Second
}
