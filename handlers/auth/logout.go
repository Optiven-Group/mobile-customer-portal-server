package auth

import (
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

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
