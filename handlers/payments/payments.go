package payments

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	mpesa "github.com/jwambugu/mpesa-golang-sdk"
)

type MpesaPaymentRequest struct {
    Amount                string `json:"amount"`
    PhoneNumber           string `json:"phone_number"`
    InstallmentScheduleID string `json:"installment_schedule_id"`
    CustomerNumber        string `json:"customer_number"`
}

// InitiateMpesaPayment handles the initiation of an M-PESA STK Push payment.
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

    // Convert amount to integer
    amount, err := strconv.Atoi(req.Amount)
    if err != nil || amount <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount format"})
        return
    }

    // Validate phone number format (must be numeric and start with country code)
    phoneNumber, err := strconv.ParseUint(req.PhoneNumber, 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number format"})
        return
    }

    // Initialize Mpesa client
    consumerKey := os.Getenv("DARAJA_CONSUMER_KEY")
    consumerSecret := os.Getenv("DARAJA_CONSUMER_SECRET")
    passKey := os.Getenv("DARAJA_PASSKEY")
    callbackURL := os.Getenv("DARAJA_CALLBACK_URL")

    if consumerKey == "" || consumerSecret == "" || passKey == "" || callbackURL == "" {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "M-PESA configuration not properly set"})
        return
    }

    // mpesaClient := mpesa.NewApp(http.DefaultClient, consumerKey, consumerSecret, mpesa.EnvironmentProduction)
    mpesaClient := mpesa.NewApp(http.DefaultClient, consumerKey, consumerSecret, mpesa.EnvironmentSandbox)


    // Prepare the STK Push request
    businessShortCodeStr := os.Getenv("DARAJA_BUSINESS_SHORT_CODE")
    businessShortCode, err := strconv.Atoi(businessShortCodeStr)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid business shortcode"})
        return
    }

    timestamp := time.Now().Format("20060102150405")
    passwordStr := businessShortCodeStr + passKey + timestamp
    password := base64.StdEncoding.EncodeToString([]byte(passwordStr))

    stkPushRequest := mpesa.STKPushRequest{
        BusinessShortCode: uint(businessShortCode),
        Password:          password,
        Timestamp:         timestamp,
        TransactionType:   mpesa.CustomerPayBillOnlineTransactionType,
        Amount:            uint(amount),
        PartyA:            uint(phoneNumber),
        PartyB:            uint(businessShortCode),
        PhoneNumber:       phoneNumber,
        CallBackURL:       callbackURL,
        AccountReference:  req.CustomerNumber,
        TransactionDesc:   "Payment of Installment",
    }

    // Initiate the STK Push
    response, err := mpesaClient.STKPush(context.Background(), passKey, stkPushRequest)
    if err != nil {
        log.Printf("Error initiating M-PESA payment: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate M-PESA payment"})
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
        log.Printf("Error saving M-PESA payment: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save M-PESA payment"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message":             "M-PESA payment initiated",
        "CheckoutRequestID":   response.CheckoutRequestID,
        "MerchantRequestID":   response.MerchantRequestID,
        "ResponseCode":        response.ResponseCode,
        "ResponseDescription": response.ResponseDescription,
        "CustomerMessage":     response.CustomerMessage,
    })
}

// MpesaCallback handles the M-PESA STK Push callback.
func MpesaCallback(c *gin.Context) {
    var callback mpesa.STKPushCallback

    // Read the request body
    bodyBytes, err := ioutil.ReadAll(c.Request.Body)
    if err != nil {
        log.Printf("Error reading callback body: %v", err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback data"})
        return
    }

    // Unmarshal the callback
    if err := json.Unmarshal(bodyBytes, &callback); err != nil {
        log.Printf("Error parsing M-PESA callback: %v", err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback data"})
        return
    }

    stkCallback := callback.Body.STKCallback

    if stkCallback.ResultCode == 0 {
        // Payment successful
        log.Printf("M-PESA payment successful: %+v", stkCallback)

        // Extract necessary details
        checkoutRequestID := stkCallback.CheckoutRequestID

        // Update the payment status
        if err := utils.CustomerPortalDB.Model(&models.MpesaPayment{}).
            Where("checkout_request_id = ?", checkoutRequestID).
            Updates(map[string]interface{}{
                "status": "Success",
            }).Error; err != nil {
            log.Printf("Failed to update M-PESA payment status: %v", err)
        }

        isID := getInstallmentScheduleIDByCheckoutRequestID(checkoutRequestID)

        if isID == "" {
            log.Printf("Could not find InstallmentScheduleID for CheckoutRequestID: %s", checkoutRequestID)
            return
        }

        // Update the installment schedule
        if err := utils.CRMDB.Model(&models.InstallmentSchedule{}).
            Where("IS_id = ?", isID).
            Updates(map[string]interface{}{
                "paid": "Yes",
            }).Error; err != nil {
            log.Printf("Failed to update installment schedule: %v", err)
        } else {
            log.Printf("Successfully updated installment schedule ISID=%s to paid", isID)
        }

        // Fetch the user's push token
        var mpesaPayment models.MpesaPayment
        if err := utils.CustomerPortalDB.
            Where("checkout_request_id = ?", checkoutRequestID).
            First(&mpesaPayment).Error; err != nil {
            log.Printf("Error finding M-PESA payment: %v", err)
            return
        }
        customerNumber := mpesaPayment.CustomerNumber

        var user models.User
        if err := utils.CustomerPortalDB.
            Where("customer_number = ?", customerNumber).
            First(&user).Error; err != nil {
            log.Printf("Failed to find user: %v", err)
            return
        }

        if user.PushToken != "" {
            sendPushNotification(user.PushToken, "Payment Successful", "Your M-PESA payment was successful.")
        } else {
            log.Printf("User does not have a push token")
        }
    } else {
        // Payment failed or cancelled
        log.Printf("M-PESA payment failed or cancelled: %+v", stkCallback)

        // Extract necessary details
        checkoutRequestID := stkCallback.CheckoutRequestID

        // Update the payment status to Failed
        if err := utils.CustomerPortalDB.Model(&models.MpesaPayment{}).
            Where("checkout_request_id = ?", checkoutRequestID).
            Updates(map[string]interface{}{
                "status": "Failed",
            }).Error; err != nil {
            log.Printf("Failed to update M-PESA payment status: %v", err)
        }

        // Optionally, notify the user
        var mpesaPayment models.MpesaPayment
        if err := utils.CustomerPortalDB.
            Where("checkout_request_id = ?", checkoutRequestID).
            First(&mpesaPayment).Error; err != nil {
            log.Printf("Error finding M-PESA payment: %v", err)
            return
        }
        customerNumber := mpesaPayment.CustomerNumber

        var user models.User
        if err := utils.CustomerPortalDB.
            Where("customer_number = ?", customerNumber).
            First(&user).Error; err != nil {
            log.Printf("Failed to find user: %v", err)
            return
        }

        if user.PushToken != "" {
            sendPushNotification(user.PushToken, "Payment Failed", "Your M-PESA payment failed or was cancelled.")
        } else {
            log.Printf("User does not have a push token")
        }
    }

    // Return 200 OK
    c.JSON(http.StatusOK, gin.H{"message": "Callback received"})
}

// getInstallmentScheduleIDByCheckoutRequestID retrieves the InstallmentScheduleID using the CheckoutRequestID.
func getInstallmentScheduleIDByCheckoutRequestID(checkoutRequestID string) string {
    var mpesaPayment models.MpesaPayment
    if err := utils.CustomerPortalDB.
        Where("checkout_request_id = ?", checkoutRequestID).
        First(&mpesaPayment).Error; err != nil {
        log.Printf("Error finding M-PESA payment: %v", err)
        return ""
    }
    return mpesaPayment.InstallmentScheduleID
}

// sendPushNotification sends a push notification to the user.
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
