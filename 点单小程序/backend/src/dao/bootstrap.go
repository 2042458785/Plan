package dao

import (
	"Boutique/src/api"
	m "Boutique/src/model/http_model"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

//go:embed catalog.seed.json
var catalogSeed []byte

type LegacyProduct struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	English     string   `json:"english"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Price       int64    `json:"price"`
	Image       string   `json:"image"`
	Available   bool     `json:"available"`
	Stock       int      `json:"stock"`
	Material    string   `json:"material"`
	Details     string   `json:"details"`
	Care        string   `json:"care"`
	Options     []string `json:"options"`
	Colors      []string `json:"colors"`
}
type LegacyOrder struct {
	ID          string        `json:"id"`
	Number      string        `json:"number"`
	Token       string        `json:"token"`
	RequestKey  string        `json:"requestKey"`
	Fingerprint string        `json:"fingerprint"`
	Items       []m.OrderItem `json:"items"`
	Total       int64         `json:"total"`
	Note        string        `json:"note"`
	Customer    string        `json:"customer"`
	Contact     string        `json:"contact"`
	Pickup      string        `json:"pickup"`
	Status      string        `json:"status"`
	Paid        bool          `json:"paid"`
	CreatedAt   time.Time     `json:"createdAt"`
}
type LegacyState struct {
	Products []LegacyProduct `json:"products"`
	Settings m.ShopSetting   `json:"settings"`
	Orders   []LegacyOrder   `json:"orders"`
}

func initializeCatalog(tx *gorm.DB, state LegacyState) error {
	for index, p := range state.Products {
		var category m.Category
		if e := tx.Where("name = ?", p.Category).Attrs(m.Category{Name: p.Category, Position: index}).FirstOrCreate(&category).Error; e != nil {
			return e
		}
		product := m.Product{ID: p.ID, Name: p.Name, English: p.English, Description: p.Description, CategoryID: category.ID, Price: p.Price, Image: p.Image, Available: p.Available, Material: p.Material, Details: p.Details, Care: p.Care, Version: 1, Position: index}
		if e := tx.Create(&product).Error; e != nil {
			return e
		}
		if len(p.Options) == 0 {
			p.Options = []string{"标准款"}
		}
		if len(p.Colors) == 0 {
			p.Colors = []string{"原色"}
		}
		n := len(p.Options) * len(p.Colors)
		i := 0
		for _, option := range p.Options {
			for _, color := range p.Colors {
				stock := p.Stock / n
				if i < p.Stock%n {
					stock++
				}
				sku := m.ProductSKU{ProductID: p.ID, Variant: option, Color: color, Stock: stock, Active: true, Version: 1}
				if e := tx.Create(&sku).Error; e != nil {
					return e
				}
				if e := tx.Create(&m.StockMovement{SKUID: sku.ID, ProductID: p.ID, Delta: stock, Reason: "initial_stock", BusinessKey: fmt.Sprintf("seed:%d", sku.ID)}).Error; e != nil {
					return e
				}
				i++
			}
		}
	}
	setting := state.Settings
	setting.ID = 1
	setting.Version = 1
	return tx.Create(&setting).Error
}
func Bootstrap(ctx context.Context, db *gorm.DB, username, password string, seed bool) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var users int64
		if e := tx.Model(&m.AdminUser{}).Where("username = ?", username).Count(&users).Error; e != nil {
			return e
		}
		if users == 0 {
			if username == "" || len(password) < 10 {
				return fmt.Errorf("首次启动请在配置文件填写 bootstrap.admin_username 和至少10字符的 admin_password")
			}
			hash, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if e != nil {
				return e
			}
			if e = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&m.AdminUser{Username: username, PasswordHash: string(hash), Role: "admin", Enabled: true}).Error; e != nil {
				return e
			}
		}
		var settings int64
		if e := tx.Model(&m.ShopSetting{}).Count(&settings).Error; e != nil {
			return e
		}
		if settings == 0 {
			if seed {
				var data LegacyState
				if e := json.Unmarshal(catalogSeed, &data); e != nil {
					return e
				}
				return initializeCatalog(tx, data)
			}
			return tx.Create(&m.ShopSetting{ID: 1, Name: "素序 SUXU", Open: false, Version: 1}).Error
		}
		return nil
	})
}
func ImportLegacy(ctx context.Context, db *gorm.DB, data []byte) error {
	var state LegacyState
	if e := json.Unmarshal(data, &state); e != nil {
		return e
	}
	if len(state.Products) == 0 {
		return fmt.Errorf("旧数据无商品，拒绝导入")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		tx.Model(&m.MigrationRecord{}).Where("name = ?", "legacy-json-v1").Count(&count)
		if count > 0 {
			return api.Problem(409, "旧数据已经导入，不会重复执行")
		}
		if e := tx.Model(&m.Product{}).Count(&count).Error; e != nil {
			return e
		}
		if count > 0 {
			return api.Problem(409, "导入仅允许空商城库；已有商品时不会覆盖")
		}
		if e := initializeCatalog(tx, state); e != nil {
			return e
		}
		for _, o := range state.Orders {
			if o.RequestKey == "" || o.Token == "" {
				return fmt.Errorf("旧订单缺少幂等键或查询凭证，拒绝不完整导入")
			}
			order := m.Order{ID: o.ID, Number: o.Number, LookupToken: o.Token, RequestKey: o.RequestKey, Fingerprint: o.Fingerprint, Total: o.Total, Note: o.Note, Customer: o.Customer, Contact: o.Contact, Pickup: o.Pickup, Status: o.Status, Paid: o.Paid, Version: 1, CreatedAt: o.CreatedAt}
			var total int64
			for _, item := range o.Items {
				var sku m.ProductSKU
				if e := tx.Where("product_id = ? AND variant = ? AND color = ?", item.ProductID, item.Variant, item.Color).First(&sku).Error; e != nil {
					return fmt.Errorf("旧订单规格无法映射: %w", e)
				}
				item.ID = 0
				item.OrderID = order.ID
				item.SKUID = sku.ID
				order.Items = append(order.Items, item)
				total += item.Price * int64(item.Quantity)
			}
			if total != order.Total {
				return fmt.Errorf("旧订单金额不一致，事务已回滚")
			}
			if e := tx.Create(&order).Error; e != nil {
				return e
			}
		}
		return tx.Create(&m.MigrationRecord{Name: "legacy-json-v1", AppliedAt: time.Now()}).Error
	})
}
