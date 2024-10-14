package referrals

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
)

func SubmitReferral(c *gin.Context) {
	var referral models.Referral
	if err := c.ShouldBindJSON(&referral); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
	}

	// Get the user from the context
	userInterface, exists := c.Get("user")
	if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
			return
	}
	user := userInterface.(models.User)

	referral.ReferrerID = user.CustomerNumber
	referral.Status = "Pending"
	referral.AmountPaid = 0

	if err := utils.CustomerPortalDB.Create(&referral).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit referral"})
			return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Referral submitted successfully"})
}

func GetUserReferrals(c *gin.Context) {
	// Get the user from the context
	userInterface, exists := c.Get("user")
	if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
			return
	}
	user := userInterface.(models.User)

	var referrals []models.Referral
	if err := utils.CustomerPortalDB.Where("referrer_id = ?", user.CustomerNumber).Find(&referrals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch referrals"})
			return
	}

	c.JSON(http.StatusOK, gin.H{"referrals": referrals})
}

func RedeemReferralReward(c *gin.Context) {
	referralID := c.Param("id")
	var referral models.Referral

	if err := utils.CustomerPortalDB.First(&referral, referralID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Referral not found"})
			return
	}

	// Update the referral status or handle the redemption logic eg, mark as redeemed
	referral.Status = "Redeemed"
	if err := utils.CustomerPortalDB.Save(&referral).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to redeem reward"})
			return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reward redeemed successfully"})
}