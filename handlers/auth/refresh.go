package auth

import (
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

func RefreshToken(c *gin.Context) {
    var input struct {
        RefreshToken string `json:"refresh_token"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data. Please provide a refresh token."})
        return
    }

    // Parse the refresh token
    token, err := jwt.Parse(input.RefreshToken, func(token *jwt.Token) (interface{}, error) {
        return utils.JwtSecret, nil
    })

    if err != nil || !token.Valid {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
        return
    }

    // Get the user ID from the token claims
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
        return
    }

    userIDFloat, ok := claims["user_id"].(float64)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
        return
    }

    userID := uint(userIDFloat)

    // Fetch the user from the database
    var user models.User
    if err := utils.CustomerPortalDB.First(&user, userID).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
        return
    }

    // Verify the refresh token matches the one in the database
    err = bcrypt.CompareHashAndPassword([]byte(user.RefreshToken), []byte(input.RefreshToken))
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
        return
    }

    // Generate new access and refresh tokens
    newAccessToken, err := utils.GenerateAccessToken(user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate access token"})
        return
    }

    newRefreshToken, err := utils.GenerateRefreshToken(user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate refresh token"})
        return
    }

    // Save the new refresh token hash
    hashedRefreshToken, err := bcrypt.GenerateFromPassword([]byte(newRefreshToken), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash refresh token"})
        return
    }

    user.RefreshToken = string(hashedRefreshToken)
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
