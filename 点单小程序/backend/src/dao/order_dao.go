package dao

import (
	"Boutique/src/api"
	m "Boutique/src/model/http_model"
	"Boutique/src/utils"
	"context"
	"crypto/subtle"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sort"
	"strings"
	"time"
)

type OrderDao struct{ Orm *gorm.DB }

func (d *OrderDao) Create(ctx context.Context, input m.CreateOrderInput) (m.OrderView, error) {
	var result m.OrderView
	fingerprint := utils.Fingerprint(input)
	e := utils.WithTransaction(ctx, d.Orm, func(tx *gorm.DB) error {
		// 锁住店铺设置后做幂等校验，保证跨进程请求也不会重复落单。
		var settings m.ShopSetting
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&settings, 1).Error; e != nil {
			return e
		}
		var existing m.Order
		e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Items").First(&existing, "request_key = ?", input.Key).Error
		if e == nil {
			if existing.Fingerprint != fingerprint || existing.CustomerID != input.CustomerID {
				return api.Problem(409, "幂等键已用于不同内容，请重新确认预约")
			}
			result = m.OrderView{Order: existing, Token: existing.LookupToken}
			return nil
		}
		if e != gorm.ErrRecordNotFound {
			return e
		}
		if !settings.Open {
			return api.Problem(409, "店铺暂未接受预约")
		}
		type line struct {
			m.OrderLineInput
			SKU     m.ProductSKU
			Product m.Product
		}
		lines := make([]line, 0, len(input.Items))
		for _, in := range input.Items {
			var sku m.ProductSKU
			if e = tx.Where("product_id = ? AND variant = ? AND color = ? AND active = ?", in.ProductID, in.Variant, in.Color, true).First(&sku).Error; e != nil {
				if e == gorm.ErrRecordNotFound {
					return api.Problem(409, "商品规格或颜色已变化，请刷新")
				}
				return e
			}
			lines = append(lines, line{OrderLineInput: in, SKU: sku})
		}
		sort.Slice(lines, func(i, j int) bool {
			if lines[i].ProductID == lines[j].ProductID {
				return lines[i].SKU.ID < lines[j].SKU.ID
			}
			return lines[i].ProductID < lines[j].ProductID
		})
		order := m.Order{CustomerID: input.CustomerID, ID: utils.RandomID(), LookupToken: utils.RandomID(), RequestKey: input.Key, Fingerprint: fingerprint, Customer: input.Customer, Contact: input.Contact, Pickup: input.Pickup, Note: input.Note, Status: "pending", Version: 1, CreatedAt: time.Now(), Items: []m.OrderItem{}}
		order.Fulfillment = "pickup"
		if input.Fulfillment == "delivery" {
			order.Fulfillment = input.Fulfillment
			order.Address = input.Address
			order.DeliveryFee = m.DeliveryFeeCents
			order.Total = order.DeliveryFee
		}
		order.Number = "SU" + order.CreatedAt.Format("060102") + strings.ToUpper(order.ID[:12])
		// 所有商品及 SKU 按固定顺序加锁，和后台编辑保持一致。
		lockedProducts := map[string]m.Product{}
		for i := range lines {
			l := &lines[i]
			p, ok := lockedProducts[l.ProductID]
			if !ok {
				if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, "id = ?", l.ProductID).Error; e != nil {
					if e == gorm.ErrRecordNotFound {
						return api.Problem(409, "商品已删除，请刷新购物袋")
					}
					return e
				}
				lockedProducts[p.ID] = p
			}
			l.Product = p
			if !p.Available {
				return api.Problem(409, "商品已下架，请调整购物袋")
			}
			if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&l.SKU, l.SKU.ID).Error; e != nil {
				return e
			}
			if !l.SKU.Active || l.SKU.Stock < l.Quantity {
				return api.Problem(409, "所选规格库存不足，请调整数量")
			}
			updated := tx.Model(&m.ProductSKU{}).Where("id = ? AND stock >= ? AND active = ?", l.SKU.ID, l.Quantity, true).Updates(map[string]any{"stock": gorm.Expr("stock - ?", l.Quantity), "version": gorm.Expr("version + 1")})
			if updated.Error != nil {
				return updated.Error
			}
			if updated.RowsAffected != 1 {
				return api.Problem(409, "库存已变化，请重试")
			}
			order.Items = append(order.Items, m.OrderItem{ProductID: p.ID, SKUID: l.SKU.ID, Name: p.Name, Price: p.Price, Quantity: l.Quantity, Variant: l.Variant, Color: l.Color, Image: p.Image})
			order.Total += p.Price * int64(l.Quantity)
			if e = tx.Create(&m.StockMovement{SKUID: l.SKU.ID, ProductID: p.ID, OrderID: order.ID, Delta: -l.Quantity, Reason: "order_created", BusinessKey: fmt.Sprintf("reserve:%s:%d:%d", order.ID, l.SKU.ID, i)}).Error; e != nil {
				return e
			}
		}
		for id := range lockedProducts {
			if e = tx.Model(&m.Product{}).Where("id = ?", id).UpdateColumn("version", gorm.Expr("version + 1")).Error; e != nil {
				return e
			}
		}
		if e = tx.Create(&order).Error; e != nil {
			return e
		}
		result = m.OrderView{Order: order, Token: order.LookupToken}
		return nil
	})
	return result, e
}
func (d *OrderDao) Lookup(ctx context.Context, id, token string) (m.OrderView, error) {
	var order m.Order
	e := d.Orm.WithContext(ctx).Preload("Items").First(&order, "id = ?", id).Error
	if e == gorm.ErrRecordNotFound {
		return m.OrderView{}, api.Problem(404, "预约不存在")
	}
	if e != nil {
		return m.OrderView{}, e
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(order.LookupToken)) != 1 {
		return m.OrderView{}, api.Problem(404, "预约不存在")
	}
	return m.OrderView{Order: order, Token: order.LookupToken}, nil
}
func (d *OrderDao) List(ctx context.Context, page, size int, status, search string) ([]m.Order, int64, error) {
	var orders []m.Order
	var total int64
	q := d.Orm.WithContext(ctx).Model(&m.Order{})
	if status == "active" {
		q = q.Where("status IN ?", []string{"pending", "confirmed", "ready"})
	} else if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}
	if search != "" {
		pattern := "%" + search + "%"
		q = q.Where("number LIKE ? OR customer LIKE ? OR customer_id LIKE ? OR EXISTS (SELECT 1 FROM sys_order_item AS i WHERE i.order_id = sys_order.id AND i.name LIKE ?)", pattern, pattern, pattern, pattern)
	}
	if e := q.Count(&total).Error; e != nil {
		return nil, 0, e
	}
	e := q.Preload("Items").Order("created_at DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&orders).Error
	return orders, total, e
}
func (d *OrderDao) Update(ctx context.Context, id string, input m.UpdateOrderInput, adminID uint) (m.Order, error) {
	var result m.Order
	e := utils.WithTransaction(ctx, d.Orm, func(tx *gorm.DB) error {
		var order m.Order
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Items").First(&order, "id = ?", id).Error; e != nil {
			if e == gorm.ErrRecordNotFound {
				return api.Problem(404, "预约不存在")
			}
			return e
		}
		if order.Version != input.Version {
			return api.Problem(409, "预约状态已变化，请刷新后重试")
		}
		if input.Status != "" && input.Status != order.Status {
			allowed := map[string]map[string]bool{"pending": {"confirmed": true, "cancelled": true}, "confirmed": {"ready": true, "cancelled": true}, "ready": {"completed": true, "cancelled": true}}
			if !allowed[order.Status][input.Status] {
				return api.Problem(409, "不允许的预约状态变更")
			}
			if input.Status == "cancelled" {
				items := append([]m.OrderItem{}, order.Items...)
				sort.Slice(items, func(i, j int) bool {
					if items[i].ProductID == items[j].ProductID {
						return items[i].SKUID < items[j].SKUID
					}
					return items[i].ProductID < items[j].ProductID
				})
				changed := map[string]bool{}
				deleted := map[string]bool{}
				for _, it := range items {
					if deleted[it.ProductID] {
						continue
					}
					if !changed[it.ProductID] {
						var p m.Product
						if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, "id = ?", it.ProductID).Error; e != nil {
							if e == gorm.ErrRecordNotFound {
								deleted[it.ProductID] = true
								continue
							}
							return e
						}
						changed[it.ProductID] = true
					}
					var sku m.ProductSKU
					if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&sku, it.SKUID).Error; e != nil {
						return e
					}
					if e := tx.Model(&sku).Updates(map[string]any{"stock": gorm.Expr("stock + ?", it.Quantity), "version": gorm.Expr("version + 1")}).Error; e != nil {
						return e
					}
					if e := tx.Create(&m.StockMovement{SKUID: it.SKUID, ProductID: it.ProductID, OrderID: order.ID, Delta: it.Quantity, Reason: "order_cancelled", BusinessKey: fmt.Sprintf("release:%s:%d", order.ID, it.ID)}).Error; e != nil {
						return e
					}
				}
				for pid := range changed {
					if e := tx.Model(&m.Product{}).Where("id = ?", pid).UpdateColumn("version", gorm.Expr("version + 1")).Error; e != nil {
						return e
					}
				}
			}
			order.Status = input.Status
		}
		if input.Paid != nil {
			order.Paid = *input.Paid
		}
		order.Version++
		if e := tx.Omit("Items").Save(&order).Error; e != nil {
			return e
		}
		if e := tx.Create(&m.AdminAuditLog{AdminID: adminID, Action: "update_order", ResourceID: order.ID}).Error; e != nil {
			return e
		}
		result = order
		return nil
	})
	return result, e
}

func (d *OrderDao) CustomerOrders(ctx context.Context, customerID string) ([]m.Order, error) {
	orders := []m.Order{}
	err := d.Orm.WithContext(ctx).Where("customer_id = ?", customerID).
		Preload("Items").Order("created_at DESC, id DESC").Limit(100).Find(&orders).Error
	return orders, err
}

func (d *OrderDao) CustomerOrder(ctx context.Context, customerID, id string) (m.Order, error) {
	var order m.Order
	err := d.Orm.WithContext(ctx).Where("customer_id = ? AND id = ?", customerID, id).Preload("Items").First(&order).Error
	if err == gorm.ErrRecordNotFound {
		return order, api.Problem(404, "预约不存在")
	}
	return order, err
}
