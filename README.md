# Wajub Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/wajubhq/wajub-go.svg)](https://pkg.go.dev/github.com/wajubhq/wajub-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Official **server-side** SDK for the [Wajub merchant API](https://docs.wajub.com). Accept mobile-money and card payments across Africa with a Stripe-inspired, resource-oriented client.

Use **[Wajub.js](https://docs.wajub.com/libraries/components/js)** for embedded checkout in the browser. Use this SDK on your backend with a secret (`sk_`) or restricted (`rk_`) API key — never expose secret keys in client-side code.

## Features

- Resource-oriented API (`client.Payments`, `client.Customers`, …)
- Automatic `Idempotency-Key` on mutating requests (override per call)
- Typed errors per HTTP status (`AuthenticationError`, `RateLimitError`, …)
- Automatic retries on 429 and 5xx (max 2, exponential backoff)
- Page-based pagination with `AutoPagingIter` and `GetNextPage`
- Webhook signature verification (HMAC-SHA256, timestamp tolerance)
- Zero third-party runtime dependencies (stdlib only)

## Requirements

| Requirement | Version |
|-------------|---------|
| Go | 1.22 or later |
| Dependencies | Standard library only |

## Installation

```bash
go get github.com/wajubhq/wajub-go
```

## Quick start

Amounts are passed in the **smallest currency unit** (e.g. cents for EUR/USD; whole francs for XAF).

### Redirect checkout

```go
package main

import (
    "context"
    "log"
    "os"

    wajub "github.com/wajubhq/wajub-go"
)

func main() {
    client, err := wajub.New(wajub.Config{
        APIKey: os.Getenv("WAJUB_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    payment, err := client.Payments.Create(context.Background(), map[string]any{
        "amount":   15000,
        "currency": "XAF",
        "email":    "buyer@example.com",
        "callback": "https://shop.example.com/order/complete",
    }, nil)
    if err != nil {
        log.Fatal(err)
    }

    log.Println(payment.AuthorizationURL)
}
```

### Inline / overlay (embed token)

```go
embed, err := client.Payments.Create(ctx, map[string]any{
    "amount":   15000,
    "currency": "XAF",
    "metadata": map[string]any{"mode": "embed"},
}, nil)
// Pass to Wajub.js: embed.AuthorizationToken
```

## HTTP server webhook

Use the **raw request body**:

```go
func webhookHandler(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    event, err := client.Webhooks.ConstructEvent(
        body,
        r.Header.Get("X-Wajub-Signature"),
        r.Header.Get("X-Wajub-Timestamp"),
        0, // default tolerance
    )
    if err != nil {
        http.Error(w, "invalid signature", http.StatusBadRequest)
        return
    }

    if eventType, _ := event["type"].(string); eventType == "payment.succeeded" {
        // fulfill order
    }

    w.WriteHeader(http.StatusOK)
}
```

## Configuration

| Variable | Description |
|----------|-------------|
| `WAJUB_API_KEY` | Secret or restricted API key (`sk_`, `sk_test.`, `rk_`, …) |
| `WAJUB_WEBHOOK_SECRET` | Webhook signing secret (`whsec_`) for `ConstructEvent()` |

Test mode is selected by your API key prefix (`sk_test.…`), not by the API URL. Production calls always go to `https://api.wajub.com`.

Pass options via `wajub.Config`:

```go
retries := 2
client, err := wajub.New(wajub.Config{
    APIKey:               os.Getenv("WAJUB_API_KEY"),
    WebhookSecret:        os.Getenv("WAJUB_WEBHOOK_SECRET"),
    IdempotencyKeyPrefix: "myshop",
    MaxNetworkRetries:    &retries,
    Timeout:              30 * time.Second,
})
```

## Resources (merchant API)

| Service | Methods |
|---------|---------|
| `client.Global` | `Ping`, `Channels`, `Countries`, `Currencies` |
| `client.Payments` | `Create`, `Retrieve`, `List`, `Cancel`, `Process`, `ProcessSplit`, `ListRefunds` |
| `client.Customers` | `Create`, `Retrieve`, `Update`, `Delete`, `List`, `Block`, `Unblock`, `Activate`, `Deactivate`, `ListTaxIds`, `CreateTaxId`, `DeleteTaxId` |
| `client.Refunds` | `Create`, `Retrieve`, `List` |
| `client.Transfers` | `Create`, `Retrieve`, `List` |
| `client.Beneficiaries` | `Create`, `Retrieve`, `Update`, `Delete`, `List` |
| `client.Links` | `Create`, `Retrieve`, `Update`, `Delete`, `List` |
| `client.Invoices` | `Create`, `Retrieve`, `Update`, `Delete`, `List`, `Send`, `MarkPaid`, `Cancel` |
| `client.Accounts` | `Create`, `Retrieve`, `Update`, `Delete`, `List`, `RegenerateToken` |
| `client.WebhookEndpoints` | `Create`, `Retrieve`, `Update`, `Delete`, `List`, `RotateSecret` |
| `client.Balance` | `Retrieve` |
| `client.Events` | `List`, `Retrieve`, `Resend` |
| `client.Disputes` | `List`, `Retrieve`, `SubmitEvidence`, `Accept`, `Close`, `SendMessage` |
| `client.Identity` | `Resolve`, `Validate` |
| `client.Tax` | `GetSettings`, `UpdateSettings`, `Rates`, `Calculate`, `Reports`, `ListCodes`, `RetrieveCode`, `ListRegistrations`, `CreateRegistration`, `RetrieveRegistration`, `UpdateRegistration`, `DeleteRegistration`, `Jurisdictions`, `Thresholds`, `ThresholdAlerts` |
| `client.Shield` | `GetSettings`, `UpdateSettings`, `Stats`, `ListBlocklist`, `AddToBlocklist`, `RemoveFromBlocklist` |
| `client.Listen` | `Config`, `Auth` |
| `client.Webhooks` | `ConstructEvent` (local — no HTTP) |

## Sync (Connect)

```go
client.Payments.Create(ctx, params, &wajub.RequestOptions{Sync: "acct_sync_ref"})
```

## Webhooks

```go
event, err := client.Webhooks.ConstructEvent(
    body,    // []byte — raw body, not parsed JSON
    r.Header.Get("X-Wajub-Signature"),
    r.Header.Get("X-Wajub-Timestamp"),
    0,
)
if err != nil {
    // signature or timestamp invalid
}

if event["type"] == "payment.succeeded" {
    // fulfill order
}
```

During local development, use the [Wajub CLI](https://github.com/wajubhq/wajub-cli) to forward webhooks to your machine.

## Pagination

```go
page, err := client.Payments.List(ctx, map[string]any{"per_page": 50})
if err != nil {
    log.Fatal(err)
}

all, err := page.AutoPagingIter(ctx)
if err != nil {
    log.Fatal(err)
}

// Manual page control
if page.HasMore {
    next, err := page.GetNextPage(ctx)
}
```

## Idempotency

POST and PUT requests automatically receive an `Idempotency-Key` header. Pass your own:

```go
client.Payments.Create(ctx, params, &wajub.RequestOptions{
    IdempotencyKey: fmt.Sprintf("order-%s", orderID),
})
```

## Error handling

Use type assertions on returned errors:

```go
payment, err := client.Payments.Create(ctx, params, nil)
if err != nil {
    var invalid *wajub.InvalidRequestError
    if errors.As(err, &invalid) {
        log.Println(invalid.Errors) // field-level validation errors
        return
    }
    var auth *wajub.AuthenticationError
    if errors.As(err, &auth) {
        // 401 — bad API key
        return
    }
    log.Fatal(err)
}
```

## Development

```bash
go test ./...
```

See [`examples/create_payment/`](examples/create_payment/) for a runnable sample.

## Documentation & support

- Full API reference: [docs.wajub.com/libraries/sdks/go](https://docs.wajub.com/libraries/sdks/go)
- Report issues: [github.com/wajubhq/wajub-go/issues](https://github.com/wajubhq/wajub-go/issues)

## License

MIT — see [LICENSE](LICENSE).
