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

// RequestOTP handles password reset requests by generating and sending a new OTP via email
func RequestOTP(c *gin.Context) {
    var input struct {
        Email string `json:"email"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid input data. Please provide a valid email address.",
        })
        return
    }

    if input.Email == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Email address is required.",
        })
        return
    }

    var user models.User
    // Check if the user exists in the customer-portal database by email
    if err := utils.CustomerPortalDB.Where("email = ?", input.Email).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "User not found. Please check your email address.",
        })
        return
    }

    // Generate a new OTP
    otp := generateOTP()
    now := time.Now()

    // Create a new PasswordReset record
    passwordReset := models.PasswordReset{
        UserID:         user.ID,
        OTP:            otp,
        OTPGeneratedAt: now,
    }

    // Save the PasswordReset record to the database
    if err := utils.CustomerPortalDB.Create(&passwordReset).Error; err != nil {
        log.Printf("Failed to create password reset record: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "We encountered an issue saving the OTP. Please try again later.",
        })
        return
    }

    // Send the OTP via email
    sendOTP(user.Email, otp)

    c.JSON(http.StatusOK, gin.H{
        "message": "OTP sent to your registered email address.",
    })
}

// VerifyOTPReset validates the OTP during password reset
func VerifyOTPReset(c *gin.Context) {
    var input struct {
        Email string `json:"email"`
        OTP   string `json:"otp"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid input data. Please ensure all required fields are filled correctly.",
        })
        return
    }

    if input.Email == "" || input.OTP == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Email and OTP are required.",
        })
        return
    }

    var user models.User
    // Check if the user exists in the customer-portal database by email
    if err := utils.CustomerPortalDB.Where("email = ?", input.Email).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "User not found. Please check your email address.",
        })
        return
    }

    var passwordReset models.PasswordReset
    // Find the latest PasswordReset record for the user
    if err := utils.CustomerPortalDB.Where("user_id = ?", user.ID).Order("created_at DESC").First(&passwordReset).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "OTP not found or expired. Please request a new OTP.",
        })
        return
    }

    // Check OTP validity
    if input.OTP != passwordReset.OTP {
        log.Printf("OTP mismatch: received %s, expected %s", input.OTP, passwordReset.OTP)
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "The OTP is incorrect. Please try again or request a new one.",
        })
        return
    }

    if time.Since(passwordReset.OTPGeneratedAt) > otpValidityDuration {
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

// ResetPassword handles the password reset after verifying the OTP
func ResetPassword(c *gin.Context) {
    var input struct {
        Email       string `json:"email"`
        OTP         string `json:"otp"`
        NewPassword string `json:"new_password"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid input data. Please ensure all required fields are filled correctly.",
        })
        return
    }

    if input.Email == "" || input.OTP == "" || input.NewPassword == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Email, OTP, and new password are required.",
        })
        return
    }

    var user models.User
    // Check if the user exists in the customer-portal database by email
    if err := utils.CustomerPortalDB.Where("email = ?", input.Email).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "User not found. Please check your email address.",
        })
        return
    }

    var passwordReset models.PasswordReset
    // Find the latest PasswordReset record for the user
    if err := utils.CustomerPortalDB.Where("user_id = ?", user.ID).Order("created_at DESC").First(&passwordReset).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "OTP not found or expired. Please request a new OTP.",
        })
        return
    }

    // Check OTP validity
    if input.OTP != passwordReset.OTP {
        log.Printf("OTP mismatch: received %s, expected %s", input.OTP, passwordReset.OTP)
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "The OTP is incorrect. Please try again or request a new one.",
        })
        return
    }

    if time.Since(passwordReset.OTPGeneratedAt) > otpValidityDuration {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "The OTP has expired. Please request a new OTP.",
        })
        return
    }

    // Hash the new password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "An error occurred while processing your password. Please try again.",
        })
        return
    }

    // Update the user's password in the database
    user.Password = string(hashedPassword)

    // Save the updated user
    if err := utils.CustomerPortalDB.Save(&user).Error; err != nil {
        log.Printf("Failed to update user password in the database: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "We encountered an issue updating your password. Please try again later.",
        })
        return
    }

    // Delete the used PasswordReset record
    if err := utils.CustomerPortalDB.Delete(&passwordReset).Error; err != nil {
        log.Printf("Failed to delete password reset record: %v", err)
        // Proceed anyway; this is not critical
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Your password has been reset successfully. You can now log in with your new password.",
    })
}