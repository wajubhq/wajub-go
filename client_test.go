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

func TestInvalidRequestErrorKeepsEveryMessageOfAField(t *testing.T) {
	server, _ := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			return 422, map[string]any{
				"message": "The given data was invalid.",
				"code":    "validation_error",
				"errors": map[string]any{
					"amount":   []any{"The amount field must be a number.", "The amount field must be at least 100."},
					"currency": []any{"The currency field is required."},
					"email":    "The email field must be a valid email address.",
				},
			}
		},
	})

	client := newTestClient(t, server)
	_, err := client.Payments.Create(context.Background(), map[string]any{"amount": "abc"}, nil)
	var invalid *InvalidRequestError
	if !errors.As(err, &invalid) {
		t.Fatalf("expected InvalidRequestError, got %T", err)
	}

	want := map[string][]string{
		"amount":   {"The amount field must be a number.", "The amount field must be at least 100."},
		"currency": {"The currency field is required."},
		"email":    {"The email field must be a valid email address."},
	}
	if fmt.Sprint(invalid.FieldErrors) != fmt.Sprint(want) {
		t.Fatalf("FieldErrors = %v, want %v", invalid.FieldErrors, want)
	}

	// Errors keeps the first message of each field, as before.
	if invalid.Errors["amount"] != "The amount field must be a number." ||
		invalid.Errors["currency"] != "The currency field is required." ||
		invalid.Errors["email"] != "The email field must be a valid email address." ||
		len(invalid.Errors) != 3 {
		t.Fatalf("unexpected Errors: %v", invalid.Errors)
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

func TestPaymentAmountKeepsItsDecimals(t *testing.T) {
	cases := []struct {
		name     string
		amount   string
		currency string
		want     string
		whole    int64
	}{
		{"zero-decimal currency", `15000`, "XAF", "15000", 15000},
		{"two decimals", `12.5`, "GHS", "12.5", 12},
		{"two decimals as a string", `"12.50"`, "GHS", "12.5", 12},
		{"a tenth that has no exact binary form", `12.1`, "GHS", "12.1", 12},
		{"one cent", `0.01`, "USD", "0.01", 0},
		{"three decimals", `1.234`, "KWD", "1.234", 1},
		{"large amount with cents", `9876543210.99`, "NGN", "9876543210.99", 9876543210},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"transaction":{"id":"pay_1","amount":%s,"currency":%q}}`, tc.amount, tc.currency)
			}))
			t.Cleanup(server.Close)

			payment, err := newTestClient(t, server).Payments.Retrieve(context.Background(), "pay_1")
			if err != nil {
				t.Fatalf("retrieve failed: %v", err)
			}
			if got := strconv.FormatFloat(payment.AmountDecimal, 'f', -1, 64); got != tc.want {
				t.Fatalf("AmountDecimal = %s, want %s", got, tc.want)
			}
			if payment.Amount != tc.whole {
				t.Fatalf("Amount = %d, want %d", payment.Amount, tc.whole)
			}
		})
	}
}

func TestRefundAmountKeepsItsDecimals(t *testing.T) {
	refund := parseRefund(map[string]any{"refund": map[string]any{"id": "ref_1", "amount": 7.25}})
	if refund.AmountDecimal != 7.25 || refund.Amount != 7 {
		t.Fatalf("unexpected refund amounts: %+v", refund)
	}

	refund = parseRefund(map[string]any{"refund": map[string]any{"id": "ref_2", "amount": "0.50"}})
	if refund.AmountDecimal != 0.5 {
		t.Fatalf("unexpected refund amount: %v", refund.AmountDecimal)
	}

	refund = parseRefund(map[string]any{"refund": map[string]any{"id": "ref_3", "amount": "not a number"}})
	if refund.AmountDecimal != 0 || refund.Amount != 0 {
		t.Fatalf("an unreadable amount should read 0: %+v", refund)
	}
}

func TestAmountIsSentWithItsDecimals(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"transaction":{"id":"pay_1","amount":12.1,"currency":"GHS"}}`)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server)
	payment, err := client.Payments.Create(context.Background(), map[string]any{
		"amount": 12.1, "currency": "GHS", "email": "buyer@example.com",
	}, nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// The amount read back goes out again unchanged, and so does a three
	// decimal one.
	if _, err := client.Refunds.Create(context.Background(), map[string]any{
		"payment_id": payment.ID, "amount": payment.AmountDecimal,
	}, nil); err != nil {
		t.Fatalf("refund failed: %v", err)
	}
	if _, err := client.Payments.Create(context.Background(), map[string]any{
		"amount": 1.234, "currency": "KWD", "email": "buyer@example.com",
	}, nil); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	for i, want := range []string{`"amount":12.1,`, `"amount":12.1,`, `"amount":1.234,`} {
		if !strings.Contains(bodies[i], want) {
			t.Fatalf("request %d body %s does not carry %s", i, bodies[i], want)
		}
	}
}

func TestPaymentsRefundReturnsTheRefund(t *testing.T) {
	var method, path, idempotencyKey string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, idempotencyKey = r.Method, r.URL.Path, r.Header.Get("Idempotency-Key")
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"refund":{"id":"rfd_1","status":"pending","amount":12.5,"currency":"GHS"}}`)
	}))
	t.Cleanup(server.Close)

	params := map[string]any{"amount": 12.5, "reason": "requested_by_customer"}
	client := newTestClient(t, server)
	refund, err := client.Payments.Refund(context.Background(), "pay_1", params, &RequestOptions{IdempotencyKey: "refund-order-42"})
	if err != nil {
		t.Fatalf("refund failed: %v", err)
	}

	if method != http.MethodPost || path != "/refunds" {
		t.Fatalf("expected POST /refunds, got %s %s", method, path)
	}
	if body["payment"] != "pay_1" || body["amount"] != 12.5 || body["reason"] != "requested_by_customer" {
		t.Fatalf("unexpected body: %v", body)
	}
	if idempotencyKey != "refund-order-42" {
		t.Fatalf("expected the caller's idempotency key, got %q", idempotencyKey)
	}
	if _, touched := params["payment"]; touched || len(params) != 2 {
		t.Fatalf("the caller's params were changed: %v", params)
	}
	if refund.ID != "rfd_1" || refund.Status != "pending" || refund.AmountDecimal != 12.5 || refund.Currency != "GHS" {
		t.Fatalf("unexpected refund: %+v", refund)
	}
}

func TestPaymentsRefundSendsAnIdempotencyKeyOfItsOwn(t *testing.T) {
	server, rec := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			return 200, map[string]any{"refund": map[string]any{"id": "rfd_1"}}
		},
	})

	client := newTestClient(t, server)
	if _, err := client.Payments.Refund(context.Background(), "pay_1", nil, nil); err != nil {
		t.Fatalf("refund failed: %v", err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/refunds" || !strings.HasPrefix(rec.IdempotencyKey, "wajub-") {
		t.Fatalf("unexpected request: %+v", rec)
	}
}

func TestPaymentsRetrieveRefundReturnsTheRefund(t *testing.T) {
	server, rec := newTestServer(t, []func(r *http.Request) (int, any){
		func(r *http.Request) (int, any) {
			return 200, map[string]any{"refund": map[string]any{"id": "rfd_1", "status": "succeeded", "amount": 7.25, "currency": "GHS"}}
		},
	})

	client := newTestClient(t, server)
	refund, err := client.Payments.RetrieveRefund(context.Background(), "rfd_1")
	if err != nil {
		t.Fatalf("retrieve failed: %v", err)
	}
	if rec.Method != http.MethodGet || rec.Path != "/refunds/rfd_1" || rec.IdempotencyKey != "" {
		t.Fatalf("unexpected request: %+v", rec)
	}
	if refund.ID != "rfd_1" || refund.Status != "succeeded" || refund.AmountDecimal != 7.25 || refund.Currency != "GHS" {
		t.Fatalf("unexpected refund: %+v", refund)
	}
}

// idPathCall is a method that puts an id in the request path.
type idPathCall struct {
	name   string
	method string
	path   string // {id} is replaced with the escaped id
	do     func(id string) error
}

// idPathCalls lists every method that puts an id in the request path.
func idPathCalls(ctx context.Context, client *Client) []idPathCall {
	calls := []idPathCall{
		{"Payments.Retrieve", http.MethodGet, "/payments/{id}", func(id string) error { _, err := client.Payments.Retrieve(ctx, id); return err }},
		{"Payments.Cancel", http.MethodDelete, "/payments/{id}", func(id string) error { _, err := client.Payments.Cancel(ctx, id, nil); return err }},
		{"Payments.Process", http.MethodPost, "/payments/{id}", func(id string) error { _, err := client.Payments.Process(ctx, id, nil, nil); return err }},
		{"Payments.ProcessSplit", http.MethodPost, "/payments/{id}/splits", func(id string) error { _, err := client.Payments.ProcessSplit(ctx, id, nil, nil); return err }},
		{"Payments.ListRefunds", http.MethodGet, "/payments/{id}/refunds", func(id string) error { _, _, err := client.Payments.ListRefunds(ctx, id, nil); return err }},
		{"Payments.ListRefunds with a query", http.MethodGet, "/payments/{id}/refunds", func(id string) error {
			_, _, err := client.Payments.ListRefunds(ctx, id, map[string]any{"page": 2})
			return err
		}},
		{"Payments.RetrieveRefund", http.MethodGet, "/refunds/{id}", func(id string) error { _, err := client.Payments.RetrieveRefund(ctx, id); return err }},
		{"Customers.Block", http.MethodPost, "/customers/{id}/block", func(id string) error { _, err := client.Customers.Block(ctx, id, nil, nil); return err }},
		{"Customers.Unblock", http.MethodPost, "/customers/{id}/unblock", func(id string) error { _, err := client.Customers.Unblock(ctx, id, nil); return err }},
		{"Customers.Activate", http.MethodPost, "/customers/{id}/activate", func(id string) error { _, err := client.Customers.Activate(ctx, id, nil); return err }},
		{"Customers.Deactivate", http.MethodPost, "/customers/{id}/deactivate", func(id string) error { _, err := client.Customers.Deactivate(ctx, id, nil); return err }},
		{"Customers.ListTaxIDs", http.MethodGet, "/customers/{id}/tax_ids", func(id string) error { _, _, err := client.Customers.ListTaxIDs(ctx, id); return err }},
		{"Customers.CreateTaxID", http.MethodPost, "/customers/{id}/tax_ids", func(id string) error { _, err := client.Customers.CreateTaxID(ctx, id, nil, nil); return err }},
		{"Customers.DeleteTaxID", http.MethodDelete, "/customers/{id}/tax_ids/{id}", func(id string) error { return client.Customers.DeleteTaxID(ctx, id, id, nil) }},
		{"Invoices.Send", http.MethodPost, "/invoices/{id}/send", func(id string) error { _, err := client.Invoices.Send(ctx, id, nil); return err }},
		{"Invoices.MarkPaid", http.MethodPost, "/invoices/{id}/mark-paid", func(id string) error { _, err := client.Invoices.MarkPaid(ctx, id, nil, nil); return err }},
		{"Invoices.Cancel", http.MethodPost, "/invoices/{id}/cancel", func(id string) error { _, err := client.Invoices.Cancel(ctx, id, nil); return err }},
		{"Disputes.Retrieve", http.MethodGet, "/disputes/{id}", func(id string) error { _, err := client.Disputes.Retrieve(ctx, id); return err }},
		{"Disputes.SubmitEvidence", http.MethodPost, "/disputes/{id}/submit-evidence", func(id string) error { _, err := client.Disputes.SubmitEvidence(ctx, id, nil, nil); return err }},
		{"Disputes.Accept", http.MethodPost, "/disputes/{id}/accept", func(id string) error { _, err := client.Disputes.Accept(ctx, id, nil); return err }},
		{"Disputes.Close", http.MethodPost, "/disputes/{id}/close", func(id string) error { _, err := client.Disputes.Close(ctx, id, nil); return err }},
		{"Disputes.SendMessage", http.MethodPost, "/disputes/{id}/messages", func(id string) error { _, err := client.Disputes.SendMessage(ctx, id, nil, nil); return err }},
		{"Events.Retrieve", http.MethodGet, "/events/{id}", func(id string) error { _, err := client.Events.Retrieve(ctx, id); return err }},
		{"Events.Resend", http.MethodPost, "/events/{id}/resend", func(id string) error { _, err := client.Events.Resend(ctx, id, nil); return err }},
		{"WebhookEndpoints.RotateSecret", http.MethodPost, "/webhooks/{id}/rotate-secret", func(id string) error { _, err := client.WebhookEndpoints.RotateSecret(ctx, id, nil); return err }},
		{"Accounts.RegenerateToken", http.MethodPost, "/accounts/{id}/token", func(id string) error { _, err := client.Accounts.RegenerateToken(ctx, id, nil); return err }},
		{"Tax.RetrieveCode", http.MethodGet, "/tax/codes/{id}", func(id string) error { _, err := client.Tax.RetrieveCode(ctx, id); return err }},
		{"Tax.RetrieveRegistration", http.MethodGet, "/tax/registrations/{id}", func(id string) error { _, err := client.Tax.RetrieveRegistration(ctx, id); return err }},
		{"Tax.UpdateRegistration", http.MethodPut, "/tax/registrations/{id}", func(id string) error { _, err := client.Tax.UpdateRegistration(ctx, id, nil, nil); return err }},
		{"Tax.DeleteRegistration", http.MethodDelete, "/tax/registrations/{id}", func(id string) error { return client.Tax.DeleteRegistration(ctx, id, nil) }},
		{"Shield.RemoveFromBlocklist", http.MethodDelete, "/shield/blocklist/{id}", func(id string) error { return client.Shield.RemoveFromBlocklist(ctx, id, nil) }},
	}

	// Every service built on CrudService: Retrieve, Update and Delete.
	cruds := []struct {
		name    string
		service *CrudService
		path    string
	}{
		{"Customers", &client.Customers.CrudService, "/customers/"},
		{"Refunds", client.Refunds, "/refunds/"},
		{"Transfers", client.Transfers, "/transfers/"},
		{"Beneficiaries", client.Beneficiaries, "/beneficiaries/"},
		{"Links", client.Links, "/links/"},
		{"Accounts", &client.Accounts.CrudService, "/accounts/"},
		{"WebhookEndpoints", &client.WebhookEndpoints.CrudService, "/webhooks/"},
		{"Invoices", &client.Invoices.CrudService, "/invoices/"},
	}
	for _, crud := range cruds {
		service := crud.service
		calls = append(calls,
			idPathCall{crud.name + ".Retrieve", http.MethodGet, crud.path + "{id}", func(id string) error { _, err := service.Retrieve(ctx, id); return err }},
			idPathCall{crud.name + ".Update", http.MethodPut, crud.path + "{id}", func(id string) error { _, err := service.Update(ctx, id, nil, nil); return err }},
			idPathCall{crud.name + ".Delete", http.MethodDelete, crud.path + "{id}", func(id string) error { return service.Delete(ctx, id, nil) }},
		)
	}
	return calls
}

func TestIDsAreEscapedInRequestPaths(t *testing.T) {
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	t.Cleanup(server.Close)
	calls := idPathCalls(context.Background(), newTestClient(t, server))

	ids := []struct{ raw, escaped string }{
		{"pay_1?y=1#z", "pay_1%3Fy=1%23z"},
		{"pay 1%2F", "pay%201%252F"},
		{"..", "%2E%2E"},
		{".", "%2E"},
		// A blank id is not empty: it reaches the resource's own route,
		// where the API answers that there is no such resource.
		{" ", "%20"},
	}
	for _, c := range calls {
		for _, id := range ids {
			if err := c.do(id.raw); err != nil {
				t.Fatalf("%s(%q): %v", c.name, id.raw, err)
			}
			want := strings.ReplaceAll(c.path, "{id}", id.escaped)
			if method != c.method || path != want {
				t.Errorf("%s(%q) sent %s %s, want %s %s", c.name, id.raw, method, path, c.method, want)
			}
		}
	}
}

func TestUnusableIDsAreRefusedBeforeAnyRequest(t *testing.T) {
	var sent []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent = append(sent, r.Method+" "+r.URL.EscapedPath())
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	t.Cleanup(server.Close)
	client := newTestClient(t, server)
	ctx := context.Background()

	calls := idPathCalls(ctx, client)
	// DeleteTaxID takes two ids: either one is refused on its own.
	calls = append(calls,
		idPathCall{name: "Customers.DeleteTaxID's customer", do: func(id string) error { return client.Customers.DeleteTaxID(ctx, id, "txi_1", nil) }},
		idPathCall{name: "Customers.DeleteTaxID's tax id", do: func(id string) error { return client.Customers.DeleteTaxID(ctx, "cus_1", id, nil) }},
	)

	// The API reads an escaped "/" as a "/": "pay_1/refunds" would list the
	// refunds of pay_1, and "cus_1/tax_ids/txi_1" delete a tax id.
	ids := []struct {
		id   string
		want error
	}{
		{"", ErrEmptyID},
		{"pay_1/refunds", ErrInvalidID},
		{"cus_1/tax_ids/txi_1", ErrInvalidID},
		{"/", ErrInvalidID},
	}
	for _, c := range calls {
		for _, id := range ids {
			sent = nil
			err := c.do(id.id)
			if !errors.Is(err, id.want) {
				t.Errorf("%s(%q) returned %v, want %v", c.name, id.id, err, id.want)
			}
			if len(sent) > 0 {
				t.Errorf("%s(%q) sent %v", c.name, id.id, sent)
			}
		}
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection refused")
}

func TestCancellingTheContextEndsARetryWait(t *testing.T) {
	retryAfter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(retryAfter.Close)
	keyInProgress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"message":"Still being processed.","errors":{"idempotency_key":["Still being processed."]}}`)
	}))
	t.Cleanup(keyInProgress.Close)

	newClient := func(transport http.RoundTripper) *Client {
		client, err := New(Config{APIKey: testAPIKey, HTTPClient: &http.Client{Transport: transport}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return client
	}

	cases := []struct {
		name   string
		client *Client
	}{
		{"a 503 with Retry-After: 3", newClient(&testTransport{server: retryAfter})},
		{"a 409 for a key still being processed", newClient(&testTransport{server: keyInProgress})},
		{"a network error", newClient(failingTransport{})},
	}
	for _, c := range cases {
		ctx, cancel := context.WithCancel(context.Background())
		timer := time.AfterFunc(20*time.Millisecond, cancel)
		start := time.Now()
		_, err := c.client.Payments.Create(ctx, map[string]any{"amount": 1000}, nil)
		elapsed := time.Since(start)
		timer.Stop()
		cancel()

		if !errors.Is(err, context.Canceled) {
			t.Errorf("after %s, the call returned %v, want context.Canceled", c.name, err)
		}
		var connErr *APIConnectionError
		if !errors.As(err, &connErr) {
			t.Errorf("after %s, the call returned %T, want *APIConnectionError", c.name, err)
		}
		if elapsed > 200*time.Millisecond {
			t.Errorf("after %s, the call returned %v after the context was cancelled", c.name, elapsed-20*time.Millisecond)
		}
	}
}
