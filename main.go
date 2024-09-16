package main

import (
	"log"
	"os"
	"time"

	"mobile-customer-portal-server/handlers"
	"mobile-customer-portal-server/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
    err := godotenv.Load()
    if err != nil {
        log.Fatalf("Error loading .env file")
    }

    r := gin.Default()

    // Setup CORS
    r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"*"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))

    utils.ConnectDatabase()

    // Define the routes
    r.POST("/login", handlers.Login)
    r.POST("/verify-user", handlers.VerifyUser)
    r.POST("/verify-otp", handlers.VerifyOTP)
    r.POST("/complete-registration", handlers.CompleteRegistration)
    r.POST("/request-otp", handlers.RequestOTP)
    r.POST("/verify-otp-reset", handlers.VerifyOTPReset)
    r.POST("/reset-password", handlers.ResetPassword)

    // Set the port
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    // Run the server
    r.Run(":" + port) // listen and serve on 0.0.0.0:8080
}
