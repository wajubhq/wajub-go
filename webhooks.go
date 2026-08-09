package wajub

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const defaultWebhookTolerance = 300 * time.Second

// Webhooks verifies inbound webhook signatures.
type Webhooks struct {
	Secret string
}

// ConstructEvent verifies X-Wajub-Signature and parses the event payload.
//
// A tolerance of 0 or less uses the default 300s window; there is no way to disable
// replay protection.
func (w *Webhooks) ConstructEvent(payload []byte, signature, timestamp string, tolerance time.Duration) (map[string]any, error) {
	if w.Secret == "" {
		return nil, fmt.Errorf("wajub: webhook secret is required for ConstructEvent")
	}
	if !strings.HasPrefix(signature, "v1=") {
		return nil, &WebhookSignatureVerificationError{Message: "Invalid webhook signature format. Expected v1={hash}."}
	}

	body := string(payload)
	expected := hmacSha256(w.Secret, timestamp+"."+body)
	received := strings.TrimPrefix(signature, "v1=")

	if !timingSafeEqual(expected, received) {
		return nil, &WebhookSignatureVerificationError{Message: "Webhook signature verification failed."}
	}

	var ts float64
	if _, err := fmt.Sscan(timestamp, &ts); err != nil {
		return nil, &WebhookSignatureVerificationError{Message: "Invalid webhook timestamp."}
	}

	if tolerance <= 0 {
		tolerance = defaultWebhookTolerance
	}
	drift := time.Duration(abs(float64(time.Now().Unix())-ts)) * time.Second
	if drift > tolerance {
		return nil, &WebhookSignatureVerificationError{
			Message: fmt.Sprintf("Timestamp outside tolerance zone (%ds drift, allowed %ds).", int(drift.Seconds()), int(tolerance.Seconds())),
		}
	}

	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, &WebhookSignatureVerificationError{Message: "Invalid webhook payload JSON."}
	}
	return event, nil
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
