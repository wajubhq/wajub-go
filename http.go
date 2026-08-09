package wajub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RequestOptions configures a single API request.
type RequestOptions struct {
	IdempotencyKey string
	Headers        map[string]string
	Sync           string
}

type baseClient struct {
	apiKey               string
	baseURL              string
	idempotencyKeyPrefix string
	httpClient           *http.Client
	maxNetworkRetries    int
}

func newBaseClient(apiKey, baseURL, prefix string, httpClient *http.Client, maxNetworkRetries *int, timeout time.Duration) baseClient {
	if prefix == "" {
		prefix = "wajub"
	}
	if httpClient == nil {
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	retries := defaultMaxRetries
	if maxNetworkRetries != nil {
		retries = *maxNetworkRetries
	}
	return baseClient{
		apiKey:               normalizeAPIKey(apiKey),
		baseURL:              strings.TrimRight(baseURL, "/"),
		idempotencyKeyPrefix: prefix,
		httpClient:           httpClient,
		maxNetworkRetries:    retries,
	}
}

func (c *baseClient) get(ctx context.Context, path string, params map[string]any) (map[string]any, error) {
	return c.request(ctx, http.MethodGet, path, nil, params, nil)
}

func (c *baseClient) post(ctx context.Context, path string, body map[string]any, opts *RequestOptions) (map[string]any, error) {
	return c.request(ctx, http.MethodPost, path, body, nil, opts)
}

func (c *baseClient) put(ctx context.Context, path string, body map[string]any, opts *RequestOptions) (map[string]any, error) {
	return c.request(ctx, http.MethodPut, path, body, nil, opts)
}

func (c *baseClient) delete(ctx context.Context, path string, opts *RequestOptions) (map[string]any, error) {
	return c.request(ctx, http.MethodDelete, path, nil, nil, opts)
}

func (c *baseClient) request(
	ctx context.Context,
	method, path string,
	body map[string]any,
	query map[string]any,
	opts *RequestOptions,
) (map[string]any, error) {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}
		q := u.Query()
		for key, value := range query {
			if value != nil {
				q.Set(key, fmt.Sprint(value))
			}
		}
		u.RawQuery = q.Encode()
		endpoint = u.String()
	}

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	attempt := 0

	for {
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", c.apiKey)
		req.Header.Set("User-Agent", "wajub-go/"+Version)

		if opts != nil {
			for key, value := range opts.Headers {
				req.Header.Set(key, value)
			}
			if opts.Sync != "" {
				req.Header.Set("X-Sync", opts.Sync)
			}
		}

		idempotencyKey := ""
		if opts != nil {
			idempotencyKey = opts.IdempotencyKey
		}
		if idempotencyKey == "" && method != http.MethodGet && method != http.MethodDelete {
			idempotencyKey = createIdempotencyKey(c.idempotencyKeyPrefix)
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		res, err := c.httpClient.Do(req)
		if err != nil {
			if attempt < c.maxNetworkRetries {
				time.Sleep(retryDelay(attempt+1, nil))
				attempt++
				continue
			}
			return nil, &APIConnectionError{WajubError: WajubError{Message: err.Error(), Code: "network_error"}, Err: err}
		}

		data, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, err
		}

		parsed := decodeJSON(data)

		if res.StatusCode >= 200 && res.StatusCode < 300 {
			return parsed, nil
		}

		if shouldRetryStatus(res.StatusCode) && attempt < c.maxNetworkRetries {
			time.Sleep(retryDelay(attempt+1, res))
			attempt++
			continue
		}

		retryAfter := int(parseRetryAfter(res).Seconds())
		return nil, errorFromResponse(res.StatusCode, parsed, retryAfter)
	}
}
