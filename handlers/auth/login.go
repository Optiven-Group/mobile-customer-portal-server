package auth

import (
	"log"
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
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

    // Return the token and user data in the response
    c.JSON(http.StatusOK, gin.H{
        "message": "Login successful.",
        "token":   tokenString,
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
    // Since JWT tokens are stateless, you can't invalidate them server-side without additional mechanisms.
    // One common approach is to handle token blacklisting.
    // For simplicity, we'll just return a success message.

    c.JSON(http.StatusOK, gin.H{
        "message": "Logout successful.",
    })
}
