package auth

import (
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
            c.Abort()
            return
        }

        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
            c.Abort()
            return
        }

        tokenString := parts[1]

        // Parse the token
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return utils.JwtSecret, nil
        })

        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        if !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // Get the user ID and iat from the token claims
        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
            c.Abort()
            return
        }

        userIDFloat, ok := claims["user_id"].(float64)
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
            c.Abort()
            return
        }

        iatFloat, iatExists := claims["iat"].(float64)
        if !iatExists {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Token missing issued-at (iat) field"})
            c.Abort()
            return
        }

        userID := uint(userIDFloat)
        iatTime := time.Unix(int64(iatFloat), 0)

        // Fetch the user from the database
        var user models.User
        if err := utils.CustomerPortalDB.First(&user, userID).Error; err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
            c.Abort()
            return
        }

        // Check if the token was issued before the user's last logout
        if user.LastLogoutAt != nil && user.LastLogoutAt.After(iatTime) {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalidated by logout"})
            c.Abort()
            return
        }

        // Set the user in the context
        c.Set("user", user)

        c.Next()
    }
}
