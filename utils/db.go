package utils

import (
	"fmt"
	"log"
	"mobile-customer-portal-server/models"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Global variables to hold the database connections
var DefaultDB *gorm.DB
var CustomerPortalDB *gorm.DB

// ConnectDatabase establishes connections to the default and customer portal databases
func ConnectDatabase() {
    // Construct the DSN (Data Source Name) for the default database using environment variables
    defaultDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("LOGISTICS_DB"),
    )

    // Construct the DSN (Data Source Name) for the customer portal database using environment variables
    customerPortalDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("CUSTOMER_PORTAL_DB"),
    )

    var err error

    // Open a connection to the default database
    DefaultDB, err = gorm.Open(mysql.Open(defaultDSN), &gorm.Config{})
    if err != nil {
        log.Fatalf("Failed to connect to default database: %v", err) // Log and exit if the connection fails
    }

    // Open a connection to the customer portal database
    CustomerPortalDB, err = gorm.Open(mysql.Open(customerPortalDSN), &gorm.Config{})
    if err != nil {
        log.Fatalf("Failed to connect to customer portal database: %v", err) // Log and exit if the connection fails
    }

    // Automatically migrate the schema for the User and Group models to the customer portal database
    CustomerPortalDB.AutoMigrate(&models.User{}, &models.Group{})
}
