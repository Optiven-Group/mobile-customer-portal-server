package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
    gorm.Model
    CustomerNumber string     `gorm:"unique;not null" json:"customer_number"`
    Email          string     `gorm:"unique;not null" json:"email"`
    PhoneNumber    string     `json:"phone_number"`
    Password       string     `gorm:"not null" json:"password"`
    Verified       bool       `gorm:"default:false" json:"verified"`
    UserType       string     `gorm:"not null" json:"user_type"`
    PushToken      string     `gorm:"column:push_token" json:"push_token"`
    LastLogoutAt   *time.Time `gorm:"column:last_logout_at" json:"-"`
}
