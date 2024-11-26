package notifications

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"

	"github.com/gin-gonic/gin"
)

type PushMessage struct {
	To        string                 `json:"to"`
	Sound     string                 `json:"sound"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Data      map[string]interface{} `json:"data,omitempty"`
	ChannelID string                 `json:"channelId,omitempty"`
}

func SendNotification(c *gin.Context) {
	var req struct {
		UserID uint                   `json:"user_id"`
		Title  string                 `json:"title"`
		Body   string                 `json:"body"`
		Data   map[string]interface{} `json:"data,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	var user models.User
	if err := utils.CustomerPortalDB.Where("id = ?", req.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.PushToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User does not have a push token"})
		return
	}

	pushMessage := PushMessage{
		To:    user.PushToken,
		Sound: "default",
		Title: req.Title,
		Body:  req.Body,
		Data:  req.Data,
	}

	body, err := json.Marshal(pushMessage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal push message"})
		return
	}

	resp, err := http.Post("https://exp.host/--/api/v2/push/send", "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send push notification"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Push notification service returned an error"})
		return
	}

	// Optionally, save the notification to the database
	notification := models.Notification{
		UserID: user.ID, // Now uint, matches the updated model
		Title:  req.Title,
		Body:   req.Body,
		Data:   "",
	}

	if req.Data != nil {
		dataBytes, err := json.Marshal(req.Data)
		if err == nil {
			notification.Data = string(dataBytes)
		}
	}

	if err := utils.CustomerPortalDB.Create(&notification).Error; err != nil {
		log.Printf("Failed to save notification: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"status": "Notification sent"})
}

func GetNotifications(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	var notifications []models.Notification
	if err := utils.CustomerPortalDB.Where("user_id = ?", user.ID).Order("created_at desc").Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
	})
}
