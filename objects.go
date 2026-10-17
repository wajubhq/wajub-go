package wajub

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Payment represents a Wajub payment (transaction).
type Payment struct {
	ID     string
	Status string
	// Amount is the amount without its decimals: GHS 12.50 reads 12.
	//
	// Deprecated: Amount drops the decimals of every currency that has them.
	// Read AmountDecimal.
	Amount int64
	// AmountDecimal is the amount in the currency's major unit, decimals
	// included, as the API sends it: 15000 for XAF 15,000, 12.5 for GHS 12.50.
	// Pass it back as is where the API takes an amount.
	AmountDecimal      float64
	Currency           string
	AuthorizationURL   string
	AuthorizationToken string
	Raw                map[string]any
}

func parsePayment(body map[string]any) Payment {
	transaction := pickResource(body, "transaction")
	payment := Payment{Raw: transaction}

	if id, ok := transaction["id"].(string); ok {
		payment.ID = id
	}
	if status, ok := transaction["status"].(string); ok {
		payment.Status = status
	}
	payment.AmountDecimal = decimalAmount(transaction["amount"])
	payment.Amount = int64(payment.AmountDecimal)
	if currency, ok := transaction["currency"].(string); ok {
		payment.Currency = currency
	}

	token := body["authorization_token"]
	if token == nil {
		token = transaction["authorization_token"]
	}
	if token == nil {
		token = transaction["id"]
	}
	url := body["authorization_url"]
	if url == nil {
		url = transaction["authorization_url"]
	}

	payment.AuthorizationToken = fmt.Sprint(token)
	payment.AuthorizationURL = fmt.Sprint(url)

	return payment
}

// Customer represents a Wajub customer.
type Customer struct {
	ID    string
	Email string
	Name  string
	Raw   map[string]any
}

func parseCustomer(body map[string]any) Customer {
	data := pickResource(body, "customer")
	customer := Customer{Raw: data}

	if id, ok := data["id"].(string); ok {
		customer.ID = id
	}
	if email, ok := data["email"].(string); ok {
		customer.Email = email
	}
	if name, ok := data["name"].(string); ok {
		customer.Name = name
	}

	return customer
}

// Refund represents a Wajub refund.
type Refund struct {
	ID     string
	Status string
	// Amount is the amount without its decimals: GHS 12.50 reads 12.
	//
	// Deprecated: Amount drops the decimals of every currency that has them.
	// Read AmountDecimal.
	Amount int64
	// AmountDecimal is the amount in the currency's major unit, decimals
	// included, as the API sends it: 12.5 for GHS 12.50.
	AmountDecimal float64
	// Currency is the refund's ISO 4217 currency code, such as "GHS".
	Currency string
	Raw      map[string]any
}

func parseRefund(body map[string]any) Refund {
	data := pickResource(body, "refund")
	refund := Refund{Raw: data}

	if id, ok := data["id"].(string); ok {
		refund.ID = id
	}
	if status, ok := data["status"].(string); ok {
		refund.Status = status
	}
	refund.AmountDecimal = decimalAmount(data["amount"])
	refund.Amount = int64(refund.AmountDecimal)
	if currency, ok := data["currency"].(string); ok {
		refund.Currency = currency
	}

	return refund
}

// decimalAmount reads an amount as the API sends it, a JSON number or a
// numeric string ("12.50"), in the currency's major unit. Anything else, or no
// amount, reads 0.
func decimalAmount(value any) float64 {
	var amount float64
	switch v := value.(type) {
	case float64:
		amount = v
	case int:
		amount = float64(v)
	case int64:
		amount = float64(v)
	case json.Number:
		parsed, err := v.Float64()
		if err != nil {
			return 0
		}
		amount = parsed
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0
		}
		amount = parsed
	}
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0
	}
	return amount
}
