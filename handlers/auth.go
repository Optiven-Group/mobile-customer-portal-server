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

// sendOTP sends the OTP via email
func sendOTP(contactInfo, otp string) {
	utils.SendOTPEmail(contactInfo, otp)
}

// VerifyUser checks if the customer exists in the CRM database and sends an OTP for verification
func VerifyUser(c *gin.Context) {
	var input struct {
		CustomerNumber string `json:"customer_number"`
		EmailOrPhone   string `json:"email_or_phone"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please provide a valid customer number and contact information."})
		return
	}

	var customer models.Customer
	// Find the customer by customer number, email or phone, and ensure customer_type is "Individual"
	if err := utils.CRMDB.Where("customer_no = ? AND (primary_email = ? OR phone = ?) AND customer_type = ?", input.CustomerNumber, input.EmailOrPhone, input.EmailOrPhone, "Individual").First(&customer).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No matching individual customer found. Please verify your details or contact support if you are not an individual customer."})
		return
	}

	// Check if an OTP already exists and is still valid
	otpGeneratedAt, err := time.Parse("2006-01-02 15:04:05", customer.OTPGeneratedAt)
	if err == nil && time.Since(otpGeneratedAt) < otpValidityDuration {
		// If OTP is still valid, do not generate a new one
		c.JSON(http.StatusOK, gin.H{"message": "An OTP was already sent recently. Please check your email."})
		return
	}

	// Generate a new OTP and set generation time
	otp := generateOTP()
	customer.OTP = otp
	customer.OTPGeneratedAt = time.Now().Format("2006-01-02 15:04:05")

	// Save OTP and generation time in the CRM database
	if err := utils.CRMDB.Save(&customer).Error; err != nil {
		log.Printf("Failed to save OTP in the CRM database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "We encountered an issue processing your request. Please try again later."})
		return
	}

	// Send OTP via email
	sendOTP(customer.PrimaryEmail, otp)

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent successfully to your email."})
}

// VerifyOTP validates the OTP and either registers the user or informs them if they are already registered
func VerifyOTP(c *gin.Context) {
	var input struct {
		CustomerNumber string `json:"customer_number"`
		OTP            string `json:"otp"`
		NewPassword    string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please ensure all required fields are filled correctly."})
		return
	}

	var customer models.Customer
	// Find the customer by customer number in the CRM database
	if err := utils.CRMDB.Where("customer_no = ?", input.CustomerNumber).First(&customer).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Customer not found. Please verify your customer number."})
		return
	}

	// Logging for debugging
	log.Printf("Retrieved customer with OTP: %s and generated at: %s", customer.OTP, customer.OTPGeneratedAt)

	// Ensure OTP and OTPGeneratedAt are correctly populated
	if customer.OTP == "" || customer.OTPGeneratedAt == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP is missing or not properly set. Please request a new OTP."})
		return
	}

	// Parse the OTP generation time with the correct format
	otpGeneratedAt, err := time.Parse(time.RFC3339, customer.OTPGeneratedAt)
	if err != nil {
		log.Printf("Error parsing OTPGeneratedAt: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "There was an error verifying your OTP. Please request a new one."})
		return
	}

	// Log parsed OTP generation time
	log.Printf("Parsed OTP generated at: %s", otpGeneratedAt)

	// Verify OTP and check expiration
	if input.OTP != customer.OTP {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP is incorrect. Please try again or request a new one."})
		return
	}

	if time.Now().After(otpGeneratedAt.Add(otpValidityDuration)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP has expired. Please request a new OTP."})
		return
	}

	// Check if user already exists in the customer-portal database
	var existingUser models.User
	if err := utils.CustomerPortalDB.Where("customer_number = ?", input.CustomerNumber).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "User already exists. If you've forgotten your password, please use the password reset option.",
			"reset_password": "If you need assistance, please contact support or use the reset password option.",
		})
		return
	}

	// Create a new user in the customer-portal database
	user := models.User{
		CustomerNumber: input.CustomerNumber,
		Email:          customer.PrimaryEmail,
		PhoneNumber:    customer.Phone,
		Verified:       true,
		UserType:       "individual", // Placeholder for future group or company types
		InitialSetup:   true,
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred while processing your password. Please try again."})
		return
	}
	user.Password = string(hashedPassword)

	// Save the new user to the customer-portal database
	if err := utils.CustomerPortalDB.Create(&user).Error; err != nil {
		log.Printf("Failed to create user in the database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "We encountered an issue creating your account. Please contact support."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User created successfully. You can now log in."})
}

// RequestOTP handles password reset requests by generating and sending a new OTP via email
func RequestOTP(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
	}

	// Validate the input data
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please provide a valid email address."})
		return
	}

	// Check if the email is provided
	if input.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email address is required."})
		return
	}

	var user models.User
	// Check if the user exists in the customer-portal database by email
	if err := utils.CustomerPortalDB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found. Please check your email address."})
		return
	}

	// Generate a new OTP
	otp := generateOTP()
	user.OTP = otp
	user.OTPGeneratedAt = time.Now()

	// Log the OTP and generation time before saving
	log.Printf("Saving OTP: %s and generated at: %s for user: %s", user.OTP, user.OTPGeneratedAt, user.Email)

	// Save the user with the new OTP data
	if err := utils.CustomerPortalDB.Save(&user).Error; err != nil {
		log.Printf("Failed to update user with new OTP: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "We encountered an issue saving the OTP. Please try again later."})
		return
	}

	// Send the OTP via email
	sendOTP(user.Email, otp)

	// Inform the user that the OTP has been sent
	c.JSON(http.StatusOK, gin.H{"message": "OTP sent to your registered email address."})
}

// ResetPassword handles the password reset after verifying the OTP
func ResetPassword(c *gin.Context) {
	var input struct {
		Email       string `json:"email"`
		OTP         string `json:"otp"`
		NewPassword string `json:"new_password"`
	}

	// Validate the input data
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please ensure all required fields are filled correctly."})
		return
	}

	// Check if all required fields are provided
	if input.Email == "" || input.OTP == "" || input.NewPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email, OTP, and new password are required."})
		return
	}

	var user models.User
	// Check if the user exists in the customer-portal database by email
	if err := utils.CustomerPortalDB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found. Please check your email address."})
		return
	}

	// Log retrieved OTP and its generated timestamp
	log.Printf("Retrieved user with OTP: %s and generated at: %v", user.OTP, user.OTPGeneratedAt)

	// Ensure OTP and OTPGeneratedAt are correctly populated
	if user.OTP == "" || user.OTPGeneratedAt.IsZero() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP is missing or not properly set. Please request a new OTP."})
		return
	}

	// Parse the OTP generation time using the correct format
	otpGeneratedAt, err := time.Parse(time.RFC3339, user.OTPGeneratedAt.Format(time.RFC3339))
	if err != nil {
		log.Printf("Error parsing OTPGeneratedAt: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "There was an error verifying your OTP. Please request a new one."})
		return
	}

	// Log parsed OTP generation time for debugging
	log.Printf("Parsed OTP generated at: %v", otpGeneratedAt)

	// Verify OTP and check expiration
	if input.OTP != user.OTP {
		log.Printf("OTP mismatch: received %s, expected %s", input.OTP, user.OTP)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP is incorrect. Please try again or request a new one."})
		return
	}

	if time.Now().After(otpGeneratedAt.Add(otpValidityDuration)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP has expired. Please request a new OTP."})
		return
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred while processing your password. Please try again."})
		return
	}

	// Update the user's password in the database
	user.Password = string(hashedPassword)
	user.OTP = ""                       // Clear the OTP
	user.OTPGeneratedAt = time.Time{}   // Clear OTP generation time

	// Save the updated user data to the database
	if err := utils.CustomerPortalDB.Save(&user).Error; err != nil {
		log.Printf("Failed to update user password in the database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "We encountered an issue updating your password. Please try again later."})
		return
	}

	// Inform the user that the password has been reset successfully
	c.JSON(http.StatusOK, gin.H{"message": "Your password has been reset successfully. You can now log in with your new password."})
}

// Placeholder function for future development handling group or company user sign-ins
func HandleGroupUser() {
	// Logic to handle group or company users
}
