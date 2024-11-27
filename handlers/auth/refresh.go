package auth

import (
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RefreshToken(c *gin.Context) {
    var input struct {
        RefreshToken string `json:"refresh_token"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please provide a refresh token."})
        return
    }

    // Extract user ID from the expired access token in the Authorization header
    authHeader := c.GetHeader("Authorization")
    if authHeader == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
        return
    }

    userID, err := utils.ExtractUserIDFromToken(authHeader)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access token"})
        return
    }

    // Fetch the user from the database
    var user models.User
    if err := utils.CustomerPortalDB.First(&user, userID).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
        return
    }

    // Hash the provided refresh token
    hashedInputToken := utils.HashToken(input.RefreshToken)

    // Verify the refresh token matches the one in the database
    if hashedInputToken != user.RefreshToken {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
        return
    }

    // Generate new access and refresh tokens
    newAccessToken, err := utils.GenerateAccessToken(user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate access token"})
        return
    }

    newRefreshToken, err := utils.GenerateRefreshToken()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate refresh token"})
        return
    }

    // Hash and save the new refresh token
    user.RefreshToken = utils.HashToken(newRefreshToken)
    if err := utils.CustomerPortalDB.Save(&user).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save refresh token"})
        return
    }

    // Return the new tokens
    c.JSON(http.StatusOK, gin.H{
        "access_token":  newAccessToken,
        "refresh_token": newRefreshToken,
    })
}
