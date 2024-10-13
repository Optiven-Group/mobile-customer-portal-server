package payments

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	stripe "github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/paymentintent"
	"github.com/stripe/stripe-go/webhook"
)

type CreatePaymentIntentRequest struct {
	Amount                int64  `json:"amount"`
	Currency              string `json:"currency"`
	CustomerEmail         string `json:"customer_email"`
	InstallmentScheduleID string `json:"installment_schedule_id"`
	CustomerNumber        string `json:"customer_number"`
}

func HandleStripeWebhook(c *gin.Context) {
	const MaxBodyBytes = int64(65536)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)
	payload, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.Writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	event, err := webhook.ConstructEvent(payload, c.Request.Header.Get("Stripe-Signature"), endpointSecret)
	if err != nil {
		log.Printf("⚠️  Webhook signature verification failed. %v\n", err)
		c.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	if event.Type == "payment_intent.succeeded" {
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			c.Writer.WriteHeader(http.StatusBadRequest)
			return
		}

		// Update installment_schedule in the database
		handlePaymentSuccess(paymentIntent)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func handlePaymentSuccess(paymentIntent stripe.PaymentIntent) {
    isID := paymentIntent.Metadata["installment_schedule_id"]
    customerNumber := paymentIntent.Metadata["customer_number"]

    if isID == "" {
        log.Printf("PaymentIntent does not have installment_schedule_id in metadata")
        return
    }

    if customerNumber == "" {
        log.Printf("PaymentIntent does not have customer_number in metadata")
        return
    }

    if err := utils.CRMDB.Model(&models.InstallmentSchedule{}).Where("IS_id = ?", isID).Updates(map[string]interface{}{
        "paid": "Yes",
    }).Error; err != nil {
        log.Printf("Failed to update installment schedule: %v", err)
    } else {
        log.Printf("Successfully updated installment schedule ISID=%s to paid", isID)
    }

    // Fetch the user's push token
    var user models.User
    if err := utils.CustomerPortalDB.Where("customer_number = ?", customerNumber).First(&user).Error; err != nil {
        log.Printf("Failed to find user: %v", err)
        return
    }

    if user.PushToken != "" {
        sendPushNotification(user.PushToken, "Payment Successful", "Your payment was successful.")
    } else {
        log.Printf("User does not have a push token")
    }
}

func sendPushNotification(pushToken, title, message string) {
	notification := map[string]interface{}{
		"to":    pushToken,
		"sound": "default",
		"title": title,
		"body":  message,
	}

	payload, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Failed to marshal notification payload: %v", err)
		return
	}

	resp, err := http.Post("https://exp.host/--/api/v2/push/send", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Failed to send push notification: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Failed to send push notification, status: %d, response: %s", resp.StatusCode, string(bodyBytes))
	} else {
		log.Printf("Push notification sent successfully to %s", pushToken)
	}
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

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.Amount),
		Currency: stripe.String(req.Currency),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
	}

	if req.CustomerEmail != "" {
		params.ReceiptEmail = stripe.String(req.CustomerEmail)
	}

	// Add metadata
	params.Metadata = map[string]string{
		"installment_schedule_id": req.InstallmentScheduleID,
		"customer_number":         req.CustomerNumber,
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