package handlers

import (
	"log"
	"math/rand"
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
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
func sendOTP(email, otp string) {
    utils.SendOTPEmail(email, otp)
}

// Login handles user authentication
func Login(c *gin.Context) {
    var input struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please provide a valid email and password."})
        return
    }

    var user models.User
    if err := utils.CustomerPortalDB.Where("email = ?", input.Email).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password."})
        return
    }

    // Check password
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password."})
        return
    }

    // Retrieve the JWT secret from the environment variable and convert it to []byte
    jwtSecret := []byte(os.Getenv("JWT_SECRET"))
    if len(jwtSecret) == 0 {
        log.Println("JWT secret is not set or empty")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
        return
    }

    // Generate JWT token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": user.ID,
        "exp":     time.Now().Add(time.Hour * 72).Unix(), // Token expires in 72 hours
    })

    tokenString, err := token.SignedString(jwtSecret)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
        return
    }

    // Return the token in the response
    c.JSON(http.StatusOK, gin.H{
        "message": "Login successful.",
        "token":   tokenString,
    })
}

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
    customer.OTPGeneratedAt = time.Now().Format("2006-01-02 15:04:05.000") // Including milliseconds

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

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please ensure all required fields are filled correctly."})
        return
    }

    if input.CustomerNumber == "" || input.Email == "" || input.OTP == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Customer number, email, and OTP are required."})
        return
    }

    var customer models.Customer
    // Find the customer by customer number and email in the CRM database
    if err := utils.CRMDB.Where("customer_no = ? AND primary_email = ?", input.CustomerNumber, input.Email).First(&customer).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Customer not found. Please verify your customer number and email."})
        return
    }

    // Ensure OTP and OTPGeneratedAt are correctly populated
    if customer.OTP == "" || customer.OTPGeneratedAt == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP is missing or not properly set. Please request a new OTP."})
        return
    }

    // Parse the OTP generation time
    otpGeneratedAt, err := time.Parse("2006-01-02 15:04:05.000", customer.OTPGeneratedAt)
    if err != nil {
        log.Printf("Error parsing OTPGeneratedAt: %v", err)
        c.JSON(http.StatusUnauthorized, gin.H{"error": "There was an error verifying your OTP. Please request a new one."})
        return
    }

    // Verify OTP and check expiration
    if input.OTP != customer.OTP {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP is incorrect. Please try again or request a new one."})
        return
    }

    if time.Now().After(otpGeneratedAt.Add(otpValidityDuration)) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP has expired. Please request a new OTP."})
        return
    }

    // OTP is valid, proceed
    c.JSON(http.StatusOK, gin.H{"message": "OTP verified successfully."})
}

// CompleteRegistration finalizes the registration process after OTP verification
func CompleteRegistration(c *gin.Context) {
    var input struct {
        CustomerNumber string `json:"customer_number"`
        Email          string `json:"email"`
        OTP            string `json:"otp"`
        NewPassword    string `json:"new_password"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please ensure all required fields are filled correctly."})
        return
    }

    if input.CustomerNumber == "" || input.Email == "" || input.OTP == "" || input.NewPassword == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Customer number, email, OTP, and new password are required."})
        return
    }

    var customer models.Customer
    // Find the customer by customer number and email in the CRM database
    if err := utils.CRMDB.Where("customer_no = ? AND primary_email = ?", input.CustomerNumber, input.Email).First(&customer).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Customer not found. Please verify your customer number and email."})
        return
    }

    // Ensure OTP and OTPGeneratedAt are correctly populated
    if customer.OTP == "" || customer.OTPGeneratedAt == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP is missing or not properly set. Please request a new OTP."})
        return
    }

    // Parse the OTP generation time
    otpGeneratedAt, err := time.Parse("2006-01-02 15:04:05.000", customer.OTPGeneratedAt)
    if err != nil {
        log.Printf("Error parsing OTPGeneratedAt: %v", err)
        c.JSON(http.StatusUnauthorized, gin.H{"error": "There was an error verifying your OTP. Please request a new one."})
        return
    }

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

    // Clear OTP and OTPGeneratedAt in CRM database
    customer.OTP = ""
    customer.OTPGeneratedAt = ""
    utils.CRMDB.Save(&customer)

    c.JSON(http.StatusOK, gin.H{"message": "User registered successfully. You can now log in."})
}

// RequestOTP handles password reset requests by generating and sending a new OTP via email
func RequestOTP(c *gin.Context) {
    var input struct {
        Email string `json:"email"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please provide a valid email address."})
        return
    }

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
    now := time.Now()
    user.OTPGeneratedAt = &now

    // Save the user with the new OTP data
    if err := utils.CustomerPortalDB.Save(&user).Error; err != nil {
        log.Printf("Failed to update user with new OTP: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "We encountered an issue saving the OTP. Please try again later."})
        return
    }

    // Send the OTP via email
    sendOTP(user.Email, otp)

    c.JSON(http.StatusOK, gin.H{"message": "OTP sent to your registered email address."})
}

// VerifyOTPReset validates the OTP during password reset
func VerifyOTPReset(c *gin.Context) {
    var input struct {
        Email string `json:"email"`
        OTP   string `json:"otp"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please ensure all required fields are filled correctly."})
        return
    }

    if input.Email == "" || input.OTP == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Email and OTP are required."})
        return
    }

    var user models.User
    // Check if the user exists in the customer-portal database by email
    if err := utils.CustomerPortalDB.Where("email = ?", input.Email).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found. Please check your email address."})
        return
    }

    // Ensure OTP and OTPGeneratedAt are correctly populated
    if user.OTP == "" || user.OTPGeneratedAt == nil || user.OTPGeneratedAt.IsZero() {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP is missing or not properly set. Please request a new OTP."})
			return
	}

    // Check OTP validity
    if input.OTP != user.OTP {
        log.Printf("OTP mismatch: received %s, expected %s", input.OTP, user.OTP)
        c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP is incorrect. Please try again or request a new one."})
        return
    }

    if time.Since(*user.OTPGeneratedAt) > otpValidityDuration {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP has expired. Please request a new OTP."})
        return
    }

    // OTP is valid, proceed
    c.JSON(http.StatusOK, gin.H{"message": "OTP verified successfully."})
}

// ResetPassword handles the password reset after verifying the OTP
func ResetPassword(c *gin.Context) {
	var input struct {
			Email       string `json:"email"`
			OTP         string `json:"otp"`
			NewPassword string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please ensure all required fields are filled correctly."})
			return
	}

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

	// Ensure OTP and OTPGeneratedAt are correctly populated
	if user.OTP == "" || user.OTPGeneratedAt == nil || user.OTPGeneratedAt.IsZero() {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP is missing or not properly set. Please request a new OTP."})
			return
	}

	// Check OTP validity
	if input.OTP != user.OTP {
			log.Printf("OTP mismatch: received %s, expected %s", input.OTP, user.OTP)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "The OTP is incorrect. Please try again or request a new one."})
			return
	}

	if time.Since(*user.OTPGeneratedAt) > otpValidityDuration {
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
	user.OTP = ""          // Clear the OTP
	user.OTPGeneratedAt = nil // Clear OTP generation time

	// Save the updated user
	if err := utils.CustomerPortalDB.Save(&user).Error; err != nil {
			log.Printf("Failed to update user password in the database: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "We encountered an issue updating your password. Please try again later."})
			return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Your password has been reset successfully. You can now log in with your new password."})
}
