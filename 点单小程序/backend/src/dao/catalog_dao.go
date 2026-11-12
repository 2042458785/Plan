package dao

import (
	"Boutique/src/api"
	m "Boutique/src/model/http_model"
	"Boutique/src/utils"
	"context"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"slices"
	"sort"
	"time"
)

type CatalogDao struct{ Orm *gorm.DB }

func (d *CatalogDao) Catalog(ctx context.Context, includeUnavailable bool) ([]m.ProductView, []m.Category, m.ShopSetting, error) {
	var products []m.Product
	var setting m.ShopSetting
	categories := []m.Category{}
	e := utils.WithTransaction(ctx, d.Orm, func(tx *gorm.DB) error {
		if err := tx.Order("position ASC, id ASC").Find(&categories).Error; err != nil {
			return err
		}
		query := tx.Preload("Category").Preload("SKUs", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).Order("position ASC, id ASC")
		if !includeUnavailable {
			query = query.Where("available = ?", true)
		}
		if err := query.Find(&products).Error; err != nil {
			return err
		}
		return tx.First(&setting, 1).Error
	})
	if e != nil {
		return nil, nil, setting, e
	}
	out := make([]m.ProductView, 0, len(products))
	for _, p := range products {
		out = append(out, m.ViewProduct(p))
	}
	return out, categories, setting, nil
}
func (d *CatalogDao) SaveProduct(ctx context.Context, id string, input m.ProductInput, adminID uint) (m.ProductView, error) {
	var result m.ProductView
	e := utils.WithTransaction(ctx, d.Orm, func(tx *gorm.DB) error {
		var category m.Category
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"})
		if input.CategoryID != 0 {
			query = query.Where("id = ?", input.CategoryID)
		} else {
			query = query.Where("name = ?", input.Category)
		}
		if e := query.First(&category).Error; e != nil {
			if e == gorm.ErrRecordNotFound {
				return api.Problem(409, "请先创建或重新选择商品分类")
			}
			return e
		}
		var product m.Product
		creating := id == "new"
		if creating {
			product.ID = utils.RandomID()
			product.Version = 1
		} else {
			if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("SKUs").First(&product, "id = ?", id).Error; e != nil {
				if e == gorm.ErrRecordNotFound {
					return api.Problem(404, "商品不存在")
				}
				return e
			}
			if product.Version != input.Version {
				return api.Problem(409, "商品或库存已被更新，请刷新后再编辑")
			}
			product.Version++
		}
		product.CategoryID = category.ID
		product.Name = input.Name
		product.Description = input.Description
		product.English = input.English
		product.Price = input.Price
		if err := updateProductImages(&product, input); err != nil {
			return err
		}
		product.Available = input.Available
		product.Material = input.Material
		product.Details = input.Details
		if input.Care != nil {
			product.Care = *input.Care
		}
		if creating {
			if e := tx.Omit("Category", "SKUs").Create(&product).Error; e != nil {
				return e
			}
		} else {
			if e := tx.Omit("Category", "SKUs").Save(&product).Error; e != nil {
				return e
			}
		}
		// 规格记录保留 ID，已有订单仍可引用历史规格。不存在的旧规格只下架，不物理删除。
		old := map[string]m.ProductSKU{}
		for _, sku := range product.SKUs {
			old[sku.Variant+"\x00"+sku.Color] = sku
		}
		seen := map[string]bool{}
		inputs := append([]m.SKUInput{}, input.SKUs...)
		sort.Slice(inputs, func(i, j int) bool { return inputs[i].Variant+inputs[i].Color < inputs[j].Variant+inputs[j].Color })
		for _, in := range inputs {
			key := in.Variant + "\x00" + in.Color
			seen[key] = true
			sku, exists := old[key]
			before := sku.Stock
			if !exists {
				sku = m.ProductSKU{ProductID: product.ID, Variant: in.Variant, Color: in.Color, Version: 1}
			} else {
				if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&sku, sku.ID).Error; e != nil {
					return e
				}
				before = sku.Stock
				sku.Version++
			}
			sku.Stock = in.Stock
			sku.Active = true
			if e := tx.Save(&sku).Error; e != nil {
				return e
			}
			if delta := in.Stock - before; delta != 0 {
				if e := tx.Create(&m.StockMovement{SKUID: sku.ID, ProductID: product.ID, Delta: delta, Reason: "manual_adjustment", BusinessKey: fmt.Sprintf("edit:%s:%d:%d", product.ID, product.Version, sku.ID)}).Error; e != nil {
					return e
				}
			}
		}
		for key, sku := range old {
			if !seen[key] {
				if e := tx.Model(&m.ProductSKU{}).Where("id = ?", sku.ID).Updates(map[string]any{"active": false, "version": gorm.Expr("version + 1")}).Error; e != nil {
					return e
				}
			}
		}
		if e := tx.Create(&m.AdminAuditLog{AdminID: adminID, Action: "save_product", ResourceID: product.ID}).Error; e != nil {
			return e
		}
		var refreshed m.Product
		if e := tx.Preload("Category").Preload("SKUs").First(&refreshed, "id = ?", product.ID).Error; e != nil {
			return e
		}
		result = m.ViewProduct(refreshed)
		return nil
	})
	return result, e
}
func updateProductImages(product *m.Product, input m.ProductInput) error {
	if input.Gallery != nil {
		product.Gallery = input.Gallery
		product.Image = ""
	} else if input.Image != product.Image {
		product.Image = input.Image
		if len(product.Gallery) > 0 {
			// 旧版只更新首图，保留其余图片。
			remaining := slices.DeleteFunc(product.Gallery[1:], func(image string) bool { return image == input.Image })
			product.Gallery = append([]string{input.Image}, remaining...)
		}
	}
	if input.ColorImages != nil {
		product.ColorImages = input.ColorImages
	}
	// 只保留当前规格使用的颜色，且用规格顺序决定颜色封面的回退值。
	activeImages := make(map[string][]string)
	for _, sku := range input.SKUs {
		images := product.ColorImages[sku.Color]
		if len(images) != 0 {
			activeImages[sku.Color] = images
			if product.Image == "" {
				product.Image = images[0]
			}
		}
	}
	product.ColorImages = activeImages
	if len(product.Gallery) != 0 {
		product.Image = product.Gallery[0]
	}
	if product.Image == "" {
		return api.Problem(400, "请在商品相册或任一颜色上传至少一张图片")
	}
	return nil
}

func (d *CatalogDao) SaveSettings(ctx context.Context, input m.ShopSetting, adminID uint) (m.ShopSetting, error) {
	e := utils.WithTransaction(ctx, d.Orm, func(tx *gorm.DB) error {
		var old m.ShopSetting
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&old, 1).Error; e != nil {
			return e
		}
		if input.Version != old.Version {
			return api.Problem(409, "店铺信息已变化，请刷新后再保存")
		}
		input.ID = 1
		input.Version++
		if e := tx.Save(&input).Error; e != nil {
			return e
		}
		return tx.Create(&m.AdminAuditLog{AdminID: adminID, Action: "save_settings", ResourceID: "1"}).Error
	})
	return input, e
}

func (d *CatalogDao) Dashboard(ctx context.Context) (map[string]any, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, 1)
	var revenue int64
	var today, pending, confirmed int64
	if e := d.Orm.WithContext(ctx).Model(&m.Order{}).Where("created_at >= ? AND created_at < ? AND paid = ? AND status <> ?", start, end, true, "cancelled").Select("COALESCE(SUM(total),0)").Scan(&revenue).Error; e != nil {
		return nil, e
	}
	if e := d.Orm.WithContext(ctx).Model(&m.Order{}).Where("created_at >= ? AND created_at < ?", start, end).Count(&today).Error; e != nil {
		return nil, e
	}
	if e := d.Orm.WithContext(ctx).Model(&m.Order{}).Where("status = ?", "pending").Count(&pending).Error; e != nil {
		return nil, e
	}
	if e := d.Orm.WithContext(ctx).Model(&m.Order{}).Where("status = ?", "confirmed").Count(&confirmed).Error; e != nil {
		return nil, e
	}
	type sale struct {
		ProductID string
		Sold      int64
	}
	var sales []sale
	if e := d.Orm.WithContext(ctx).Table("sys_order_item AS i").Select("i.product_id, SUM(i.quantity) AS sold").Joins("JOIN sys_order AS o ON o.id = i.order_id").Where("o.created_at >= ? AND o.created_at < ? AND o.status <> ?", start, end, "cancelled").Group("i.product_id").Scan(&sales).Error; e != nil {
		return nil, e
	}
	counts := map[string]int64{}
	for _, v := range sales {
		counts[v.ProductID] = v.Sold
	}
	products, _, _, e := d.Catalog(ctx, true)
	if e != nil {
		return nil, e
	}
	ranking := make([]map[string]any, 0, len(products))
	for _, p := range products {
		ranking = append(ranking, map[string]any{"id": p.ID, "name": p.Name, "image": p.Image, "sold": counts[p.ID]})
	}
	sort.SliceStable(ranking, func(i, j int) bool { return ranking[i]["sold"].(int64) > ranking[j]["sold"].(int64) })
	return map[string]any{"revenue": revenue, "todayOrders": today, "pending": pending, "confirmed": confirmed, "ranking": ranking}, nil
}

func (d *CatalogDao) CreateImage(ctx context.Context, image *m.ProductImage) error {
	return d.Orm.WithContext(ctx).Create(image).Error
}

func (d *CatalogDao) FindImage(ctx context.Context, id string) (m.ProductImage, error) {
	var image m.ProductImage
	err := d.Orm.WithContext(ctx).First(&image, "id = ?", id).Error
	return image, err
}
