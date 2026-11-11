package service

import (
	m "Boutique/src/model/http_model"
	"context"
	"testing"
)

func TestInvalidRequestsNeverReachDAO(t *testing.T) {
	catalog := CatalogService{}
	for _, input := range []m.ProductInput{{}, {Name: "测试", Category: "布包", Price: 100, Image: "javascript:alert(1)", SKUs: []m.SKUInput{{Variant: "标准", Color: "白", Stock: 1}}}, {Name: "测试", Category: "布包", Price: 100, Image: "/images/bag.jpg", SKUs: []m.SKUInput{{Variant: "标准", Color: "白", Stock: -1}}}} {
		if _, e := catalog.SaveProduct(context.Background(), "new", input, 1); e == nil {
			t.Fatal("非法商品未拒绝")
		}
	}
	order := OrderService{}
	if _, e := order.Create(context.Background(), m.CreateOrderInput{}); e == nil {
		t.Fatal("非法订单未拒绝")
	}
}
