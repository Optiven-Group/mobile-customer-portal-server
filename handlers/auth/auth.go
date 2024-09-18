package auth

import (
	"log"
	"math/rand"
	"mobile-customer-portal-server/utils"
	"os"
	"time"

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
