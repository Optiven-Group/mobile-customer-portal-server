package campaigns

import (
	"net/http"
	"time"

	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"

	"github.com/gin-gonic/gin"
)

func GetMonthlyCampaign(c *gin.Context) {
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	var campaign models.Campaign
	if err := utils.CustomerPortalDB.Where("month = ? AND year = ? AND featured = ?", month, year, true).First(&campaign).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No featured campaign found for this month"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"campaign": campaign,
	})
}
