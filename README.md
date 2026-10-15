# Wajub Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/wajubhq/wajub-go.svg)](https://pkg.go.dev/github.com/wajubhq/wajub-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Official server-side SDK for the [Wajub API](https://docs.wajub.com): create payments, confirm them, pay out and verify webhooks from your Go backend.

It runs on your server with a secret (`sk.…`, `sk_test.…`) or restricted (`rk.…`, `rk_test.…`) API key. Never expose these keys in client-side code: in the browser, use [Wajub.js](https://docs.wajub.com/libraries/components/js) with your publishable key (`pk.…`, `pk_test.…`).

## Requirements

| Requirement | Version |
|-------------|---------|
| Go | 1.22 or later |

## Installation

```bash
go get github.com/wajubhq/wajub-go
```

## Quick start

Amounts are in the currency's **main unit**: `12.50` for GHS 12.50, `5000` for XAF 5,000.

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/wajubhq/wajub-go"
)

func main() {
    client, err := wajub.New(wajub.Config{
        APIKey:        os.Getenv("WAJUB_API_KEY"),
        WebhookSecret: os.Getenv("WAJUB_WEBHOOK_SECRET"),
    })
    if err != nil {
        log.Fatal(err)
    }

    payment, err := client.Payments.Create(context.Background(), map[string]any{
        "amount":    15000,
        "currency":  "XAF",
        "email":     "buyer@example.com",
        "reference": "order-1042",
        "callback":  "https://shop.example.com/order/complete",
    }, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Redirect the customer to payment.AuthorizationURL,
    // or mount payment.AuthorizationToken with Wajub.js in your page.
    log.Println(payment.AuthorizationURL)
}
```

Then confirm the result on your server: handle the `payment.succeeded` webhook, or call `client.Payments.Retrieve(ctx, payment.ID)`.

A `Payment` holds `ID`, `Status`, `AmountDecimal` (the amount with its decimals, a `float64`), `Currency`, `AuthorizationURL` and `AuthorizationToken`; every other field the API returned is in `payment.Raw`. `Payment.Amount` drops the decimals and is deprecated.

A runnable version is in [`examples/create_payment/`](examples/create_payment/).

## Configuration

| Field | Description |
|-------|-------------|
| `APIKey` | Secret or restricted API key (`sk.…`, `sk_test.…`, `rk.…`, `rk_test.…`). Falls back to `WAJUB_API_KEY`. |
| `WebhookSecret` | Webhook signing secret (`whsec_…`), needed only for `Webhooks.ConstructEvent()`. Falls back to `WAJUB_WEBHOOK_SECRET`. |
| `HTTPClient` | Your own `*http.Client`, for a proxy, tracing or tests. |
| `Timeout` | Request timeout, a `time.Duration`. Ignored when `HTTPClient` is set: put the timeout on your own client. |
| `MaxNetworkRetries` | How many times a failed request is retried, as a `*int`. `nil` keeps the default, `0` turns retries off. |

`New` returns an error when neither `APIKey` nor `WAJUB_API_KEY` holds a key.

## Resources

| Purpose | Service | Methods |
|---------|---------|---------|
| Payments | `client.Payments` | `Create`, `Retrieve`, `List`, `Cancel`, `Process`, `ProcessSplit`, `Refund`, `RetrieveRefund`, `ListRefunds` |
| Refunds | `client.Refunds` | `Create`, `Retrieve`, `List` |
| Customers | `client.Customers` | `Create`, `Retrieve`, `Update`, `Delete`, `List`, `Block`, `Unblock`, `Activate`, `Deactivate`, `ListTaxIds`, `CreateTaxId`, `DeleteTaxId` |
| Payment links | `client.Links` | `Create`, `Retrieve`, `Update`, `Delete`, `List` |
| Invoices | `client.Invoices` | `Create`, `Retrieve`, `Update`, `Delete`, `List`, `Send`, `MarkPaid`, `Cancel` |
| Payouts | `client.Transfers` | `Create`, `Retrieve`, `List` |
| Payout recipients | `client.Beneficiaries` | `Create`, `Retrieve`, `Update`, `Delete`, `List` |
| Destination checks | `client.Identity` | `Resolve`, `Validate` |
| Balance | `client.Balance` | `Retrieve` |
| Disputes | `client.Disputes` | `List`, `Retrieve`, `SubmitEvidence`, `Accept`, `Close`, `SendMessage` |
| Events | `client.Events` | `List`, `Retrieve`, `Resend` |
| Webhook endpoints | `client.WebhookEndpoints` | `Create`, `Retrieve`, `Update`, `Delete`, `List`, `RotateSecret` |
| Webhook verification | `client.Webhooks` | `ConstructEvent` |
| Connected accounts (Sync) | `client.Accounts` | `Create`, `Retrieve`, `Update`, `Delete`, `List`, `RegenerateToken` |
| Tax | `client.Tax` | `GetSettings`, `UpdateSettings`, `Rates`, `Calculate`, `Reports`, `ListCodes`, `RetrieveCode`, `ListRegistrations`, `CreateRegistration`, `RetrieveRegistration`, `UpdateRegistration`, `DeleteRegistration`, `Jurisdictions`, `Thresholds`, `ThresholdAlerts` |
| Fraud (Shield) | `client.Shield` | `GetSettings`, `UpdateSettings`, `Stats`, `ListBlocklist`, `AddToBlocklist`, `RemoveFromBlocklist` |
| Reference data | `client.Global` | `Ping`, `Channels`, `Countries`, `Currencies` |

Every method takes a `context.Context` first. Parameters are a `map[string]any` with the API's field names, and the last argument, `*wajub.RequestOptions`, is usually `nil`.

`client.Payments.Refund` refunds a payment, in full or for the `amount` you pass, and `client.Payments.RetrieveRefund` reads a refund by its id. Both return a `wajub.Refund`, whose amount, decimals included, is in `AmountDecimal`:

```go
refund, err := client.Payments.Refund(ctx, payment.ID, map[string]any{
    "amount": 5000,
    "reason": "requested_by_customer",
}, nil)
```

## Acting for a connected account

Pass the connected account with `Sync` on the call:

```go
payment, err := client.Payments.Create(ctx, params, &wajub.RequestOptions{Sync: "acct_sync_ref"})
```

## Webhooks

Pass the **raw request body**, not parsed JSON, with the `X-Wajub-Signature` and `X-Wajub-Timestamp` headers:

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

    if event["event"] == "payment.succeeded" {
        // fulfil the order
    }
    w.WriteHeader(http.StatusOK)
}
```

A delivery that does not verify returns a `*wajub.WebhookSignatureVerificationError`. If your server's clock drifts, pass a tolerance such as `10 * time.Minute` as the last argument.

## Pagination

`List` returns a page you can walk one page at a time, or collect across all pages:

```go
page, err := client.Payments.List(ctx, map[string]any{"per_page": 50})
if err != nil {
    log.Fatal(err)
}
log.Println(page.Data, page.Meta, page.HasMore)

if page.HasMore {
    next, err := page.GetNextPage(ctx)
    if err != nil {
        log.Fatal(err)
    }
    log.Println(next.Data)
}

all, err := page.AutoPagingIter(ctx)
if err != nil {
    log.Fatal(err)
}
for _, payment := range all {
    log.Println(payment["id"], payment["status"])
}
```

Rows are `map[string]any`, read by key.

## Idempotency

Temporary failures are retried for you. To make a create safe to repeat on your side too, after a restart or in a job retry, pass your own key:

```go
payment, err := client.Payments.Create(ctx, params, &wajub.RequestOptions{
    IdempotencyKey: "order-" + orderID,
})
```

## Error handling

API errors are pointer types that embed `wajub.WajubError`; inspect them with `errors.As`:

```go
payment, err := client.Payments.Create(ctx, params, nil)
if err != nil {
    var invalid *wajub.InvalidRequestError
    if errors.As(err, &invalid) {
        log.Println(invalid.FieldErrors) // every validation message, by field
        return
    }
    var auth *wajub.AuthenticationError
    if errors.As(err, &auth) {
        // check your API key
        return
    }
    if apiErr, ok := wajub.AsAPIError(err); ok {
        log.Println(apiErr.Code, apiErr.HTTPStatus, apiErr.Message)
    }
    return
}
```

| Type | Returned on |
|------|-------------|
| `*AuthenticationError` | `401` |
| `*PermissionError` | `403` |
| `*NotFoundError` | `404` |
| `*InvalidRequestError` | `400`, `422` |
| `*RateLimitError` | `429`, with `RetryAfter` in seconds |
| `*WajubError` | Any other error status |
| `*APIConnectionError` | No response: network error or timeout; `errors.Is` reaches the cause |

Every error carries `Message`, `Code`, `HTTPStatus`, `FieldErrors` (every validation message of each field), `Errors` (the first message of each field) and `Raw` (the response body). `wajub.AsAPIError(err)` returns that shared `*WajubError` whatever the type.

A method that puts an id in the URL, such as `Retrieve`, `Update` or `Delete`, sends nothing when the id cannot be one: it returns `wajub.ErrEmptyID` for an empty id and `wajub.ErrInvalidID` for an id containing `/`. Check them with `errors.Is`.

## Testing

Use a sandbox key (`sk_test.…`): the key chooses the environment, and no real money moves. The test phone numbers and cards are on [Test scenarios](https://docs.wajub.com/developer-tools/testing/test-scenarios). To receive webhooks on your machine, forward them with the [Wajub CLI](https://docs.wajub.com/developer-tools/cli/listen).

## Documentation & support

- Documentation: [docs.wajub.com/libraries/sdks/go](https://docs.wajub.com/libraries/sdks/go)
- Report issues: [github.com/wajubhq/wajub-go/issues](https://github.com/wajubhq/wajub-go/issues)

## License

MIT — see [LICENSE](LICENSE).
