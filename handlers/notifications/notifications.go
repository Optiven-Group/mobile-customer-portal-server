package notifications

import (
	"bytes"
	"encoding/json"
	"net/http"

	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"

	"github.com/gin-gonic/gin"
)

type PushMessage struct {
	To       string `json:"to"`
	Sound    string `json:"sound"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Data     map[string]interface{} `json:"data,omitempty"`
	ChannelID string `json:"channelId,omitempty"`
}

func SendNotification(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
		Title  string `json:"title"`
		Body   string `json:"body"`
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

	c.JSON(http.StatusOK, gin.H{"status": "Notification sent"})
}
