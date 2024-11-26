package migrations

import (
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
)

func MigrateNotifications() {
	utils.CustomerPortalDB.AutoMigrate(&models.Notification{})
}