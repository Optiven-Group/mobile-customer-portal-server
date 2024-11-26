package migrations

import (
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
)

func MigrateCampaigns() {
	utils.CustomerPortalDB.AutoMigrate(&models.Campaign{})
}