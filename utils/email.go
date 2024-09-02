package utils

import (
	"log"
	"os"

	"gopkg.in/gomail.v2"
)

// SendOTPEmail sends the OTP to the user's email address
func SendOTPEmail(email string, otp string) {
	// Create a new email message
	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("SMTP_SENDER")) // Sender email address from environment
	m.SetHeader("To", email)                      // Recipient email address
	m.SetHeader("Subject", "Your OTP Code")       // Email subject
	m.SetBody("text/plain", "Your OTP code is: " + otp) // Email body with the OTP code

	// Dialer configuration for the SMTP server
	d := gomail.NewDialer(
		os.Getenv("SMTP_HOST"),
		465,
		os.Getenv("SMTP_USER"), // SMTP username, usually your email address
		os.Getenv("SMTP_PASS"),
	)

	// Sending the email
	if err := d.DialAndSend(m); err != nil {
		log.Printf("Failed to send OTP email to %s: %v", email, err)
		return
	}

	log.Printf("OTP email successfully sent to %s", email) // Log success
}
