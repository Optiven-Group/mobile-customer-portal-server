package payments

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	mpesa "github.com/jwambugu/mpesa-golang-sdk"
)


type MpesaPaymentRequest struct {
    Amount                string `json:"amount"`
    PhoneNumber           string `json:"phone_number"`
    InstallmentScheduleID string `json:"installment_schedule_id"`
    CustomerNumber        string `json:"customer_number"`
    PlotNumber            string `json:"plot_number"`
}

func isValidPhoneNumber(phoneNumber string) bool {
    // Check that the phone number is numeric and starts with '2547' and is 12 digits long
    if len(phoneNumber) != 12 {
        return false
    }
    if !strings.HasPrefix(phoneNumber, "2547") {
        return false
    }
    _, err := strconv.ParseUint(phoneNumber, 10, 64)
    return err == nil
}

type STKPushRequest struct {
    BusinessShortCode string `json:"BusinessShortCode"`
    Password          string `json:"Password"`
    Timestamp         string `json:"Timestamp"`
    TransactionType   string `json:"TransactionType"`
    Amount            int    `json:"Amount"`
    PartyA            string `json:"PartyA"`
    PartyB            string `json:"PartyB"`
    PhoneNumber       string `json:"PhoneNumber"`
    CallBackURL       string `json:"CallBackURL"`
    AccountReference  string `json:"AccountReference"`
    TransactionDesc   string `json:"TransactionDesc"`
}

func getAccessToken(consumerKey, consumerSecret string) (string, error) {
    url := "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"

    client := &http.Client{}
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return "", err
    }

    // Set Basic Auth header
    auth := base64.StdEncoding.EncodeToString([]byte(consumerKey + ":" + consumerSecret))
    req.Header.Add("Authorization", "Basic "+auth)

    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("failed to get access token: %s", string(bodyBytes))
    }

    var result map[string]interface{}
    err = json.Unmarshal(bodyBytes, &result)
    if err != nil {
        return "", err
    }

    accessToken, ok := result["access_token"].(string)
    if !ok {
        return "", fmt.Errorf("access_token not found in response")
    }

    return accessToken, nil
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

    // Validate phone number format
    if !isValidPhoneNumber(req.PhoneNumber) {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number format"})
        return
    }

    // Initialize variables
    consumerKey := os.Getenv("DARAJA_CONSUMER_KEY")
    consumerSecret := os.Getenv("DARAJA_CONSUMER_SECRET")
    passKey := os.Getenv("DARAJA_PASSKEY")
    callbackURL := os.Getenv("DARAJA_CALLBACK_URL")

    if consumerKey == "" || consumerSecret == "" || passKey == "" || callbackURL == "" {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "M-PESA configuration not properly set"})
        return
    }

    // Get access token
    accessToken, err := getAccessToken(consumerKey, consumerSecret)
    if err != nil {
        log.Printf("Error getting access token: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get access token"})
        return
    }

    // Prepare the STK Push request
    businessShortCode := os.Getenv("DARAJA_BUSINESS_SHORT_CODE")
    timestamp := time.Now().Format("20060102150405")
    passwordStr := businessShortCode + passKey + timestamp
    password := base64.StdEncoding.EncodeToString([]byte(passwordStr))

    stkPushRequest := STKPushRequest{
        BusinessShortCode: businessShortCode,
        Password:          password,
        Timestamp:         timestamp,
        TransactionType:   "CustomerPayBillOnline",
        Amount:            amount,
        PartyA:            req.PhoneNumber,
        PartyB:            businessShortCode,
        PhoneNumber:       req.PhoneNumber,
        CallBackURL:       callbackURL,
        AccountReference:  req.PlotNumber,
        TransactionDesc:   "Payment of Installment",
    }

    // Marshal the request to JSON
    requestBody, err := json.Marshal(stkPushRequest)
    if err != nil {
        log.Printf("Error marshalling STKPushRequest: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate M-PESA payment"})
        return
    }

    // Send the HTTP request
    stkPushURL := "https://api.safaricom.co.ke/mpesa/stkpush/v1/processrequest"
    reqHTTP, err := http.NewRequest("POST", stkPushURL, bytes.NewBuffer(requestBody))
    if err != nil {
        log.Printf("Error creating HTTP request: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate M-PESA payment"})
        return
    }
    reqHTTP.Header.Set("Content-Type", "application/json")
    reqHTTP.Header.Set("Authorization", "Bearer "+accessToken)

    client := &http.Client{}
    resp, err := client.Do(reqHTTP)
    if err != nil {
        log.Printf("Error sending STK Push request: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate M-PESA payment"})
        return
    }
    defer resp.Body.Close()

    responseBody, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Printf("Error reading response body: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate M-PESA payment"})
        return
    }

    // Parse the response
    var stkPushResponse map[string]interface{}
    err = json.Unmarshal(responseBody, &stkPushResponse)
    if err != nil {
        log.Printf("Error unmarshalling STK Push response: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate M-PESA payment"})
        return
    }

    // Check for errors in the response
    if resp.StatusCode != http.StatusOK {
        log.Printf("Error from M-PESA API: %s", string(responseBody))
        c.JSON(http.StatusInternalServerError, gin.H{"error": stkPushResponse["errorMessage"]})
        return
    }

    // Save the payment details
    checkoutRequestID, _ := stkPushResponse["CheckoutRequestID"].(string)
    merchantRequestID, _ := stkPushResponse["MerchantRequestID"].(string)

    mpesaPayment := models.MpesaPayment{
        CheckoutRequestID:     checkoutRequestID,
        InstallmentScheduleID: req.InstallmentScheduleID,
        CustomerNumber:        req.CustomerNumber,
        PhoneNumber:           req.PhoneNumber,
        Amount:                req.Amount,
        Status:                "Pending",
        PlotNumber:            req.PlotNumber,
    }

    if err := utils.CustomerPortalDB.Create(&mpesaPayment).Error; err != nil {
        log.Printf("Error saving M-PESA payment: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save M-PESA payment"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message":             "M-PESA payment initiated",
        "CheckoutRequestID":   checkoutRequestID,
        "MerchantRequestID":   merchantRequestID,
        "ResponseCode":        stkPushResponse["ResponseCode"],
        "ResponseDescription": stkPushResponse["ResponseDescription"],
        "CustomerMessage":     stkPushResponse["CustomerMessage"],
    })
}

// MpesaCallback handles the M-PESA STK Push callback.
func MpesaCallback(c *gin.Context) {
    var callback mpesa.STKPushCallback

    // Read the request body
    bodyBytes, err := io.ReadAll(c.Request.Body)
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
    
        checkoutRequestID := stkCallback.CheckoutRequestID
    
        // Update the M-Pesa payment status to Success
        if err := utils.CustomerPortalDB.Model(&models.MpesaPayment{}).
            Where("checkout_request_id = ?", checkoutRequestID).
            Updates(map[string]interface{}{"status": "Success"}).Error; err != nil {
            log.Printf("Failed to update M-PESA payment status: %v", err)
        }
    
        isID := getInstallmentScheduleIDByCheckoutRequestID(checkoutRequestID)
        if isID == "" {
            log.Printf("Could not find InstallmentScheduleID for CheckoutRequestID: %s", checkoutRequestID)
            return
        }
    
        // Retrieve the mpesaPayment record to know how much was paid
        var mpesaPayment models.MpesaPayment
        if err := utils.CustomerPortalDB.
            Where("checkout_request_id = ?", checkoutRequestID).
            First(&mpesaPayment).Error; err != nil {
            log.Printf("Error finding M-PESA payment: %v", err)
            return
        }
    
        // Fetch the installment schedule
        var schedule models.InstallmentSchedule
        if err := utils.CRMDB.Where("IS_id = ?", isID).First(&schedule).Error; err != nil {
            log.Printf("Failed to fetch installment schedule: %v", err)
            return
        }
    
        // Parse amounts
        installmentAmount, _ := strconv.ParseFloat(strings.ReplaceAll(schedule.InstallmentAmount, ",", ""), 64)
        currentAmountPaid, _ := strconv.ParseFloat(strings.ReplaceAll(schedule.AmountPaid, ",", ""), 64)
        paymentAmount, _ := strconv.ParseFloat(strings.ReplaceAll(mpesaPayment.Amount, ",", ""), 64)
    
        newAmountPaid := currentAmountPaid + paymentAmount
        remainingAmount := installmentAmount - newAmountPaid
    
        // Determine if fully paid or not
        paidStatus := "No"
        if remainingAmount <= 0 {
            // Fully paid
            remainingAmount = 0
            paidStatus = "Yes"
        }
    
        // Update the installment schedule
        updates := map[string]interface{}{
            "amount_paid":      fmt.Sprintf("%.2f", newAmountPaid),
            "remaining_amount": fmt.Sprintf("%.2f", remainingAmount),
            "paid":             paidStatus,
        }
    
        if err := utils.CRMDB.Model(&models.InstallmentSchedule{}).
            Where("IS_id = ?", isID).
            Updates(updates).Error; err != nil {
            log.Printf("Failed to update installment schedule: %v", err)
        } else {
            if paidStatus == "Yes" {
                log.Printf("Installment schedule ISID=%s fully paid", isID)
            } else {
                log.Printf("Installment schedule ISID=%s updated but still not fully paid", isID)
            }
        }
    
        // Notify the user
        customerNumber := mpesaPayment.CustomerNumber
        var user models.User
        if err := utils.CustomerPortalDB.
            Where("customer_number = ?", customerNumber).
            First(&user).Error; err != nil {
            log.Printf("Failed to find user: %v", err)
            return
        }
    
        var message string
        if paidStatus == "Yes" {
            // Full payment message
            message = fmt.Sprintf("Your payment of KES %s for plot %s was successful and your installment is now fully settled. Thank you!",
                mpesaPayment.Amount, mpesaPayment.PlotNumber)
        } else {
            // Partial payment message
            message = fmt.Sprintf("We’ve received your payment of KES %s for plot %s. Your remaining balance is KES %.2f. Keep going!",
                mpesaPayment.Amount, mpesaPayment.PlotNumber, remainingAmount)
        }
    
        if user.PushToken != "" {
            sendPushNotification(user.PushToken, "Payment Update", message)
        } else {
            log.Printf("User does not have a push token")
        }
    
        // Save notification
        notification := models.Notification{
            UserID: user.ID,
            Title:  "Payment Update",
            Body:   message,
            Data:   "",
        }
    
        if err := utils.CustomerPortalDB.Create(&notification).Error; err != nil {
            log.Printf("Failed to save notification: %v", err)
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

        // Send push notification and save notification
        if user.PushToken != "" {
            sendPushNotification(user.PushToken, "Payment Failed", "Your M-PESA payment failed or was cancelled.")
        } else {
            log.Printf("User does not have a push token")
        }

        // Save notification to database
        notification := models.Notification{
            UserID: user.ID,
            Title:  "Payment Failed",
            Body:   "Your M-PESA payment failed or was cancelled.",
            Data:   "",
        }

        if err := utils.CustomerPortalDB.Create(&notification).Error; err != nil {
            log.Printf("Failed to save notification: %v", err)
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
        bodyBytes, _ := io.ReadAll(resp.Body)
        log.Printf("Failed to send push notification, status: %d, response: %s", resp.StatusCode, string(bodyBytes))
    } else {
        log.Printf("Push notification sent successfully to %s", pushToken)
    }
}
