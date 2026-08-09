package wajub

import "fmt"

// Payment represents a Wajub payment (transaction).
type Payment struct {
	ID                 string
	Status             string
	Amount             int64
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
	if amount, ok := transaction["amount"].(float64); ok {
		payment.Amount = int64(amount)
	}
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
	Amount int64
	Raw    map[string]any
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
	if amount, ok := data["amount"].(float64); ok {
		refund.Amount = int64(amount)
	}

	return refund
}
