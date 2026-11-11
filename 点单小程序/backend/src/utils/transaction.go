package utils

import (
	"context"
	"gorm.io/gorm"
)

func WithTransaction(ctx context.Context, db *gorm.DB, fn func(*gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}
