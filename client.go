package wajub

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// Client is the Wajub server SDK entry point.
type Client struct {
	Global           *GlobalService
	Payments         *PaymentsService
	Customers        *CustomersService
	Refunds          *CrudService
	Transfers        *CrudService
	Beneficiaries    *CrudService
	Links            *CrudService
	Balance          *BalanceService
	Events           *EventsService
	Accounts         *AccountsService
	WebhookEndpoints *WebhookEndpointsService
	Invoices         *InvoicesService
	Disputes         *DisputesService
	Identity         *IdentityService
	Tax              *TaxService
	Shield           *ShieldService
	Listen           *ListenService
	Webhooks         *Webhooks
}

// Config configures a new Client.
type Config struct {
	APIKey               string
	WebhookSecret        string
	IdempotencyKeyPrefix string
	HTTPClient           *http.Client
	// MaxNetworkRetries overrides the number of automatic retries for transient
	// network failures and retryable HTTP statuses. Defaults to 2 when nil.
	MaxNetworkRetries *int
	// Timeout overrides the default 30s HTTP client timeout. Ignored when
	// HTTPClient is set — configure the timeout on that client instead.
	Timeout time.Duration
}

// New creates a Wajub API client.
func New(cfg Config) (*Client, error) {
	key := normalizeAPIKey(cfg.APIKey)
	if key == "" {
		key = normalizeAPIKey(os.Getenv("WAJUB_API_KEY"))
	}
	if key == "" {
		return nil, fmt.Errorf("wajub: api key is required")
	}

	baseURL := defaultAPIURL()
	prefix := cfg.IdempotencyKeyPrefix
	if prefix == "" {
		prefix = "wajub"
	}

	base := newBaseClient(key, baseURL, prefix, cfg.HTTPClient, cfg.MaxNetworkRetries, cfg.Timeout)

	whSecret := cfg.WebhookSecret
	if whSecret == "" {
		whSecret = os.Getenv("WAJUB_WEBHOOK_SECRET")
	}

	return &Client{
		Global:           &GlobalService{base: base},
		Payments:         &PaymentsService{base: base},
		Customers:        &CustomersService{CrudService{base: base, path: "customers", singular: "customer", plural: "customers"}},
		Refunds:          &CrudService{base: base, path: "refunds", singular: "refund", plural: "refunds"},
		Transfers:        &CrudService{base: base, path: "transfers", singular: "transfer", plural: "transfers"},
		Beneficiaries:    &CrudService{base: base, path: "beneficiaries", singular: "beneficiary", plural: "beneficiaries"},
		Links:            &CrudService{base: base, path: "links", singular: "link", plural: "links"},
		Balance:          &BalanceService{base: base},
		Events:           &EventsService{base: base},
		Accounts:         &AccountsService{CrudService{base: base, path: "accounts", singular: "account", plural: "accounts"}},
		WebhookEndpoints: &WebhookEndpointsService{CrudService{base: base, path: "webhooks", singular: "endpoint", plural: "endpoints"}},
		Invoices:         &InvoicesService{CrudService{base: base, path: "invoices", singular: "invoice", plural: "invoices"}},
		Disputes:         &DisputesService{base: base},
		Identity:         &IdentityService{base: base},
		Tax:              &TaxService{base: base},
		Shield:           &ShieldService{base: base},
		Listen:           &ListenService{base: base},
		Webhooks:         &Webhooks{Secret: whSecret},
	}, nil
}
