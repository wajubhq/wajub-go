package wajub

import (
	"strconv"
	"testing"
	"time"
)

func TestConstructEventVerifiesSignature(t *testing.T) {
	secret := "whsec_test_secret"
	payload := []byte(`{"event":"payment.succeeded","data":{}}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := "v1=" + hmacSha256(secret, timestamp+"."+string(payload))

	w := &Webhooks{Secret: secret}
	event, err := w.ConstructEvent(payload, signature, timestamp, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event["event"] != "payment.succeeded" {
		t.Fatalf("expected payment.succeeded, got %v", event["event"])
	}
}

func TestConstructEventRejectsWrongSignature(t *testing.T) {
	w := &Webhooks{Secret: "whsec_test"}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	_, err := w.ConstructEvent([]byte("{}"), "v1=deadbeef", timestamp, 0)
	if err == nil {
		t.Fatal("expected signature error")
	}
}

func TestPickResource(t *testing.T) {
	body := map[string]any{
		"transaction": map[string]any{"id": "pay_123"},
	}
	resource := pickResource(body, "transaction")
	if resource["id"] != "pay_123" {
		t.Fatalf("expected pay_123, got %v", resource["id"])
	}
}

func TestErrorFromResponse(t *testing.T) {
	err := errorFromResponse(422, map[string]any{
		"message": "Validation failed",
		"code":    "validation_error",
		"errors":  map[string]any{"amount": []any{"Must be positive"}},
	}, 0)

	invalid, ok := err.(*InvalidRequestError)
	if !ok {
		t.Fatalf("expected InvalidRequestError, got %T", err)
	}
	if invalid.Message != "Validation failed" || invalid.Code != "validation_error" || invalid.HTTPStatus != 422 {
		t.Fatalf("unexpected error: %+v", invalid)
	}
	if invalid.Errors["amount"] != "Must be positive" {
		t.Fatalf("unexpected field errors: %+v", invalid.Errors)
	}
}
