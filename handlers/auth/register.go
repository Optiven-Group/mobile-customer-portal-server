package auth

import (
	"log"
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// VerifyUser checks if the customer exists and sends an OTP for verification
func VerifyUser(c *gin.Context) {
    var input struct {
        CustomerNumber string `json:"customer_number"`
        Email          string `json:"email"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please provide a valid customer number and email."})
        return
    }

    if input.CustomerNumber == "" || input.Email == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Customer number and email are required."})
        return
    }

    var customer models.Customer
    // Find the customer by customer number and email in the CRM database
    if err := utils.CRMDB.Where("customer_no = ? AND primary_email = ?", input.CustomerNumber, input.Email).First(&customer).Error; err != nil {
        log.Printf("Customer not found: %v", err)
        c.JSON(http.StatusUnauthorized, gin.H{"error": "No matching customer found. Please verify your details or contact support."})
        return
    }

    // Generate a new OTP and set generation time
    otp := generateOTP()
    customer.OTP = otp
    now := time.Now()
    customer.OTPGeneratedAt = &now

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

// VerifyOTP validates the OTP during registration
func VerifyOTP(c *gin.Context) {
	var input struct {
			CustomerNumber string `json:"customer_number"`
			Email          string `json:"email"`
			OTP            string `json:"otp"`
	}

	// Bind JSON input to the struct
	if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid input data. Please ensure all required fields are filled correctly.",
			})
			return
	}

	// Validate input fields
	if input.CustomerNumber == "" || input.Email == "" || input.OTP == "" {
			c.JSON(http.StatusBadRequest, gin.H{
					"error": "Customer number, email, and OTP are required.",
			})
			return
	}

	var customer models.Customer
	// Find the customer by customer number and email in the CRM database
	if err := utils.CRMDB.Where("customer_no = ? AND primary_email = ?", input.CustomerNumber, input.Email).
			First(&customer).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Customer not found. Please verify your customer number and email.",
			})
			return
	}

	// Ensure OTP and OTPGeneratedAt are correctly populated
	if customer.OTP == "" || customer.OTPGeneratedAt == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
					"error": "The OTP is missing or not properly set. Please request a new OTP.",
			})
			return
	}

	// Verify OTP
	if input.OTP != customer.OTP {
			c.JSON(http.StatusUnauthorized, gin.H{
					"error": "The OTP is incorrect. Please try again or request a new one.",
			})
			return
	}

	// Check OTP expiration
	if time.Now().After(customer.OTPGeneratedAt.Add(otpValidityDuration)) {
			c.JSON(http.StatusUnauthorized, gin.H{
					"error": "The OTP has expired. Please request a new OTP.",
			})
			return
	}

	// OTP is valid, proceed
	c.JSON(http.StatusOK, gin.H{
			"message": "OTP verified successfully.",
	})
}

// CompleteRegistration finalizes the registration process after OTP verification
func CompleteRegistration(c *gin.Context) {
	var input struct {
			CustomerNumber string `json:"customer_number"`
			Email          string `json:"email"`
			OTP            string `json:"otp"`
			NewPassword    string `json:"new_password"`
	}

	// Bind JSON input to the struct
	if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid input data. Please ensure all required fields are filled correctly.",
			})
			return
	}

	// Validate input fields
	if input.CustomerNumber == "" || input.Email == "" || input.OTP == "" || input.NewPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{
					"error": "Customer number, email, OTP, and new password are required.",
			})
			return
	}

	var customer models.Customer
	// Find the customer by customer number and email in the CRM database
	if err := utils.CRMDB.Where("customer_no = ? AND primary_email = ?", input.CustomerNumber, input.Email).
			First(&customer).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Customer not found. Please verify your customer number and email.",
			})
			return
	}

	// Ensure OTP and OTPGeneratedAt are correctly populated
	if customer.OTP == "" || customer.OTPGeneratedAt == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
					"error": "The OTP is missing or not properly set. Please request a new OTP.",
			})
			return
	}

	// Verify OTP
	if input.OTP != customer.OTP {
			c.JSON(http.StatusUnauthorized, gin.H{
					"error": "The OTP is incorrect. Please try again or request a new one.",
			})
			return
	}

	// Check OTP expiration
	if time.Now().After(customer.OTPGeneratedAt.Add(otpValidityDuration)) {
			c.JSON(http.StatusUnauthorized, gin.H{
					"error": "The OTP has expired. Please request a new OTP.",
			})
			return
	}

	// Check if user already exists in the customer-portal database
	var existingUser models.User
	if err := utils.CustomerPortalDB.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{
					"error": "User already exists. Please log in or use the forgot password option.",
			})
			return
	}

	// Create a new user in the customer-portal database
	user := models.User{
			CustomerNumber: input.CustomerNumber,
			Email:          input.Email,
			PhoneNumber:    customer.Phone,
			Verified:       true,
			UserType:       "individual",
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
					"error": "An error occurred while processing your password. Please try again.",
			})
			return
	}
	user.Password = string(hashedPassword)

	// Save the new user to the customer-portal database
	if err := utils.CustomerPortalDB.Create(&user).Error; err != nil {
			log.Printf("Failed to create user in the database: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
					"error": "We encountered an issue creating your account. Please contact support.",
			})
			return
	}

	// Clear OTP and OTPGeneratedAt in CRM database
	customer.OTP = ""
	customer.OTPGeneratedAt = nil
	if err := utils.CRMDB.Save(&customer).Error; err != nil {
			log.Printf("Failed to clear OTP in the CRM database: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
					"error": "We encountered an issue processing your request. Please try again later.",
			})
			return
	}

	c.JSON(http.StatusOK, gin.H{
			"message": "User registered successfully. You can now log in.",
	})
}

