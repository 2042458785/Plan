package dao

import (
	"Boutique/src/api"
	m "Boutique/src/model/http_model"
	"Boutique/src/utils"
	"context"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type AuthDao struct{ Orm *gorm.DB }

func (d *AuthDao) FindUser(ctx context.Context, name string) (m.AdminUser, error) {
	var user m.AdminUser
	e := d.Orm.WithContext(ctx).Where("username = ?", name).First(&user).Error
	return user, e
}
func (d *AuthDao) SaveSession(ctx context.Context, s m.AdminSession) error {
	return d.Orm.WithContext(ctx).Create(&s).Error
}
func (d *AuthDao) ValidSession(ctx context.Context, id string, userID uint) (bool, error) {
	var count int64
	e := d.Orm.WithContext(ctx).Table("sys_admin_session AS sessions").Joins("JOIN sys_admin_user AS users ON users.id = sessions.admin_id").Where("sessions.id = ? AND sessions.admin_id = ? AND sessions.revoked = ? AND sessions.expires_at > ? AND users.enabled = ? AND users.role = ?", id, userID, false, time.Now(), true, "admin").Count(&count).Error
	return count == 1, e
}
func (d *AuthDao) Revoke(ctx context.Context, id string) error {
	return d.Orm.WithContext(ctx).Model(&m.AdminSession{}).Where("id = ?", id).Update("revoked", true).Error
}

// Authenticate 在 DAO 事务内锁定登录失败计数，防止多进程绕过限流。
func (d *AuthDao) Authenticate(ctx context.Context, in m.LoginInput, ip string) (m.AdminUser, error) {
	var user m.AdminUser
	var denied, blocked bool
	e := d.Orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		key := utils.Fingerprint(ip)
		if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&m.LoginAttempt{ID: key}).Error; e != nil {
			return e
		}
		var a m.LoginAttempt
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, "id = ?", key).Error; e != nil {
			return e
		}
		if a.BlockedUntil != nil && a.BlockedUntil.After(time.Now()) {
			blocked = true
			return nil
		}
		if a.BlockedUntil != nil {
			a.Count = 0
			a.BlockedUntil = nil
		}
		e := tx.Where("username = ? AND enabled = ? AND role = ?", in.Username, true, "admin").First(&user).Error
		if e != nil && e != gorm.ErrRecordNotFound {
			return e
		}
		hash := user.PasswordHash
		if hash == "" {
			hash = "$2a$10$7EqJtq98hPqEX7fNZaFWoO5WxhUqjKQ1mJ0Y1aGfFFh14Hk6Cvhva"
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil || user.ID == 0 {
			denied = true
			a.Count++
			if a.Count >= 5 {
				until := time.Now().Add(time.Minute)
				a.BlockedUntil = &until
			}
			return tx.Save(&a).Error
		}
		a.Count = 0
		a.BlockedUntil = nil
		return tx.Save(&a).Error
	})
	if e != nil {
		return user, e
	}
	if blocked {
		return user, api.Problem(429, "登录尝试过多，请一分钟后重试")
	}
	if denied {
		return user, api.Problem(401, "账号或密码不正确")
	}
	return user, nil
}
