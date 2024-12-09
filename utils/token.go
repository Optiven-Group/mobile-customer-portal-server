package utils

import (
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"
)

var JwtSecret []byte

func init() {
    // Load the .env file
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found or error loading .env file:", err)
    }

    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        log.Fatal("JWT_SECRET is not set in the environment")
    }

    JwtSecret = []byte(secret)
}

// GenerateAccessToken creates a new JWT access token without expiration.
func GenerateAccessToken(userID uint) (string, error) {
    claims := jwt.MapClaims{
        "user_id": userID,
        // No 'exp' claim, making it non-expiring by default.
        "iat": time.Now().Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(JwtSecret)
}
