package models

import "time"

type Campaign struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	BannerImageURL string    `json:"banner_image_url"`
	Month          int       `json:"month"`
	Year           int       `json:"year"`
	Featured       bool      `json:"featured"`
	CreatedAt      time.Time `json:"created_at"`
	Link           string    `json:"link"`
}
