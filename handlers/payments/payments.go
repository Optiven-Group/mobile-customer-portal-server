package payments

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	darajago "github.com/oyamo/daraja-go"
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

type MpesaPaymentRequest struct {
	Amount                string `json:"amount"`
	PhoneNumber           string `json:"phone_number"`
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

func InitiateMpesaPayment(c *gin.Context) {
	var req MpesaPaymentRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Amount == "" || req.PhoneNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount and phone number are required"})
		return
	}

	// Initialize Daraja API client
	consumerKey := os.Getenv("DARAJA_CONSUMER_KEY")
	consumerSecret := os.Getenv("DARAJA_CONSUMER_SECRET")
	// Testing
	// daraja := darajago.NewDarajaApi(consumerKey, consumerSecret, darajago.ENVIRONMENT_SANDBOX)

	daraja := darajago.NewDarajaApi(consumerKey, consumerSecret, darajago.ENVIRONMENT_PRODUCTION)

	// Prepare the LipaNaMpesaPayload
	businessShortCode := os.Getenv("DARAJA_BUSINESS_SHORT_CODE")
	passKey := os.Getenv("DARAJA_PASSKEY")
	timestamp := time.Now().Format("20060102150405")
	passwordStr := businessShortCode + passKey + timestamp
	password := base64.StdEncoding.EncodeToString([]byte(passwordStr))

	lnmPayload := darajago.LipaNaMpesaPayload{
		BusinessShortCode: businessShortCode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            req.Amount,
		PartyA:            req.PhoneNumber,   // The MSISDN sending the funds
		PartyB:            businessShortCode, // The organization shortcode receiving the funds
		PhoneNumber:       req.PhoneNumber,   // The MSISDN sending the funds
		CallBackURL:       os.Getenv("DARAJA_CALLBACK_URL"),
		AccountReference:  req.CustomerNumber, // Use customer number as account reference
		TransactionDesc:   "Payment of Installment",
	}

	response, err := daraja.MakeSTKPushRequest(lnmPayload)
	if err != nil {
		log.Printf("Error initiating M-Pesa payment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate M-Pesa payment"})
		return
	}

	// Save the payment details
	mpesaPayment := models.MpesaPayment{
		CheckoutRequestID:     response.CheckoutRequestID,
		InstallmentScheduleID: req.InstallmentScheduleID,
		CustomerNumber:        req.CustomerNumber,
		PhoneNumber:           req.PhoneNumber,
		Amount:                req.Amount,
		Status:                "Pending",
	}

	if err := utils.CustomerPortalDB.Create(&mpesaPayment).Error; err != nil {
		log.Printf("Error saving M-Pesa payment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save M-Pesa payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             "M-Pesa payment initiated",
		"CheckoutRequestID":   response.CheckoutRequestID,
		"MerchantRequestID":   response.MerchantRequestID,
		"ResponseCode":        response.ResponseCode,
		"ResponseDescription": response.ResponseDescription,
		"CustomerMessage":     response.CustomerMessage,
	})
}

func MpesaCallback(c *gin.Context) {
	var callbackResponse darajago.CallbackResponse

	if err := c.BindJSON(&callbackResponse); err != nil {
		log.Printf("Error parsing M-Pesa callback: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback data"})
		return
	}

	stkCallback := callbackResponse.Body.StkCallback

	if stkCallback.ResultCode == 0 {
		// Payment successful
		log.Printf("M-Pesa payment successful: %v", stkCallback)

		// Extract necessary details
		checkoutRequestID := stkCallback.CheckoutRequestID
		// Removed unused variable merchantRequestID

		// Update the payment status
		if err := utils.CustomerPortalDB.Model(&models.MpesaPayment{}).Where("checkout_request_id = ?", checkoutRequestID).Updates(map[string]interface{}{
			"status": "Success",
		}).Error; err != nil {
			log.Printf("Failed to update M-Pesa payment status: %v", err)
		}

		isID := getInstallmentScheduleIDByCheckoutRequestID(checkoutRequestID)

		if isID == "" {
			log.Printf("Could not find InstallmentScheduleID for CheckoutRequestID: %s", checkoutRequestID)
			return
		}

		// Update the installment schedule
		if err := utils.CRMDB.Model(&models.InstallmentSchedule{}).Where("IS_id = ?", isID).Updates(map[string]interface{}{
			"paid": "Yes",
		}).Error; err != nil {
			log.Printf("Failed to update installment schedule: %v", err)
		} else {
			log.Printf("Successfully updated installment schedule ISID=%s to paid", isID)
		}

		// Fetch the user's push token
		// Retrieve customer number from MpesaPayment
		var mpesaPayment models.MpesaPayment
		if err := utils.CustomerPortalDB.Where("checkout_request_id = ?", checkoutRequestID).First(&mpesaPayment).Error; err != nil {
			log.Printf("Error finding M-Pesa payment: %v", err)
			return
		}
		customerNumber := mpesaPayment.CustomerNumber

		var user models.User
		if err := utils.CustomerPortalDB.Where("customer_number = ?", customerNumber).First(&user).Error; err != nil {
			log.Printf("Failed to find user: %v", err)
			return
		}

		if user.PushToken != "" {
			sendPushNotification(user.PushToken, "Payment Successful", "Your M-Pesa payment was successful.")
		} else {
			log.Printf("User does not have a push token")
		}
	} else {
		// Payment failed or cancelled
		log.Printf("M-Pesa payment failed or cancelled: %v", stkCallback)

		// Extract necessary details
		checkoutRequestID := stkCallback.CheckoutRequestID

		// Update the payment status to Failed
		if err := utils.CustomerPortalDB.Model(&models.MpesaPayment{}).Where("checkout_request_id = ?", checkoutRequestID).Updates(map[string]interface{}{
			"status": "Failed",
		}).Error; err != nil {
			log.Printf("Failed to update M-Pesa payment status: %v", err)
		}

		// Optionally, notify the user
		// Retrieve customer number from MpesaPayment
		var mpesaPayment models.MpesaPayment
		if err := utils.CustomerPortalDB.Where("checkout_request_id = ?", checkoutRequestID).First(&mpesaPayment).Error; err != nil {
			log.Printf("Error finding M-Pesa payment: %v", err)
			return
		}
		customerNumber := mpesaPayment.CustomerNumber

		var user models.User
		if err := utils.CustomerPortalDB.Where("customer_number = ?", customerNumber).First(&user).Error; err != nil {
			log.Printf("Failed to find user: %v", err)
			return
		}

		if user.PushToken != "" {
			sendPushNotification(user.PushToken, "Payment Failed", "Your M-Pesa payment failed or was cancelled.")
		} else {
			log.Printf("User does not have a push token")
		}
	}

	// Return 200 OK
	c.JSON(http.StatusOK, gin.H{"message": "Callback received"})
}

func getInstallmentScheduleIDByCheckoutRequestID(checkoutRequestID string) string {
	var mpesaPayment models.MpesaPayment
	if err := utils.CustomerPortalDB.Where("checkout_request_id = ?", checkoutRequestID).First(&mpesaPayment).Error; err != nil {
		log.Printf("Error finding M-Pesa payment: %v", err)
		return ""
	}
	return mpesaPayment.InstallmentScheduleID
}
