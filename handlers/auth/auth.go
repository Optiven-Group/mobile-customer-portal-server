package auth

import (
	"log"
	"math/rand"
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

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

var jwtSecret []byte

func init() {
    // Load the .env file
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }

    jwtSecret = []byte(os.Getenv("JWT_SECRET"))

    if len(jwtSecret) == 0 {
        log.Fatal("JWT secret is not set or empty")
    }
}

func SavePushToken(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
			return
	}
	user := userInterface.(models.User)

	var input struct {
			PushToken string `json:"pushToken"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data."})
			return
	}

	// Update the user's push token in the database
	if err := utils.CustomerPortalDB.Model(&user).Update("push_token", input.PushToken).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save push token"})
			return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Push token saved successfully"})
}


