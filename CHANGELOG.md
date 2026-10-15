# Changelog

All notable changes to `github.com/wajubhq/wajub-go` are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
The module uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.3.0] — 2026-10-15

### Added
- **`Payment.AmountDecimal` and `Refund.AmountDecimal`, the amount with its
  decimals.** The API sends amounts in the major unit of the currency, so
  GHS 12.50 arrives as `12.5`. `AmountDecimal` is a `float64` that keeps it:
  `12.5` for GHS 12.50, `15000` for XAF 15,000. An amount sent as a string
  (`"12.50"`) reads the same. Put it back in a request as is, for a refund for
  instance, and it goes out with its cents.

### Deprecated
- **`Payment.Amount` and `Refund.Amount`.** They are `int64` and drop the
  decimals: GHS 12.50 reads `12`. They still hold that value, so code that
  reads them compiles and behaves as before. Read `AmountDecimal` instead.

### Fixed
- The README said amounts are passed in the smallest currency unit (cents).
  The API reads them in the major unit: send `12.50` for GHS 12.50, not `1250`.
  Zero-decimal currencies such as XAF and XOF are not affected.

## [1.2.1] — 2026-10-12

### Fixed
- **A retried POST or PUT reuses its `Idempotency-Key`.** The key the client
  generates when you pass none was drawn again on every attempt, so a call
  retried after a timeout, a `429` or a `5xx` reached the API as a new request
  and could create the payment, the refund or the transfer twice. It is now
  chosen once, before the first attempt.

### Changed
- **The `409` for an `Idempotency-Key` still being processed is retried.** A
  call sent again after a timeout can reach the API while the first one is
  still running; the API answers `409` with `errors.idempotency_key`. The
  client now waits 2 then 4 seconds and asks again with the same key, within
  its usual retries (`MaxNetworkRetries`, 2 by default), and returns the first
  request's result. Every other `409` is still returned at once.

## [1.2.0] — 2026-10-01

Tagged, never published to the Go module proxy. Its change ships with 1.2.1.

### Removed
- `Payments.Initialize`, an alias of `Payments.Create`. Call `Create`.

## [1.1.1] — 2026-08-09

Last version published before this changelog was started.
