package models

import (
    "time"

    "gorm.io/gorm"
)

type User struct {
    gorm.Model
    CustomerNumber string    `gorm:"unique;not null"`
    Email          string    `gorm:"unique;not null"`
    PhoneNumber    string
    Password       string    `gorm:"not null"`
    OTP            string    `gorm:"column:otp"`
    OTPGeneratedAt time.Time `gorm:"column:otp_generated_at"`
    Verified       bool      `gorm:"default:false"`
    UserType       string    `gorm:"not null"`
    GroupID        *uint
    Group          Group
    InitialSetup   bool      `gorm:"default:false"`
}
