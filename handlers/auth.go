package handlers

import (
	"log"
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
	const digits = "0123456789"
	otp := make([]byte, 6)
	for i := range otp {
		otp[i] = digits[r.Intn(len(digits))]
	}
	return string(otp)
}

func sendOTPEmail(email string, otp string) {
    utils.SendOTPEmail(email, otp)
}

func sendOTPWhatsApp(phoneNumber string, otp string) {
    utils.SendOTPWhatsApp(phoneNumber, otp)
}

// sendOTP sends the OTP via email or WhatsApp based on the delivery method
func sendOTP(deliveryMethod, contactInfo, otp string) {
	if deliveryMethod == "email" {
		utils.SendOTPEmail(contactInfo, otp)
	} else {
		utils.SendOTPWhatsApp(contactInfo, otp)
	}
}

// VerifyUser checks if the customer exists in the CRM database and sends an OTP for verification
func VerifyUser(c *gin.Context) {
	var input struct {
		CustomerNumber string `json:"customer_number"`
		EmailOrPhone   string `json:"email_or_phone"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	var customer models.Customer
	// Find the customer by customer number and either email or phone in the CRM database
	if err := utils.CRMDB.Where("customer_no = ? AND (primary_email = ? OR phone = ?)", input.CustomerNumber, input.EmailOrPhone, input.EmailOrPhone).First(&customer).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid customer number or contact information"})
		return
	}

	// Generate OTP and set generation time
	otp := generateOTP()
	customer.OTP = otp
	customer.OTPGeneratedAt = time.Now().Format("2006-01-02 15:04:05")

	// Save OTP and generation time in the CRM database
	if err := utils.CRMDB.Save(&customer).Error; err != nil {
		log.Printf("Failed to save OTP in the CRM database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save OTP"})
		return
	}

	// Send OTP via email or WhatsApp
	if input.EmailOrPhone == customer.PrimaryEmail {
		sendOTP("email", customer.PrimaryEmail, otp)
	} else if input.EmailOrPhone == customer.Phone {
		sendOTP("phone", customer.Phone, otp)
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent", "otp_sent": true})
}

// VerifyOTP validates the OTP and creates a user in the customer-portal database if valid
func VerifyOTP(c *gin.Context) {
	var input struct {
		CustomerNumber string `json:"customer_number"`
		OTP            string `json:"otp"`
		NewPassword    string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	var customer models.Customer
	// Find the customer by customer number in the CRM database
	if err := utils.CRMDB.Where("customer_no = ?", input.CustomerNumber).First(&customer).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Customer not found in CRM database"})
		return
	}

	// Parse the OTP generation time correctly using ISO 8601 format
	otpGeneratedAt, err := time.Parse(time.RFC3339, customer.OTPGeneratedAt)
	if err != nil {
		log.Printf("Error parsing OTPGeneratedAt: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to parse OTP generation time"})
		return
	}

	// Verify OTP and check expiration
	if input.OTP != customer.OTP {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP"})
		return
	}

	if time.Now().After(otpGeneratedAt.Add(otpValidityDuration)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Expired OTP"})
		return
	}

	// Create a new user in the customer-portal database
	user := models.User{
		CustomerNumber: input.CustomerNumber,
		Email:          customer.PrimaryEmail,
		PhoneNumber:    customer.Phone,
		Verified:       true,
		UserType:       "individual",
		InitialSetup:   true,
		GroupID:        nil,
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	user.Password = string(hashedPassword)

	// Save the new user to the customer-portal database
	if err := utils.CustomerPortalDB.Create(&user).Error; err != nil {
		log.Printf("Failed to create user in the database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user in the database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User created successfully"})
}

// ResetPassword handles password reset requests by generating and sending a new OTP
func ResetPassword(c *gin.Context) {
	var input struct {
		CustomerNumber string `json:"customer_number"`
		DeliveryMethod string `json:"delivery_method"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	var user models.User
	// Check if the user exists in the customer-portal database
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
	if err := utils.CustomerPortalDB.Save(&user).Error; err != nil {
		log.Printf("Failed to update user with new OTP: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update OTP information"})
		return
	}

	// Inform the user that the OTP has been sent
	c.JSON(http.StatusOK, gin.H{"message": "OTP sent"})
}
