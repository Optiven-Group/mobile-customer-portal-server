package models

type Referral struct {
    ID              uint    `gorm:"primaryKey" json:"id"`
    ReferrerID      string  `gorm:"column:referrer_id" json:"referrer_id"`
    ReferredName    string  `gorm:"column:referred_name" json:"referred_name"`
    ReferredEmail   string  `gorm:"column:referred_email" json:"referred_email"`
    PropertyID      string  `gorm:"column:property_id" json:"property_id"`
    Status          string  `gorm:"column:status" json:"status"` // e.g., "Pending", "Completed"
    AmountPaid      float64 `gorm:"column:amount_paid" json:"amount_paid"`
}
