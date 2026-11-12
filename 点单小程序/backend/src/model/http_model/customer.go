package http_model

import "time"

type CustomerAccount struct {
	ID           string    `gorm:"primaryKey;size:20" json:"number"`
	Identifier   string    `gorm:"size:64;uniqueIndex;not null" json:"contact"`
	Name         string    `gorm:"size:100;not null" json:"name"`
	Dorm         string    `gorm:"size:300;not null" json:"dorm"`
	PasswordHash string    `gorm:"size:100;not null" json:"-"`
	CreatedAt    time.Time `json:"-"`
}

type CustomerSession struct {
	TokenHash  string    `gorm:"primaryKey;size:64"`
	CustomerID string    `gorm:"size:20;not null;index"`
	ExpiresAt  time.Time `gorm:"not null;index"`
}

type CustomerCredentials struct {
	Contact  string `json:"contact"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type CustomerProfileInput struct {
	Name string `json:"name"`
	Dorm string `json:"dorm"`
}
