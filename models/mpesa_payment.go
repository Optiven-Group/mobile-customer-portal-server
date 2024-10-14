package models

import "gorm.io/gorm"

type MpesaPayment struct {
    gorm.Model
    CheckoutRequestID     string `gorm:"unique;not null"`
    InstallmentScheduleID string `gorm:"not null"`
    CustomerNumber        string `gorm:"not null"`
    PhoneNumber           string `gorm:"not null"`
    Amount                string `gorm:"not null"`
    Status                string `gorm:"not null"` // e.g., "Pending", "Success", "Failed"
}
