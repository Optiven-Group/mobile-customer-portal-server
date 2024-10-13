package payments

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	stripe "github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/paymentintent"
)

type CreatePaymentIntentRequest struct {
    Amount        int64  `json:"amount"`         // Amount in the smallest currency unit (e.g., cents)
    Currency      string `json:"currency"`       // e.g., "usd", "kes"
    CustomerEmail string `json:"customer_email"` // Optional: For sending receipts
}

func CreatePaymentIntent(c *gin.Context) {
    var req CreatePaymentIntentRequest

    if err := c.BindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }

    if req.Amount <= 0 || req.Currency == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Amount and currency are required"})
        return
    }

    // Set your Stripe secret key
    stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

    params := &stripe.PaymentIntentParams{
        Amount:   stripe.Int64(req.Amount),
        Currency: stripe.String(req.Currency),
        PaymentMethodTypes: stripe.StringSlice([]string{
            "card",
        }),
    }

    // Optionally, attach the customer's email for receipts
    if req.CustomerEmail != "" {
        params.ReceiptEmail = stripe.String(req.CustomerEmail)
    }

    pi, err := paymentintent.New(params)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "clientSecret": pi.ClientSecret,
    })
}
