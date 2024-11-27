package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
        // It's okay if the .env file isn't found; environment variables may be set elsewhere
        log.Println("No .env file found or error loading .env file:", err)
    }

    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        log.Fatal("JWT_SECRET is not set in the environment")
    }

    JwtSecret = []byte(secret)
}

// GenerateAccessToken creates a new JWT access token
func GenerateAccessToken(userID uint) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": userID,
        "exp":     time.Now().Add(15 * time.Minute).Unix(), // Access token valid for 15 minutes
    })

    return token.SignedString(JwtSecret)
}

// GenerateRefreshToken creates a new random refresh token
func GenerateRefreshToken() (string, error) {
    bytes := make([]byte, 32) // 256 bits
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

// HashToken hashes a token using SHA256
func HashToken(token string) string {
    hash := sha256.Sum256([]byte(token))
    return hex.EncodeToString(hash[:])
}
