# Changelog

All notable changes to `github.com/wajubhq/wajub-go` are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
The module uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.3.0] — 2026-10-15

### Added
- **`Payment.AmountDecimal` and `Refund.AmountDecimal`, the amount with its
  decimals.** Amounts are in the currency's main unit, so GHS 12.50 is `12.5`.
  `AmountDecimal` is a `float64` that keeps it: `12.5` for GHS 12.50, `15000`
  for XAF 15,000. Put it back in a request as is, for a refund for instance,
  and it goes out with its decimals.
- **`FieldErrors` on API errors, every validation message of each field.** A
  `422` can give a field several messages; `Errors` keeps only the first one.
  `invalid.FieldErrors["amount"]` lists them all, in the order the API sent
  them. `Errors` is unchanged.
- **`Payments.Refund` and `Payments.RetrieveRefund`, which return a
  `Refund`.** `Refund` refunds a payment, in full or for the `amount` you
  pass; `RetrieveRefund` reads a refund by its id. `Refund` now has
  `Currency`. `client.Refunds` still returns maps.

### Deprecated
- **`Payment.Amount` and `Refund.Amount`.** They are `int64` and drop the
  decimals: GHS 12.50 reads `12`. They still hold that value, so code that
  reads them compiles and behaves as before. Read `AmountDecimal` instead.
- **`client.Listen.Config` and `client.Listen.Auth`.** The API has retired
  them and answers every call with `410 Gone`. To receive webhooks on your
  machine, use the Wajub CLI's `wajub listen`.
- **`client.Refunds.Update`, `client.Refunds.Delete`,
  `client.Transfers.Update` and `client.Transfers.Delete`.** A refund or a
  payout cannot be changed or deleted, so they always return an error. Your
  editor does not flag these calls: remove them.

### Fixed
- An id containing `/`, `?` or `#`, or an id of `.` or `..`, sent the request
  to another route: `client.Payments.Retrieve(ctx, "pay_1/refunds")` listed
  the refunds of `pay_1`, and `client.Customers.Delete(ctx,
  "cus_1/tax_ids/txi_1", nil)` deleted a tax id. Wajub ids never contain `/`:
  an id with one now returns `wajub.ErrInvalidID` before anything is sent.
  `?`, `#`, `.` and `..` are escaped, and the request reaches the resource the
  id names.
- An empty id left its segment out of the URL:
  `client.Payments.Retrieve(ctx, "")` asked for the list of payments, and
  `client.Customers.Delete(ctx, "", nil)` sent `DELETE /customers/`. Every
  method that puts an id in the URL, such as `Retrieve`, `Update` and
  `Delete`, now returns `wajub.ErrEmptyID` for an empty one, before anything
  is sent.
- The README said amounts are passed in the smallest currency unit (cents).
  Amounts are in the currency's main unit: send `12.50` for GHS 12.50, not
  `1250`. Currencies without decimals, such as XAF and XOF, are not affected.

## [1.2.1] — 2026-10-12

### Fixed
- **An automatic retry no longer creates anything twice.** A call retried
  after a timeout or a server error could create the payment, the refund or
  the transfer a second time.

### Changed
- **A create sent again with the same idempotency key while the first is still
  running** now returns the first request's result instead of a `409` error.

## [1.2.0] — 2026-10-01

Install 1.2.1, which carries this change.

### Removed
- `Payments.Initialize`, an alias of `Payments.Create`. Call `Create`.

## [1.1.1] — 2026-08-09

### Changed
- README corrections; no change to the module.

## [1.1.0] — 2026-08-09

### Added
- Initial release.
