package handlers

import (
	"math/rand"
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// OTP validity duration
const otpValidityDuration = 10 * time.Minute

// generateOTP generates a 6-digit OTP
func generateOTP() string {
    source := rand.NewSource(time.Now().UnixNano())
    r := rand.New(source)
    const letters = "0123456789"
    otp := make([]byte, 6)
    for i := range otp {
        otp[i] = letters[r.Intn(len(letters))]
    }
    return string(otp)
}

func sendOTPEmail(email string, otp string) {
    utils.SendOTPEmail(email, otp)
}

func sendOTPWhatsApp(phoneNumber string, otp string) {
    utils.SendOTPWhatsApp(phoneNumber, otp)
}

// Verify user and send OTP
func VerifyUser(c *gin.Context) {
    var input struct {
        CustomerNumber string `json:"customer_number"`
        EmailOrPhone   string `json:"email_or_phone"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var customer models.Customer
    // Use CRMDB to find the customer by customer number and either email or phone
    if err := utils.CRMDB.Where("customer_no = ? AND (primary_email = ? OR phone = ?)", input.CustomerNumber, input.EmailOrPhone, input.EmailOrPhone).First(&customer).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid customer number or contact information"})
        return
    }

    otp := generateOTP()
    customer.OTP = otp
    customer.OTPGeneratedAt = time.Now().Format("2006-01-02 15:04:05") // Convert time.Time to string

    if input.EmailOrPhone == customer.PrimaryEmail {
        sendOTPEmail(customer.PrimaryEmail, otp)
    } else if input.EmailOrPhone == customer.Phone {
        sendOTPWhatsApp(customer.Phone, otp)
    }

    c.JSON(http.StatusOK, gin.H{"message": "OTP sent", "otp_sent": true})
}

// VerifyOTP handles OTP verification and completes the password reset process
func VerifyOTP(c *gin.Context) {
    var input struct {
        CustomerNumber string `json:"customer_number"`
        OTP            string `json:"otp"`
        NewPassword    string `json:"new_password"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var user models.User
    // Check if the user exists in the users table
    err := utils.CustomerPortalDB.Where("customer_number = ?", input.CustomerNumber).First(&user).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            // Fetch email and phone from the customer table in CRM
            var customer models.Customer
            if err := utils.CRMDB.Where("customer_no = ?", input.CustomerNumber).First(&customer).Error; err != nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Customer not found in CRM database"})
                return
            }

            // If user does not exist, create a new user record
            user = models.User{
                CustomerNumber: input.CustomerNumber,
                Email:          customer.PrimaryEmail,
                PhoneNumber:    customer.Phone,
                Verified:       false,
                UserType:       "individual", // Adjust based on your business logic
                InitialSetup:   false,
            }

            // Set GroupID to NULL or find an existing group
            user.GroupID = 0 // Set appropriately if needed; otherwise, allow it to be NULL
            if user.UserType == "group" {
                // Handle group association if the user type is 'group'
                var group models.Group
                if err := utils.CustomerPortalDB.First(&group).Error; err == nil {
                    user.GroupID = group.ID
                } else {
                    c.JSON(http.StatusBadRequest, gin.H{"error": "No valid group found for group user"})
                    return
                }
            }

            // Create the new user record
            if err := utils.CustomerPortalDB.Create(&user).Error; err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user in the database"})
                return
            }
        } else {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid customer number"})
            return
        }
    }

    // Check OTP validity
    if time.Now().After(user.OTPGeneratedAt.Add(otpValidityDuration)) || user.OTP != input.OTP {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
        return
    }

    // Hash and set the new password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
        return
    }

    // Update the user details
    user.Password = string(hashedPassword)
    user.Verified = true
    user.InitialSetup = true

    // Save the updated user in the users table
    if err := utils.CustomerPortalDB.Save(&user).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user in the database"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// ResetPassword handles password reset requests by generating and sending an OTP
func ResetPassword(c *gin.Context) {
    var input struct {
        CustomerNumber string `json:"customer_number"`
        DeliveryMethod string `json:"delivery_method"` // "email" or "phone"
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var user models.User
    // Check if the customer exists in the database
    if err := utils.CustomerPortalDB.Where("customer_number = ?", input.CustomerNumber).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid customer number"})
        return
    }

    // Generate a new OTP
    otp := generateOTP()
    user.OTP = otp
    user.OTPGeneratedAt = time.Now()

    // Send the OTP based on the selected delivery method
    if input.DeliveryMethod == "email" && user.Email != "" {
        sendOTPEmail(user.Email, otp)
    } else if input.DeliveryMethod == "phone" && user.PhoneNumber != "" {
        sendOTPWhatsApp(user.PhoneNumber, otp)
    } else {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or missing delivery method"})
        return
    }

    // Save the user with the new OTP data
    utils.CustomerPortalDB.Save(&user)

    // Inform the user that the OTP has been sent
    c.JSON(http.StatusOK, gin.H{"message": "OTP sent"})
}

