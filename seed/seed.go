// seed/seed.go
package seed

import (
	"errors"
	"log"
	"time"

	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"

	"gorm.io/gorm"
)

func SeedCampaign() error {
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	var existingCampaign models.Campaign
	err := utils.CustomerPortalDB.Where("month = ? AND year = ? AND featured = ?", month, year, true).First(&existingCampaign).Error
	if err == nil {
		log.Println("Featured monthly campaign already exists. Skipping seeding.")
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	campaign := models.Campaign{
		Title:          "Summer Savings",
		Description:    "Enjoy exclusive discounts on select properties this summer!",
		BannerImageURL: "https://images.unsplash.com/photo-1719937206168-f4c829152b91?q=80&w=2070&auto=format&fit=crop&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA==",
		Month:          month,
		Year:           year,
		Featured:       true,
	}

	if err := utils.CustomerPortalDB.Create(&campaign).Error; err != nil {
		return err
	}

	log.Println("Featured monthly campaign seeded successfully.")
	return nil
}
