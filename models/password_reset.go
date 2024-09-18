package models

import "time"

type PasswordReset struct {
    ID             uint      `gorm:"primaryKey"`
    CreatedAt      time.Time
    UpdatedAt      time.Time
    UserID         uint      `gorm:"index"`
    OTP            string
    OTPGeneratedAt time.Time
}
