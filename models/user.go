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
    UserType       string `gorm:"not null"` // "individual" or "group"
    GroupID        uint
    Group          Group
    InitialSetup   bool      `gorm:"default:false"`
    OTPExpiresAt   time.Time `gorm:"-"`  // Track OTP expiry time, not stored in DB
    OTPGeneratedAt time.Time `gorm:"-"`  // Track when OTP was generated, not stored in DB
}
