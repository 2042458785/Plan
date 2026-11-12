package dao

import (
	"Boutique/src/api"
	m "Boutique/src/model/http_model"
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type CustomerDao struct{ Orm *gorm.DB }

func (d *CustomerDao) Register(ctx context.Context, account m.CustomerAccount, session m.CustomerSession) error {
	return d.Orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&account).Error; err != nil {
			var duplicate *mysql.MySQLError
			if errors.As(err, &duplicate) && duplicate.Number == 1062 {
				return api.Problem(409, "账号已存在，请直接登录")
			}
			return err
		}
		return tx.Create(&session).Error
	})
}

func (d *CustomerDao) ByContact(ctx context.Context, contact string) (m.CustomerAccount, error) {
	var account m.CustomerAccount
	err := d.Orm.WithContext(ctx).Where("identifier = ?", contact).First(&account).Error
	return account, err
}

func (d *CustomerDao) CreateSession(ctx context.Context, session m.CustomerSession) error {
	return d.Orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("customer_id = ? AND expires_at <= ?", session.CustomerID, time.Now()).Delete(&m.CustomerSession{}).Error; err != nil {
			return err
		}
		return tx.Create(&session).Error
	})
}

func (d *CustomerDao) SessionAccount(ctx context.Context, hash string) (m.CustomerAccount, error) {
	var account m.CustomerAccount
	err := d.Orm.WithContext(ctx).Where("id = (?)", d.Orm.Model(&m.CustomerSession{}).
		Select("customer_id").Where("token_hash = ? AND expires_at > ?", hash, time.Now())).First(&account).Error
	return account, err
}

func (d *CustomerDao) DeleteSession(ctx context.Context, hash string) error {
	return d.Orm.WithContext(ctx).Delete(&m.CustomerSession{}, "token_hash = ?", hash).Error
}

func (d *CustomerDao) UpdateProfile(ctx context.Context, account m.CustomerAccount, input m.CustomerProfileInput) (m.CustomerAccount, error) {
	account.Name, account.Dorm = input.Name, input.Dorm
	err := d.Orm.WithContext(ctx).Model(&account).Select("name", "dorm").Updates(account).Error
	return account, err
}
