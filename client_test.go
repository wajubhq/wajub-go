package wajub

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testAPIKey = "sk_test.unit_test_key"

type recordedRequest struct {
	Method         string
	Path           string
	Authorization  string
	IdempotencyKey string
	Sync           string
	Query          string
}

type testTransport struct {
	server *httptest.Server
}

func (tr *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target, err := url.Parse(tr.server.URL)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	return http.DefaultTransport.RoundTrip(req)
}

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client, err := New(Config{
		APIKey:     testAPIKey,
		HTTPClient: &http.Client{Transport: &testTransport{server: server}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return client
}

func newTestServer(t *testing.T, responses []func(r *http.Request) (int, any)) (*httptest.Server, *recordedRequest) {
	t.Helper()
	call := 0
	var last recordedRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last.Method = r.Method
		last.Path = r.URL.Path
		last.Authorization = r.Header.Get("Authorization")
		last.IdempotencyKey = r.Header.Get("Idempotency-Key")
		last.Sync = r.Header.Get("X-Sync")
		last.Query = r.URL.RawQuery
		_, _ = io.ReadAll(r.Body)

		status := 200
		payload := any(map[string]any{})
		if call < len(responses) {
			status, payload = responses[call](r)
		}
		call++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(payload)
	}))

	t.Cleanup(server.Close)
	return server, &last
}

func signWebhook(secret, payload string, timestamp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.%s", timestamp, payload)))
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestNewRequiresAPIKey(t *testing.T) {
	t.Setenv("WAJUB_API_KEY", "")
	_, err := New(Config{})
	if err == nil || !strings.Contains(err.Error(), "api key is required") {
		t.Fatalf("expected api key error, got %v", err)
	}
}

func TestNewReadsAPIKeyFromEnvironment(t *testing.T) {
	t.Setenv("WAJUB_API_KEY", testAPIKey)
	client, err := New(Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}

func TestNewAcceptsPublicAndRestrictedKeys(t *testing.T) {
	if _, err := New(Config{APIKey: "pk_test.public"}); err != nil {
		t.Fatalf("public key rejected: %v", err)
	}
	if _, err := New(Config{APIKey: "rk_test.restricted"}); err != nil {
		t.Fatalf("restricted key rejected: %v", err)
	}
}

func TestHTTPAuthorizationHeader(t *testing.T) {
	server, rec := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) { return 200, map[string]any{"ok": true} },
	})

	client := newTestClient(t, server)
	if _, err := client.Global.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if rec.Authorization != testAPIKey {
		t.Fatalf("expected raw api key, got %q", rec.Authorization)
	}
}

func TestHTTPPostIncludesIdempotencyKey(t *testing.T) {
	server, rec := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			return 200, map[string]any{"transaction": map[string]any{"id": "pay_1"}}
		},
	})

	client := newTestClient(t, server)
	_, err := client.Payments.Create(context.Background(), map[string]any{"amount": 1000}, nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if !strings.HasPrefix(rec.IdempotencyKey, "wajub-") {
		t.Fatalf("expected idempotency prefix, got %q", rec.IdempotencyKey)
	}
}

func TestHTTPPostHonorsCustomOptions(t *testing.T) {
	server, rec := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			return 200, map[string]any{"transaction": map[string]any{"id": "pay_1"}}
		},
	})

	client := newTestClient(t, server)
	_, err := client.Payments.Create(context.Background(), map[string]any{"amount": 1000}, &RequestOptions{
		IdempotencyKey: "custom-key-123",
		Sync:           "acct_sync_ref",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if rec.IdempotencyKey != "custom-key-123" {
		t.Fatalf("expected custom idempotency key")
	}
	if rec.Sync != "acct_sync_ref" {
		t.Fatalf("expected X-Sync header")
	}
}

func TestHTTPGetDoesNotSendIdempotencyKey(t *testing.T) {
	server, rec := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) { return 200, map[string]any{"transactions": []any{}} },
	})

	client := newTestClient(t, server)
	_, err := client.Payments.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if rec.IdempotencyKey != "" {
		t.Fatalf("GET should not send idempotency key")
	}
}

func TestHTTPErrorResponse(t *testing.T) {
	server, _ := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			return 422, map[string]any{"message": "Unprocessable", "code": "validation_error"}
		},
	})

	client := newTestClient(t, server)
	_, err := client.Payments.Create(context.Background(), map[string]any{"amount": -1}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *InvalidRequestError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected InvalidRequestError, got %T", err)
	}
	if apiErr.HTTPStatus != 422 {
		t.Fatalf("expected 422, got %d", apiErr.HTTPStatus)
	}
}

func TestPaymentsCreateUnwrapsURLs(t *testing.T) {
	server, _ := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			return 200, map[string]any{
				"transaction":         map[string]any{"id": "pay_123", "status": "pending"},
				"authorization_token": "tok_abc",
				"authorization_url":   "https://checkout.test/pay/pay_123",
			}
		},
	})

	client := newTestClient(t, server)
	payment, err := client.Payments.Create(context.Background(), map[string]any{
		"amount": 5000, "currency": "XAF",
	}, nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if payment.ID != "pay_123" || payment.AuthorizationToken != "tok_abc" {
		t.Fatalf("unexpected payment payload: %+v", payment)
	}
}

func TestPaymentsListPagination(t *testing.T) {
	server, _ := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			return 200, map[string]any{
				"transactions": []map[string]any{{"id": "pay_1"}},
				"meta":         map[string]any{"current_page": float64(1), "last_page": float64(2)},
			}
		},
		func(r *http.Request) (int, any) {
			return 200, map[string]any{
				"transactions": []map[string]any{{"id": "pay_2"}},
				"meta":         map[string]any{"current_page": float64(2), "last_page": float64(2)},
			}
		},
	})

	client := newTestClient(t, server)
	page1, err := client.Payments.List(context.Background(), map[string]any{"per_page": 1})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if page1.Data[0]["id"] != "pay_1" || !page1.HasMore {
		t.Fatalf("unexpected page1")
	}

	page2, err := page1.GetNextPage(context.Background())
	if err != nil {
		t.Fatalf("next page failed: %v", err)
	}
	if page2.Data[0]["id"] != "pay_2" || page2.HasMore {
		t.Fatalf("unexpected page2")
	}
}

func TestRefundsCRUD(t *testing.T) {
	server, rec := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			return 200, map[string]any{"refund": map[string]any{"id": "ref_new"}}
		},
		func(r *http.Request) (int, any) {
			return 200, map[string]any{"refund": map[string]any{"id": "ref_new", "status": "pending"}}
		},
		func(r *http.Request) (int, any) { return 200, map[string]any{} },
	})

	client := newTestClient(t, server)
	created, err := client.Refunds.Create(context.Background(), map[string]any{
		"payment_id": "pay_1", "amount": 500,
	}, nil)
	if err != nil {
		t.Fatalf("create refund failed: %v", err)
	}
	refund := parseRefund(map[string]any{"refund": created})
	if refund.ID != "ref_new" {
		t.Fatalf("create refund failed: %+v", refund)
	}

	retrieved, err := client.Refunds.Retrieve(context.Background(), "ref_new")
	if err != nil {
		t.Fatalf("retrieve refund failed: %v", err)
	}
	parsed := parseRefund(map[string]any{"refund": retrieved})
	if parsed.Status != "pending" {
		t.Fatalf("retrieve refund failed: %+v", parsed)
	}

	if err := client.Refunds.Delete(context.Background(), "ref_new", nil); err != nil {
		t.Fatalf("delete refund failed: %v", err)
	}
	if rec.Method != http.MethodDelete {
		t.Fatalf("expected DELETE, got %s", rec.Method)
	}
}

func TestRetriesOn503ThenSucceeds(t *testing.T) {
	server, _ := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			return 503, map[string]any{"message": "Unavailable"}
		},
		func(r *http.Request) (int, any) {
			return 200, map[string]any{"ok": true}
		},
	})

	client := newTestClient(t, server)
	result, err := client.Global.Ping(context.Background())
	if err != nil {
		t.Fatalf("ping failed after retry: %v", err)
	}
	if result["ok"] != true {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestConstructEventRejectsInvalidSignatureFormat(t *testing.T) {
	w := &Webhooks{Secret: "whsec_test"}
	_, err := w.ConstructEvent([]byte("{}"), "bad", "1700000000", 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConstructEventRejectsOldTimestamp(t *testing.T) {
	secret := "whsec_test"
	payload := `{"event":"payment.succeeded"}`
	oldTS := time.Now().Add(-10 * time.Minute).Unix()
	sig := signWebhook(secret, payload, oldTS)

	w := &Webhooks{Secret: secret}
	_, err := w.ConstructEvent([]byte(payload), sig, strconv.FormatInt(oldTS, 10), 300*time.Second)
	if err == nil {
		t.Fatal("expected timestamp error")
	}
}

func TestPickListAndNormalizeAPIKey(t *testing.T) {
	data, meta := pickList(map[string]any{
		"refunds": []any{map[string]any{"id": "ref_1"}},
		"meta":    map[string]any{"total": float64(1)},
	}, "refunds")
	if len(data) != 1 || data[0]["id"] != "ref_1" {
		t.Fatalf("pickList failed")
	}
	if meta == nil {
		t.Fatal("expected meta")
	}
	if normalizeAPIKey("Bearer sk_test.abc") != "sk_test.abc" {
		t.Fatal("normalize failed")
	}
}

// A retried POST must reuse its Idempotency-Key, or the retry is a second
// request the API cannot recognise.
func TestRetriedPostReusesItsIdempotencyKey(t *testing.T) {
	var keys []string
	server, _ := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			keys = append(keys, r.Header.Get("Idempotency-Key"))
			return 503, map[string]any{"message": "Unavailable"}
		},
		func(r *http.Request) (int, any) {
			keys = append(keys, r.Header.Get("Idempotency-Key"))
			return 201, map[string]any{"transaction": map[string]any{"id": "trx_1"}}
		},
	})

	client := newTestClient(t, server)
	if _, err := client.Payments.Create(context.Background(), map[string]any{"amount": 1000, "currency": "XAF"}, nil); err != nil {
		t.Fatalf("create failed after retry: %v", err)
	}
	if len(keys) != 2 || keys[0] == "" || keys[0] != keys[1] {
		t.Fatalf("expected the same key on both attempts, got %v", keys)
	}
}

// A call sent again after a timeout can reach the API while the first one is
// still running: the API answers 409 with errors.idempotency_key, and the
// client asks again with the same key.
func TestRetriesA409ForAnIdempotencyKeyStillBeingProcessed(t *testing.T) {
	previous := idempotencyConflictDelay
	var waits []time.Duration
	idempotencyConflictDelay = func(attempt int) time.Duration {
		waits = append(waits, previous(attempt))
		return 0
	}
	t.Cleanup(func() { idempotencyConflictDelay = previous })

	var keys []string
	server, _ := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			keys = append(keys, r.Header.Get("Idempotency-Key"))
			return 409, map[string]any{
				"message": "A request with this Idempotency-Key is still being processed.",
				"errors":  map[string]any{"idempotency_key": []string{"A request with this Idempotency-Key is still being processed."}},
			}
		},
		func(r *http.Request) (int, any) {
			keys = append(keys, r.Header.Get("Idempotency-Key"))
			return 201, map[string]any{"transaction": map[string]any{"id": "trx_1"}}
		},
	})

	client := newTestClient(t, server)
	if _, err := client.Payments.Create(context.Background(), map[string]any{"amount": 1000, "currency": "XAF"}, &RequestOptions{IdempotencyKey: "ORDER-42"}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if len(keys) != 2 || keys[0] != "ORDER-42" || keys[1] != "ORDER-42" {
		t.Fatalf("expected ORDER-42 twice, got %v", keys)
	}
	if len(waits) != 1 || waits[0] != 2*time.Second {
		t.Fatalf("expected one 2s wait, got %v", waits)
	}
}

func TestRaisesAnyOther409AtOnce(t *testing.T) {
	calls := 0
	server, _ := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			calls++
			return 409, map[string]any{"error_code": "payment_in_progress", "message": "A charge is already in progress."}
		},
		func(r *http.Request) (int, any) {
			calls++
			return 200, map[string]any{"ok": true}
		},
	})

	client := newTestClient(t, server)
	if _, err := client.Payments.Process(context.Background(), "trx_1", map[string]any{"channel": "cm.mtn"}, nil); err == nil {
		t.Fatal("expected the 409 to be returned")
	}
	if calls != 1 {
		t.Fatalf("expected one call, got %d", calls)
	}
}
