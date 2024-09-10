package main

import (
	"log"
	"os"

	"mobile-customer-portal-server/handlers"
	"mobile-customer-portal-server/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	r := gin.Default()
	utils.ConnectDatabase()

	// Define the routes
	r.POST("/verify-user", handlers.VerifyUser)      // Verifies user and sends OTP
	r.POST("/request-otp", handlers.RequestOTP)      // Requests an OTP for password reset
	r.POST("/reset-password", handlers.ResetPassword) // Verifies OTP and sets new password
	r.POST("/verify-otp", handlers.VerifyOTP)        // Verifies OTP for user creation

	// Set the port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Run the server
	r.Run(":" + port) // listen and serve on 0.0.0.0:8080
}
