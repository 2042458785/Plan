package http_model

import "time"

type Category struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"size:60;uniqueIndex" json:"name"`
	Position int    `json:"position"`
}
type Product struct {
	ID          string       `gorm:"primaryKey;size:48" json:"id"`
	CategoryID  uint         `gorm:"not null;index" json:"categoryId"`
	Category    Category     `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Name        string       `gorm:"size:100;not null" json:"name"`
	English     string       `gorm:"size:200" json:"english"`
	Description string       `gorm:"size:500" json:"description"`
	Price       int64        `gorm:"not null" json:"price"`
	Image       string       `gorm:"size:1000" json:"image"`
	Available   bool         `gorm:"not null" json:"available"`
	Material    string       `gorm:"size:300" json:"material"`
	Details     string       `gorm:"type:text" json:"details"`
	Version     uint64       `gorm:"not null;default:1" json:"version"`
	Position    int          `json:"-"`
	SKUs        []ProductSKU `gorm:"foreignKey:ProductID" json:"skus"`
	CreatedAt   time.Time    `json:"-"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}
type ProductSKU struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProductID string    `gorm:"size:48;not null;uniqueIndex:idx_sku_option" json:"productId"`
	Variant   string    `gorm:"size:100;not null;uniqueIndex:idx_sku_option" json:"variant"`
	Color     string    `gorm:"size:100;not null;uniqueIndex:idx_sku_option" json:"color"`
	Stock     int       `gorm:"not null" json:"stock"`
	Active    bool      `gorm:"not null" json:"active"`
	Version   uint64    `gorm:"not null;default:1" json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type ShopSetting struct {
	ID           uint      `gorm:"primaryKey" json:"-"`
	Name         string    `gorm:"size:100" json:"name"`
	Announcement string    `gorm:"size:1000" json:"announcement"`
	Location     string    `gorm:"size:300" json:"location"`
	Hours        string    `gorm:"size:100" json:"hours"`
	Open         bool      `gorm:"not null" json:"open"`
	Version      uint64    `gorm:"not null;default:1" json:"version"`
	UpdatedAt    time.Time `json:"-"`
}

const DeliveryFeeCents int64 = 200

type ProductImage struct {
	ID          string `gorm:"primaryKey;size:48"`
	ContentType string `gorm:"size:40;not null"`
	Data        []byte `gorm:"type:longblob;not null"`
}

type Order struct {
	ID          string      `gorm:"primaryKey;size:48" json:"id"`
	Number      string      `gorm:"size:40;uniqueIndex;not null" json:"number"`
	LookupToken string      `gorm:"size:64;not null" json:"-"`
	RequestKey  string      `gorm:"size:100;uniqueIndex;not null" json:"-"`
	Fingerprint string      `gorm:"size:64;not null" json:"-"`
	Items       []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
	Total       int64       `gorm:"not null" json:"total"`
	Note        string      `gorm:"size:600" json:"note"`
	Customer    string      `gorm:"size:100" json:"customer"`
	Contact     string      `gorm:"size:100" json:"contact"`
	Pickup      string      `gorm:"size:100" json:"pickup"`
	Fulfillment string      `gorm:"size:12;not null;default:pickup" json:"fulfillment"`
	Address     string      `gorm:"size:300;not null;default:''" json:"address"`
	DeliveryFee int64       `gorm:"not null;default:0" json:"deliveryFee"`
	Status      string      `gorm:"size:20;index;not null" json:"status"`
	Paid        bool        `gorm:"not null" json:"paid"`
	Version     uint64      `gorm:"not null;default:1" json:"version"`
	CreatedAt   time.Time   `gorm:"index" json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}
type OrderItem struct {
	ID        uint   `gorm:"primaryKey" json:"-"`
	OrderID   string `gorm:"size:48;not null;index" json:"-"`
	ProductID string `gorm:"size:48;not null;index" json:"productId"`
	SKUID     uint   `gorm:"not null;index" json:"skuId"`
	Name      string `gorm:"size:100" json:"name"`
	Price     int64  `gorm:"not null" json:"price"`
	Quantity  int    `gorm:"not null" json:"quantity"`
	Variant   string `gorm:"size:100" json:"variant"`
	Color     string `gorm:"size:100" json:"color"`
	Image     string `gorm:"size:1000" json:"image"`
}
type StockMovement struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	SKUID       uint      `gorm:"not null;index" json:"skuId"`
	ProductID   string    `gorm:"size:48;not null;index" json:"productId"`
	OrderID     string    `gorm:"size:48;index" json:"orderId"`
	Delta       int       `gorm:"not null" json:"delta"`
	Reason      string    `gorm:"size:40" json:"reason"`
	BusinessKey string    `gorm:"size:150;uniqueIndex;not null" json:"-"`
	CreatedAt   time.Time `json:"createdAt"`
}
type AdminUser struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"size:60;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:100;not null"`
	Role         string `gorm:"size:20;not null"`
	Enabled      bool   `gorm:"not null"`
	CreatedAt    time.Time
}
type AdminSession struct {
	ID        string    `gorm:"primaryKey;size:48"`
	AdminID   uint      `gorm:"index;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	Revoked   bool      `gorm:"not null"`
	CreatedAt time.Time
}
type AdminAuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	AdminID    uint      `gorm:"index" json:"adminId"`
	Action     string    `gorm:"size:60" json:"action"`
	ResourceID string    `gorm:"size:100" json:"resourceId"`
	CreatedAt  time.Time `json:"createdAt"`
}
type MigrationRecord struct {
	Name      string `gorm:"primaryKey;size:100"`
	AppliedAt time.Time
}
type LoginAttempt struct {
	ID           string `gorm:"primaryKey;size:64"`
	Count        int
	BlockedUntil *time.Time
	UpdatedAt    time.Time
}
