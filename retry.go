package wajub

import (
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
