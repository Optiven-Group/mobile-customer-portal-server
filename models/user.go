package models

import (
    "gorm.io/gorm"
)

type User struct {
    gorm.Model
    CustomerNumber string    `gorm:"unique;not null"`
    Email          string    `gorm:"unique;not null"`
    PhoneNumber    string
    Password       string    `gorm:"not null"`
    Verified       bool      `gorm:"default:false"`
    UserType       string    `gorm:"not null"`
}
