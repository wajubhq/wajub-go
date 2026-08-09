// Create a payment with the Wajub Go SDK.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/wajubhq/wajub-go"
)

func main() {
	client, err := wajub.New(wajub.Config{
		APIKey: os.Getenv("WAJUB_API_KEY"),
	})
	if err != nil {
		panic(err)
	}

	payment, err := client.Payments.Create(context.Background(), map[string]any{
		"amount":   15000,
		"currency": "XAF",
		"email":    "buyer@example.com",
		"callback": "https://shop.example.com/complete",
	}, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Payment %s — redirect buyer to:\n%s\n", payment.ID, payment.AuthorizationURL)
}
