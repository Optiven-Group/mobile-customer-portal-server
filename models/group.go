package models

import (
	"gorm.io/gorm"
)

type Group struct {
    gorm.Model
    Name    string `gorm:"not null"`
    Email   string
    Members []User
}
