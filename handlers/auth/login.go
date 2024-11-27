package auth

import (
	"log"
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

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

    // Fetch CustomerName from CRM database using CustomerNumber
    var customer models.Customer
    if err := utils.CRMDB.Where("customer_no = ?", user.CustomerNumber).First(&customer).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve customer information."})
        return
    }

    // Fetch lead files associated with the customer
    var leadFiles []models.LeadFile
    if err := utils.CRMDB.Where("customer_id = ?", user.CustomerNumber).Find(&leadFiles).Error; err != nil {
        log.Printf("Error fetching lead files: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve lead files."})
        return
    }
    

    // Generate tokens
    accessToken, err := utils.GenerateAccessToken(user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate access token"})
        return
    }

    refreshToken, err := utils.GenerateRefreshToken(user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate refresh token"})
        return
    }

    // Save refresh token hash in the database
    hashedRefreshToken, err := bcrypt.GenerateFromPassword([]byte(refreshToken), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash refresh token"})
        return
    }

    user.RefreshToken = string(hashedRefreshToken)
    if err := utils.CustomerPortalDB.Save(&user).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save refresh token"})
        return
    }

    // Return the token and user data in the response
    c.JSON(http.StatusOK, gin.H{
        "message": "Login successful.",
        "access_token":  accessToken,
        "refresh_token": refreshToken,
        "user": gin.H{
            "id":             user.ID,
            "email":          user.Email,
            "name":           customer.CustomerName,
            "customerNumber": user.CustomerNumber,
            "leadFiles":      leadFiles,
        },
    })
}


// Logout handles user sign-out
func Logout(c *gin.Context) {
    userInterface, exists := c.Get("user")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
        return
    }
    user := userInterface.(models.User)

    // Remove the refresh token from the database
    user.RefreshToken = ""
    if err := utils.CustomerPortalDB.Save(&user).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log out"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Logout successful.",
    })
}
