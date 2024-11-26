package notifications

import "github.com/gin-gonic/gin"

func RegisterNotificationsRoutes(r *gin.RouterGroup) {
	r.POST("/send-notification", SendNotification)
	r.GET("/notifications", GetNotifications)
}
