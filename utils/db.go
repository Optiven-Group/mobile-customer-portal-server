package utils

import (
	"fmt"
	"log"
	"mobile-customer-portal-server/models"
	"os"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DefaultDB *gorm.DB
var CustomerPortalDB *gorm.DB

func ConnectDatabase() {
    defaultDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("LOGISTICS_DB"),
    )

    customerPortalDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("CUSTOMER_PORTAL_DB"),
    )

    var err error

    DefaultDB, err = gorm.Open(mysql.Open(defaultDSN), &gorm.Config{})
    if err != nil {
        log.Fatalf("Failed to connect to default database: %v", err)
    }

    CustomerPortalDB, err = gorm.Open(mysql.Open(customerPortalDSN), &gorm.Config{})
    if err != nil {
        log.Fatalf("Failed to connect to customer portal database: %v", err)
    }

    CustomerPortalDB.AutoMigrate(&models.User{}, &models.Group{})
}
