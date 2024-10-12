package main

import (
	"log"
	"os"
	"time"

	"mobile-customer-portal-server/handlers/auth"
	"mobile-customer-portal-server/handlers/properties"
	"mobile-customer-portal-server/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func init() {
    // Load the .env file
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }
}

func main() {

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

    // Public routes
    r.POST("/login", auth.Login)
    r.POST("/logout", auth.Logout)

    r.POST("/verify-user", auth.VerifyUser)
    r.POST("/verify-otp", auth.VerifyOTP)
    r.POST("/complete-registration", auth.CompleteRegistration)
    r.POST("/request-otp", auth.RequestOTP)
    r.POST("/verify-otp-reset", auth.VerifyOTPReset)
    r.POST("/reset-password", auth.ResetPassword)

    // Protected routes
    protected := r.Group("/")
    protected.Use(auth.AuthMiddleware())
    {
        protected.GET("/properties", properties.GetProperties)
        protected.GET("/properties/:lead_file_no/installment-schedule", properties.GetInstallmentSchedule)
        protected.GET("/properties/:lead_file_no/installment-schedule/pdf", properties.GetInstallmentSchedulePDF)
        protected.GET("/properties/:lead_file_no/transactions", properties.GetTransactions)
        protected.GET("/projects", properties.GetUserProjects)
        protected.GET("/projects/:project_id/properties", properties.GetUserPropertiesByProject)
        protected.GET("/properties/:lead_file_no/receipts", properties.GetReceiptsByProperty)
    }

    // Set the port
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    // Run the server
    r.Run(":" + port) // listen and serve on 0.0.0.0:8080
}
