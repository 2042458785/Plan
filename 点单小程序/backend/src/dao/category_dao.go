package dao

import (
	"Boutique/src/api"
	m "Boutique/src/model/http_model"
	"Boutique/src/utils"
	"context"
	"errors"
	"strconv"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *CatalogDao) SaveCategory(ctx context.Context, id uint, name string, adminID uint) (m.Category, error) {
	var category m.Category
	err := utils.WithTransaction(ctx, d.Orm, func(tx *gorm.DB) error {
		if id == 0 {
			category.Name = name
			if err := tx.Create(&category).Error; err != nil {
				return categoryWriteError(err)
			}
		} else {
			if err := lockCategory(tx, id, &category); err != nil {
				return err
			}
			if err := tx.Model(&category).Update("name", name).Error; err != nil {
				return categoryWriteError(err)
			}
		}
		return tx.Create(&m.AdminAuditLog{AdminID: adminID, Action: "save_category", ResourceID: strconv.FormatUint(uint64(category.ID), 10)}).Error
	})
	return category, err
}

func (d *CatalogDao) DeleteCategory(ctx context.Context, id, adminID uint) error {
	return utils.WithTransaction(ctx, d.Orm, func(tx *gorm.DB) error {
		var category m.Category
		if err := lockCategory(tx, id, &category); err != nil {
			return err
		}
		var ids []string
		if err := tx.Model(&m.Product{}).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("category_id = ?", id).Order("id").Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) > 0 {
			// 订单明细和库存流水独立保存，删除目录时保留历史记录。
			if err := tx.Where("product_id IN ?", ids).Delete(&m.ProductSKU{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", ids).Delete(&m.Product{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Delete(&category).Error; err != nil {
			return err
		}
		return tx.Create(&m.AdminAuditLog{AdminID: adminID, Action: "delete_category", ResourceID: strconv.FormatUint(uint64(id), 10)}).Error
	})
}

func lockCategory(tx *gorm.DB, id uint, category *m.Category) error {
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(category, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return api.Problem(404, "分类不存在")
	}
	return err
}

func categoryWriteError(err error) error {
	var duplicate *mysql.MySQLError
	if errors.As(err, &duplicate) && duplicate.Number == 1062 {
		return api.Problem(409, "分类名称已存在")
	}
	return err
}
