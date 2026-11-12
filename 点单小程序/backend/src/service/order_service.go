package service

import (
	"Boutique/src/api"
	"Boutique/src/dao"
	m "Boutique/src/model/http_model"
	"context"
	"strings"
)

type OrderService struct{ Dao *dao.OrderDao }

func (s *OrderService) Create(ctx context.Context, in m.CreateOrderInput) (m.OrderView, error) {
	in.Customer = strings.TrimSpace(in.Customer)
	in.Note = strings.TrimSpace(in.Note)
	in.Address = strings.TrimSpace(in.Address)
	if in.WeChat != nil {
		in.Contact = strings.TrimSpace(*in.WeChat)
		if !wechatID.MatchString(strings.ToLower(in.Contact)) {
			return m.OrderView{}, api.Problem(400, "请填写有效的微信号，以字母开头，6–20 位")
		}
		// 统一保存到原联系方式字段，保持同一订单的幂等指纹。
		in.WeChat = nil
	}
	switch in.Fulfillment {
	case "", "pickup":
		// 旧版自提请求使用相同的幂等指纹。
		in.Fulfillment = ""
		in.Address = ""
	case "delivery":
		in.Contact = strings.TrimSpace(in.Contact)
		if in.Address == "" || len(in.Address) > 300 || in.Contact == "" {
			return m.OrderView{}, api.Problem(400, "送货上门需要填写联系方式和配送地址")
		}
		in.Pickup = ""
	default:
		return m.OrderView{}, api.Problem(400, "请选择自提或送货上门")
	}
	if len(in.Items) == 0 || len(in.Items) > 30 || len(in.Key) < 16 || len(in.Key) > 100 || in.Customer == "" || len(in.Customer) > 100 || len(in.Note) > 600 || len(in.Contact) > 100 || len(in.Pickup) > 100 {
		return m.OrderView{}, api.Problem(400, "请填写联系人称呼并检查商品和备注")
	}
	for _, it := range in.Items {
		if it.Quantity < 1 || it.Quantity > 50 || len(it.ProductID) > 48 || it.Variant == "" || it.Color == "" || len(it.Variant) > 100 || len(it.Color) > 100 {
			return m.OrderView{}, api.Problem(400, "商品规格、颜色或数量不正确")
		}
	}
	return s.Dao.Create(ctx, in)
}

func (s *OrderService) Lookup(ctx context.Context, id, token string) (m.OrderView, error) {
	if len(id) > 48 || len(token) > 100 {
		return m.OrderView{}, api.Problem(404, "预约不存在")
	}
	return s.Dao.Lookup(ctx, id, token)
}
func (s *OrderService) List(ctx context.Context, page, size int, status, search string) ([]m.Order, int64, error) {
	valid := map[string]bool{"": true, "all": true, "active": true, "pending": true, "confirmed": true, "ready": true, "completed": true, "cancelled": true}
	if !valid[status] || len(search) > 100 || page < 1 || size < 1 || size > 200 {
		return nil, 0, api.Problem(400, "筛选或分页参数不正确")
	}
	return s.Dao.List(ctx, page, size, status, search)
}
func (s *OrderService) Update(ctx context.Context, id string, input m.UpdateOrderInput, adminID uint) (m.Order, error) {
	if len(id) > 48 || input.Version == 0 {
		return m.Order{}, api.Problem(400, "预约编号或版本号不正确")
	}
	return s.Dao.Update(ctx, id, input, adminID)
}

func (s *OrderService) CustomerOrders(ctx context.Context, customerID string) ([]m.Order, error) {
	return s.Dao.CustomerOrders(ctx, customerID)
}

func (s *OrderService) CustomerOrder(ctx context.Context, customerID, id string) (m.Order, error) {
	if len(id) > 48 {
		return m.Order{}, api.Problem(404, "预约不存在")
	}
	return s.Dao.CustomerOrder(ctx, customerID, id)
}
