package handlers

import (
	"math/rand"
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// OTP validity duration
const otpValidityDuration = 10 * time.Minute

// generateOTP generates a 6-digit OTP
func generateOTP() string {
    source := rand.NewSource(time.Now().UnixNano())
    r := rand.New(source)
    const letters = "0123456789"
    otp := make([]byte, 6)
    for i := range otp {
        otp[i] = letters[r.Intn(len(letters))]
    }
    return string(otp)
}

// sendOTPEmail sends the OTP to the user's email
func sendOTPEmail(email string, otp string) {
    // Implement email sending logic here
}

// sendOTPWhatsApp sends the OTP to the user's phone number
func sendOTPWhatsApp(phoneNumber string, otp string) {
    // Implement WhatsApp sending logic here
}

// Login handles user login requests
func Login(c *gin.Context) {
    var input struct {
        CustomerNumber string `json:"customer_number"`
        EmailOrPhone   string `json:"email_or_phone"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var user models.User
    // Find user by customer number and either email or phone
    if err := utils.CustomerPortalDB.Where("customer_number = ? AND (email = ? OR phone_number = ?)", input.CustomerNumber, input.EmailOrPhone, input.EmailOrPhone).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid customer number or contact information"})
        return
    }

    // Generate and send OTP
    otp := generateOTP()
    user.OTP = otp
    user.OTPGeneratedAt = time.Now()

    if input.EmailOrPhone == user.Email {
        sendOTPEmail(user.Email, otp)
    } else if input.EmailOrPhone == user.PhoneNumber {
        sendOTPWhatsApp(user.PhoneNumber, otp)
    }

    // Save OTP in the user struct (not in DB, to avoid security risks)
    utils.CustomerPortalDB.Save(&user)

    c.JSON(http.StatusOK, gin.H{"message": "OTP sent", "otp_sent": true})
}

// VerifyOTP handles OTP verification and completes the password reset process
func VerifyOTP(c *gin.Context) {
    var input struct {
        CustomerNumber string `json:"customer_number"`
        OTP            string `json:"otp"`
        NewPassword    string `json:"new_password"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var user models.User
    if err := utils.CustomerPortalDB.Where("customer_number = ?", input.CustomerNumber).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid customer number"})
        return
    }

    // Check OTP validity
    if time.Now().After(user.OTPGeneratedAt.Add(otpValidityDuration)) || user.OTP != input.OTP {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
        return
    }

    // Hash and set the new password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
        return
    }

    user.Password = string(hashedPassword)
    user.Verified = true
    user.InitialSetup = true
    utils.CustomerPortalDB.Save(&user)

    c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// ResetPassword handles password reset requests by generating and sending an OTP
func ResetPassword(c *gin.Context) {
    var input struct {
        CustomerNumber string `json:"customer_number"`
        DeliveryMethod string `json:"delivery_method"` // "email" or "phone"
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var user models.User
    // Check if the customer exists in the database
    if err := utils.CustomerPortalDB.Where("customer_number = ?", input.CustomerNumber).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid customer number"})
        return
    }

    // Generate a new OTP
    otp := generateOTP()
    user.OTP = otp
    user.OTPGeneratedAt = time.Now()

    // Send the OTP based on the selected delivery method
    if input.DeliveryMethod == "email" && user.Email != "" {
        sendOTPEmail(user.Email, otp)
    } else if input.DeliveryMethod == "phone" && user.PhoneNumber != "" {
        sendOTPWhatsApp(user.PhoneNumber, otp)
    } else {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or missing delivery method"})
        return
    }

    // Save the user with the new OTP data
    utils.CustomerPortalDB.Save(&user)

    // Inform the user that the OTP has been sent
    c.JSON(http.StatusOK, gin.H{"message": "OTP sent"})
}

