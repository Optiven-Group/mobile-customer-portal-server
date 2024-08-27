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

    r.POST("/login", handlers.Login)
    r.POST("/reset-password", handlers.ResetPassword)
    r.POST("/verify-otp", handlers.VerifyOTP)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    r.Run(":" + port) // listen and serve on 0.0.0.0:8080
}
