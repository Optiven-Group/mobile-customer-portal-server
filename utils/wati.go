package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// WatiMessage represents the structure of a message to send via Wati API
type WatiMessage struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// SendOTPWhatsApp sends the OTP to the user's phone number via WhatsApp using Wati API
func SendOTPWhatsApp(phoneNumber string, otp string) {
	// Create the message payload
	message := WatiMessage{
		Phone:   phoneNumber,
		Message: fmt.Sprintf("Your OTP code is: %s", otp),
	}

	// Convert the message struct to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal OTP message: %v", err)
		return
	}

	// Create the Wati API request
	req, err := http.NewRequest("POST", os.Getenv("WATI_URL")+"/api/v1/sendSessionMessage", bytes.NewBuffer(messageJSON))
	if err != nil {
		log.Printf("Failed to create Wati API request: %v", err)
		return
	}

	// Set the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("WATI_API_KEY"))

	// Send the request to Wati API
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send OTP via WhatsApp: %v", err)
		return
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to send OTP via WhatsApp: received status code %d", resp.StatusCode)
	}
}
