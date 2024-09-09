package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
    gorm.Model
    CustomerNumber string `gorm:"unique;not null"`
    Email          string
    PhoneNumber    string
    Password       string
    OTP            string `gorm:"-"`
    NewPassword    string `gorm:"-"`
    Verified       bool   `gorm:"default:false"`
    UserType       string `gorm:"not null"`
    GroupID        *uint
    Group          Group
    InitialSetup   bool      `gorm:"default:false"`
    OTPExpiresAt   time.Time `gorm:"-"`
    OTPGeneratedAt time.Time `gorm:"-"`
}
