package test

import (
	"Boutique/src/app"
	"Boutique/src/config"
	"Boutique/src/dao"
	"Boutique/src/global"
	m "Boutique/src/model/http_model"
	"Boutique/src/service"
	"Boutique/src/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fixture struct {
	db     *gorm.DB
	router http.Handler
	cfg    *config.AppConfig
}

func setup(t *testing.T, empty ...bool) *fixture {
	t.Helper()
	configPath := os.Getenv("SUXU_TEST_CONFIG")
	if configPath == "" {
		configPath = filepath.Join("..", "src", "config", "config.yaml")
	}
	cfg, e := config.LoadConfig(configPath)
	if e != nil {
		t.Fatal(e)
	}
	cfg.MySQL.DBName = "suxu_test_" + utils.RandomID()[:12]
	cfg.MySQL.AutoCreateDatabase = true
	cfg.Redis.Enable = false
	cfg.Server.Mode = "release"
	cfg.RateLimit.Enable = false
	cfg.JWT.Secret = "integration-test-secret-with-sufficient-length"
	global.Logger = zap.NewNop().Sugar()
	db, e := config.InitMysql(cfg.MySQL)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := db.Exec("DROP DATABASE `" + cfg.MySQL.DBName + "`").Error; e != nil {
			t.Errorf("清理测试专用库失败: %v", e)
		}
		if pool, e := db.DB(); e == nil {
			_ = pool.Close()
		}
	})
	if len(empty) == 0 || !empty[0] {
		if e = dao.Bootstrap(context.Background(), db, "test-admin", "test-admin-password", true); e != nil {
			t.Fatal(e)
		}
	}
	runtime, e := app.New(cfg, app.Dependencies{DB: db})
	if e != nil {
		t.Fatal(e)
	}
	return &fixture{db, runtime.Handler, cfg}
}
func call(h http.Handler, method, path string, body any, token, lookup string) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(raw))
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if lookup != "" {
		r.Header.Set("X-Order-Token", lookup)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func decode(t *testing.T, w *httptest.ResponseRecorder, target any) {
	t.Helper()
	var envelope struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
		Msg  string          `json:"msg"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &envelope); e != nil {
		t.Fatal(e, w.Body.String())
	}
	if envelope.Code != 200 {
		t.Fatalf("code=%d: %s", envelope.Code, envelope.Msg)
	}
	if target != nil {
		if e := json.Unmarshal(envelope.Data, target); e != nil {
			t.Fatal(e)
		}
	}
}
func login(t *testing.T, f *fixture) string {
	t.Helper()
	var response struct {
		Token string `json:"token"`
	}
	decode(t, call(f.router, "POST", "/api/v1/admin/login", m.LoginInput{Username: "test-admin", Password: "test-admin-password"}, "", ""), &response)
	if response.Token == "" {
		t.Fatal("登录未返回JWT")
	}
	return response.Token
}
func orderInput(key string) m.CreateOrderInput {
	return m.CreateOrderInput{Key: key, Customer: "集成验收", Pickup: "营业时间内自提", Items: []m.OrderLineInput{{ProductID: "p2", Quantity: 1, Variant: "港码 12", Color: "银白色"}}}
}
func create(t *testing.T, f *fixture, key string) m.OrderView {
	t.Helper()
	var order m.OrderView
	decode(t, call(f.router, "POST", "/api/v1/orders", orderInput(key), "", ""), &order)
	return order
}
func TestMySQLIntegration(t *testing.T) {
	if os.Getenv("RUN_MYSQL_TESTS") != "1" {
		t.Skip("真实MySQL用例需显式 RUN_MYSQL_TESTS=1；跳过不算验收通过")
	}
	t.Run("BusinessTablesAndSKUSeed", func(t *testing.T) {
		f := setup(t)
		var names []string
		if e := f.db.Raw("SELECT table_name FROM information_schema.tables WHERE table_schema = ? ORDER BY table_name", f.cfg.MySQL.DBName).Scan(&names).Error; e != nil {
			t.Fatal(e)
		}
		required := []string{"sys_category", "sys_product", "sys_product_sku", "sys_product_image", "sys_shop_setting", "sys_order", "sys_order_item", "sys_stock_movement", "sys_admin_user", "sys_admin_session", "sys_admin_audit_log", "sys_migration_record", "sys_login_attempt", "sys_customer_account", "sys_customer_session"}
		if len(names) != len(required) {
			t.Fatalf("业务表数量不正确: %v", names)
		}
		for _, name := range required {
			found := false
			for _, actual := range names {
				if actual == name {
					found = true
				}
			}
			if !found {
				t.Errorf("缺少表 %s", name)
			}
		}
		var skus []m.ProductSKU
		f.db.Where("product_id = ?", "p2").Find(&skus)
		if len(skus) != 4 {
			t.Fatalf("戒围应独立建SKU: %+v", skus)
		}
		for _, sku := range skus {
			if sku.Stock != 3 {
				t.Fatal("初始SKU分配不正确")
			}
		}
		var body map[string]any
		decode(t, call(f.router, "GET", "/api/v1/catalog", nil, "", ""), &body)
		if len(body["products"].([]any)) != 7 {
			t.Fatal("目录数量不正确")
		}
	})
	t.Run("LifecycleAuthorizationAndRevocation", func(t *testing.T) {
		f := setup(t)
		for _, path := range []string{"/api/v1/admin/session", "/api/v1/admin/catalog", "/api/v1/admin/orders", "/api/v1/admin/dashboard"} {
			if w := call(f.router, "GET", path, nil, "", ""); w.Code != 401 {
				t.Fatalf("未鉴权管理接口: %s %d", path, w.Code)
			}
		}
		token := login(t, f)
		order := create(t, f, "integration-lifecycle-001")
		lookupToken := order.Token
		if order.Total != 69900 || order.Token == "" || order.Items[0].Variant != "港码 12" {
			t.Fatal("订单价格或规格不正确")
		}
		if w := call(f.router, "GET", "/api/v1/orders/"+order.ID, nil, "", ""); w.Code != 404 {
			t.Fatal("订单无凭证可读")
		}
		for _, status := range []string{"confirmed", "ready", "completed"} {
			paid := true
			decode(t, call(f.router, "PATCH", "/api/v1/admin/orders/"+order.ID, m.UpdateOrderInput{Status: status, Paid: &paid, Version: order.Version}, token, ""), &order)
		}
		if order.Status != "completed" || !order.Paid {
			t.Fatal("完整状态流未闭环")
		}
		if w := call(f.router, "PATCH", "/api/v1/admin/orders/"+order.ID, m.UpdateOrderInput{Status: "pending", Version: order.Version}, token, ""); w.Code != 409 {
			t.Fatal("已完成订单允许倒退")
		}
		var looked m.OrderView
		w := call(f.router, "GET", "/api/v1/orders/"+order.ID, nil, "", lookupToken)
		decode(t, w, &looked)
		if strings.Contains(w.Body.String(), "requestKey") || strings.Contains(w.Body.String(), "fingerprint") {
			t.Fatal("内部幂等字段泄漏")
		}
		runtime, e := app.New(f.cfg, app.Dependencies{DB: f.db})
		if e != nil {
			t.Fatal(e)
		}
		if w = call(runtime.Handler, "GET", "/api/v1/admin/session", nil, token, ""); w.Code != 200 {
			t.Fatal("重建应用后持久化会话丢失")
		}
		decode(t, call(f.router, "POST", "/api/v1/admin/logout", map[string]any{}, token, ""), nil)
		if w = call(runtime.Handler, "GET", "/api/v1/admin/session", nil, token, ""); w.Code != 401 {
			t.Fatal("退出后的JWT仍有效")
		}
	})
	t.Run("MerchantCatalogSettingsAndDatabaseQueries", func(t *testing.T) {
		f := setup(t)
		token := login(t, f)
		input := m.ProductInput{Category: "首饰", Name: "验收专用首饰", Price: 19901, Image: "/images/jewelry.jpg", Available: true, SKUs: []m.SKUInput{{Variant: "标准款", Color: "香槟金", Stock: 5}}}
		details, care := "商家填写的商品细节", "避免受潮\n单独收纳"
		input.Details, input.Material, input.Care = details, "合成材质", &care
		var product m.ProductView
		decode(t, call(f.router, "PUT", "/api/v1/admin/products/new", input, token, ""), &product)
		if product.ID == "" || product.Stock != 5 || product.Price != 19901 {
			t.Fatal("商品或规格未准确落库")
		}
		if product.Care != care || product.Details != details || product.Material != input.Material {
			t.Fatal("材质、养护与细节没有独立保存")
		}
		input.Care = nil // 老版管理端没有养护字段。
		input.Version = product.Version
		input.Price = 20909
		input.SKUs[0].Stock = 7
		decode(t, call(f.router, "PUT", "/api/v1/admin/products/"+product.ID, input, token, ""), &product)
		if product.Stock != 7 || product.Price != 20909 {
			t.Fatal("商品编辑未持久化")
		}
		var publicCatalog struct{ Products []m.ProductView }
		decode(t, call(f.router, "GET", "/api/v1/catalog", nil, "", ""), &publicCatalog)
		found := false
		for _, item := range publicCatalog.Products {
			if item.ID == product.ID {
				found = item.Care == care && item.Material == input.Material && item.Details == details
				break
			}
		}
		if !found {
			t.Fatal("公开商品详情未保留后台资料或旧客户端覆盖了养护字段")
		}
		var catalog struct {
			Settings m.ShopSetting `json:"settings"`
		}
		decode(t, call(f.router, "GET", "/api/v1/admin/catalog", nil, token, ""), &catalog)
		settings := catalog.Settings
		settings.Open = false
		decode(t, call(f.router, "PUT", "/api/v1/admin/settings", settings, token, ""), &catalog.Settings)
		if w := call(f.router, "POST", "/api/v1/orders", orderInput("integration-closed-shop-001"), "", ""); w.Code != 409 {
			t.Fatal("暂停预约后仍可下单")
		}
		if w := call(f.router, "PUT", "/api/v1/admin/settings", settings, token, ""); w.Code != 409 {
			t.Fatal("店铺设置未检查版本")
		}
		settings = catalog.Settings
		settings.Open = true
		decode(t, call(f.router, "PUT", "/api/v1/admin/settings", settings, token, ""), &catalog.Settings)
		order := create(t, f, "integration-merchant-query-001")
		paid := true
		decode(t, call(f.router, "PATCH", "/api/v1/admin/orders/"+order.ID, m.UpdateOrderInput{Paid: &paid, Version: order.Version}, token, ""), &order)
		var list struct {
			Items []m.Order `json:"items"`
			Total int64     `json:"total"`
		}
		decode(t, call(f.router, "GET", "/api/v1/admin/orders?page=1&size=1&status=pending&search="+order.Number, nil, token, ""), &list)
		if list.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != order.ID {
			t.Fatal("订单筛选分页未返回数据库记录")
		}
		var dashboard struct {
			Revenue     int64 `json:"revenue"`
			TodayOrders int64 `json:"todayOrders"`
			Pending     int64 `json:"pending"`
		}
		decode(t, call(f.router, "GET", "/api/v1/admin/dashboard", nil, token, ""), &dashboard)
		if dashboard.Revenue != 69900 || dashboard.TodayOrders != 1 || dashboard.Pending != 1 {
			t.Fatalf("营业汇总不正确: %+v", dashboard)
		}
		var audits int64
		if e := f.db.Model(&m.AdminAuditLog{}).Count(&audits).Error; e != nil || audits < 5 {
			t.Fatalf("缺少管理操作记录: %d %v", audits, e)
		}
	})
	t.Run("LegacyImportAtomicityAndRepeatProtection", func(t *testing.T) {
		f := setup(t, true)
		raw, e := os.ReadFile(filepath.Join("..", "src", "dao", "catalog.seed.json"))
		if e != nil {
			t.Fatal(e)
		}
		var state dao.LegacyState
		if e = json.Unmarshal(raw, &state); e != nil {
			t.Fatal(e)
		}
		state.Orders = []dao.LegacyOrder{{ID: "legacy-acceptance-order", Number: "SU-LEGACY-001", Token: "legacy-private-lookup-token", RequestKey: "legacy-acceptance-key-001", Total: 1, Customer: "旧数据验收", Status: "ready", CreatedAt: time.Now(), Items: []m.OrderItem{{ProductID: "p2", Name: "历史戒指资料", Price: 69900, Quantity: 1, Variant: "港码 12", Color: "银白色"}}}}
		invalid, _ := json.Marshal(state)
		if e = dao.ImportLegacy(context.Background(), f.db, invalid); e == nil {
			t.Fatal("错误金额导入未拒绝")
		}
		var count int64
		if e = f.db.Model(&m.Product{}).Count(&count).Error; e != nil || count != 0 {
			t.Fatal("失败导入未回滚商品")
		}
		state.Orders[0].Total = 69900
		valid, _ := json.Marshal(state)
		if e = dao.ImportLegacy(context.Background(), f.db, valid); e != nil {
			t.Fatal(e)
		}
		if e = dao.ImportLegacy(context.Background(), f.db, valid); e == nil {
			t.Fatal("重复导入未拦截")
		}
		var order m.OrderView
		decode(t, call(f.router, "GET", "/api/v1/orders/legacy-acceptance-order", nil, "", "legacy-private-lookup-token"), &order)
		if order.Number != "SU-LEGACY-001" || order.Total != 69900 || order.Status != "ready" || order.Items[0].Name != "历史戒指资料" {
			t.Fatal("历史订单快照或查询凭证未保留")
		}
	})
	t.Run("ForgeryExpiryRoleAndRateLimit", func(t *testing.T) {
		f := setup(t)
		token := login(t, f)
		var claims service.Claims
		_, _, _ = new(jwt.Parser).ParseUnverified(token, &claims)
		for name, modify := range map[string]func(*service.Claims){"wrong-role": func(c *service.Claims) { c.Role = "customer" }, "expired": func(c *service.Claims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour)) }} {
			copy := claims
			modify(&copy)
			bad, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, copy).SignedString([]byte(f.cfg.JWT.Secret))
			if w := call(f.router, "GET", "/api/v1/admin/session", nil, bad, ""); w.Code != 401 {
				t.Fatalf("%s token 未拒绝", name)
			}
		}
		bad, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("different-secret"))
		if w := call(f.router, "GET", "/api/v1/admin/session", nil, bad, ""); w.Code != 401 {
			t.Fatal("伪造令牌未拒绝")
		}
		for i := 0; i < 5; i++ {
			w := call(f.router, "POST", "/api/v1/admin/login", m.LoginInput{Username: "test-admin", Password: "wrong"}, "", "")
			if w.Code != 401 {
				t.Fatalf("错误密码响应不正确: %d %s", w.Code, w.Body.String())
			}
		}
		if w := call(f.router, "POST", "/api/v1/admin/login", m.LoginInput{Username: "test-admin", Password: "test-admin-password"}, "", ""); w.Code != 429 {
			t.Fatal("登录限流未生效")
		}
	})
	t.Run("ConcurrentInventoryAndIdempotency", func(t *testing.T) {
		f := setup(t)
		var wg sync.WaitGroup
		codes := make(chan int, 12)
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				w := call(f.router, "POST", "/api/v1/orders", orderInput(fmt.Sprintf("concurrent-independent-%03d", i)), "", "")
				codes <- w.Code
			}(i)
		}
		wg.Wait()
		close(codes)
		success := 0
		for code := range codes {
			if code == 200 {
				success++
			} else if code != 409 {
				t.Fatalf("并发请求发生非业务错误: %d", code)
			}
		}
		if success != 3 {
			t.Fatalf("3件库存应成功3单，实际%d", success)
		}
		var sku m.ProductSKU
		f.db.Where("product_id = ? AND variant = ?", "p2", "港码 12").First(&sku)
		if sku.Stock != 0 {
			t.Fatal("SKU库存未归零")
		}
		input := orderInput("concurrent-same-idempotency")
		input.Items[0].Variant = "港码 14"
		ids := make(chan string, 8)
		errors := make(chan string, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				w := call(f.router, "POST", "/api/v1/orders", input, "", "")
				if w.Code != 200 {
					errors <- w.Body.String()
					return
				}
				var e struct {
					Data m.OrderView `json:"data"`
				}
				_ = json.Unmarshal(w.Body.Bytes(), &e)
				ids <- e.Data.ID
			}()
		}
		wg.Wait()
		close(ids)
		close(errors)
		for e := range errors {
			t.Fatal(e)
		}
		unique := map[string]bool{}
		for id := range ids {
			unique[id] = true
		}
		if len(unique) != 1 {
			t.Fatal("幂等请求产生重复订单")
		}
		sku = m.ProductSKU{}
		if e := f.db.Where("product_id = ? AND variant = ?", "p2", "港码 14").First(&sku).Error; e != nil {
			t.Fatal(e)
		}
		if sku.Stock != 2 {
			t.Fatal("幂等请求重复扣库存")
		}
	})
	t.Run("RollbackCancellationAndStaleEdits", func(t *testing.T) {
		f := setup(t)
		bad := orderInput("integration-rollback-001")
		bad.Items = []m.OrderLineInput{{ProductID: "p1", Quantity: 1, Variant: "标准款", Color: "奶油白"}, {ProductID: "p2", Quantity: 4, Variant: "港码 12", Color: "银白色"}}
		if w := call(f.router, "POST", "/api/v1/orders", bad, "", ""); w.Code != 409 {
			t.Fatal(w.Body.String())
		}
		var sku m.ProductSKU
		f.db.Where("product_id = ? AND color = ?", "p1", "奶油白").First(&sku)
		if sku.Stock != 9 {
			t.Fatal("失败订单未回滚先扣的库存")
		}
		var count int64
		f.db.Model(&m.Order{}).Count(&count)
		if count != 0 {
			t.Fatal("失败订单落库")
		}
		token := login(t, f)
		var catalog struct {
			Products []m.ProductView `json:"products"`
		}
		decode(t, call(f.router, "GET", "/api/v1/admin/catalog", nil, token, ""), &catalog)
		stale := catalog.Products[1]
		order := create(t, f, "integration-cancellation-001")
		input := m.ProductInput{ID: stale.ID, Category: stale.CategoryName, Name: stale.Name, English: stale.English, Description: stale.Description, Price: stale.Price, Image: stale.Image, Available: true, Version: stale.Version}
		for _, s := range stale.SKUs {
			input.SKUs = append(input.SKUs, m.SKUInput{ID: s.ID, Variant: s.Variant, Color: s.Color, Stock: s.Stock})
		}
		if w := call(f.router, "PUT", "/api/v1/admin/products/"+stale.ID, input, token, ""); w.Code != 409 {
			t.Fatal("旧版本商品编辑覆盖了新库存")
		}
		decode(t, call(f.router, "PATCH", "/api/v1/admin/orders/"+order.ID, m.UpdateOrderInput{Status: "cancelled", Version: order.Version}, token, ""), &order)
		version := order.Version
		decode(t, call(f.router, "PATCH", "/api/v1/admin/orders/"+order.ID, m.UpdateOrderInput{Status: "cancelled", Version: version}, token, ""), &order)
		sku = m.ProductSKU{}
		if e := f.db.Where("product_id = ? AND variant = ?", "p2", "港码 12").First(&sku).Error; e != nil {
			t.Fatal(e)
		}
		if sku.Stock != 3 {
			t.Fatal("取消预约未准确返还库存")
		}
		f.db.Model(&m.StockMovement{}).Where("order_id = ? AND reason = ?", order.ID, "order_cancelled").Count(&count)
		if count != 1 {
			t.Fatal("取消预约重复记库存流水")
		}
		changed := orderInput("integration-cancellation-001")
		changed.Customer = "变更内容"
		if w := call(f.router, "POST", "/api/v1/orders", changed, "", ""); w.Code != 409 {
			t.Fatal("同幂等键允许不同内容")
		}
	})
}

func TestCatalogAndDeliveryIntegration(t *testing.T) {
	if os.Getenv("RUN_MYSQL_TESTS") != "1" {
		t.Skip("需要临时 MySQL 实例")
	}
	f := setup(t, true)
	if err := dao.Bootstrap(context.Background(), f.db, "test-admin", "test-admin-password", false); err != nil {
		t.Fatal(err)
	}
	const public = "/api/v1/catalog"
	const admin = "/api/v1/admin"
	type catalogResult struct {
		Categories []m.Category
		Products   []m.ProductView
		Settings   m.ShopSetting
	}
	var catalog catalogResult
	decode(t, call(f.router, "GET", public, nil, "", ""), &catalog)
	if catalog.Categories == nil || catalog.Products == nil || len(catalog.Categories)+len(catalog.Products) != 0 {
		t.Fatalf("首次启动应为空目录: %+v", catalog)
	}
	if w := call(f.router, "PUT", admin+"/categories/new", m.CategoryInput{Name: "包包"}, "", ""); w.Code != 401 {
		t.Fatal("匿名用户可新增分类")
	}
	token := login(t, f)
	catalog.Settings.Open = true
	decode(t, call(f.router, "PUT", admin+"/settings", catalog.Settings, token, ""), nil)
	var category m.Category
	decode(t, call(f.router, "PUT", admin+"/categories/new", m.CategoryInput{Name: "包包"}, token, ""), &category)
	categoryPath := fmt.Sprintf("%s/categories/%d", admin, category.ID)
	decode(t, call(f.router, "GET", public, nil, "", ""), &catalog)
	if len(catalog.Categories) != 1 || len(catalog.Products) != 0 {
		t.Fatal("空分类应能单独显示")
	}
	if w := call(f.router, "PUT", admin+"/categories/new", m.CategoryInput{Name: "包包"}, token, ""); w.Code != 409 {
		t.Fatal("重复分类应返回 409")
	}

	// 使用真实 JPEG 检验后台上传与两个商城共享的公开图片接口。
	data, err := os.ReadFile(filepath.Join("..", "..", "frontend", "public", "images", "bag.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	var payload bytes.Buffer
	form := multipart.NewWriter(&payload)
	part, err := form.CreateFormFile("image", "bag.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	upload := httptest.NewRequest("POST", admin+"/media", &payload)
	upload.Header.Set("Content-Type", form.FormDataContentType())
	upload.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, upload)
	var image struct{ URL string }
	decode(t, w, &image)
	w = call(f.router, "GET", image.URL, nil, "", "")
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/jpeg" || !bytes.Equal(w.Body.Bytes(), data) {
		t.Fatal("上传图片无法从公开接口原样读取")
	}

	productInput := m.ProductInput{CategoryID: category.ID, Name: "测试包", Price: 2800, Image: image.URL, Available: true,
		SKUs: []m.SKUInput{{Variant: "标准款", Color: "原色", Stock: 3}}}
	var product m.ProductView
	decode(t, call(f.router, "PUT", admin+"/products/new", productInput, token, ""), &product)
	decode(t, call(f.router, "PUT", categoryPath, m.CategoryInput{Name: "帆布包"}, token, ""), &category)
	decode(t, call(f.router, "GET", public, nil, "", ""), &catalog)
	if len(catalog.Products) != 1 || catalog.Products[0].CategoryName != "帆布包" || catalog.Products[0].CategoryID != category.ID {
		t.Fatal("改名未同步到已有商品")
	}

	orderInput := m.CreateOrderInput{Fulfillment: "delivery", Customer: "测试顾客", Contact: "13800000000", Key: "integration-delivery-1",
		Items: []m.OrderLineInput{{ProductID: product.ID, Quantity: 2, Variant: "标准款", Color: "原色"}}}
	if w := call(f.router, "POST", "/api/v1/orders", orderInput, "", ""); w.Code != 400 {
		t.Fatal("送货上门未检查地址")
	}
	orderInput.Address = "测试地址 1 号"
	var order, retry m.OrderView
	decode(t, call(f.router, "POST", "/api/v1/orders", orderInput, "", ""), &order)
	decode(t, call(f.router, "POST", "/api/v1/orders", orderInput, "", ""), &retry)
	if order.Total != 5800 || order.DeliveryFee != 200 || order.Fulfillment != "delivery" || order.Address != orderInput.Address || retry.ID != order.ID {
		t.Fatalf("配送订单金额、地址或幂等不正确: %+v", order.Order)
	}
	decode(t, call(f.router, "DELETE", categoryPath, nil, token, ""), nil)
	decode(t, call(f.router, "GET", public, nil, "", ""), &catalog)
	if len(catalog.Categories)+len(catalog.Products) != 0 {
		t.Fatal("删除分类后仍有目录数据")
	}
	if w := call(f.router, "PUT", admin+"/products/new", productInput, token, ""); w.Code != 409 {
		t.Fatal("删除的分类仍可新增商品")
	}
	var cancelled m.Order
	decode(t, call(f.router, "PATCH", admin+"/orders/"+order.ID, m.UpdateOrderInput{Status: "cancelled", Version: order.Version}, token, ""), &cancelled)
	if cancelled.Status != "cancelled" || len(cancelled.Items) != 1 || cancelled.Items[0].Name != product.Name || cancelled.Items[0].Image != image.URL {
		t.Fatal("删除分类后历史订单丢失或无法取消")
	}
	if w := call(f.router, "GET", image.URL, nil, "", ""); w.Code != 200 {
		t.Fatal("历史订单图片丢失")
	}
}

func TestCustomerAccountAndCoverIntegration(t *testing.T) {
	if os.Getenv("RUN_MYSQL_TESTS") != "1" {
		t.Skip("需要临时 MySQL 实例")
	}
	f := setup(t)
	const base = "https://suxu.test/api/v1/customers"
	credentials := m.CustomerCredentials{Contact: "test_wechat_01", Password: "fixture-password-01", Name: "测试顾客"}
	if w := call(f.router, "POST", "/api/v1/customers/register", credentials, "", ""); w.Code != 403 {
		t.Fatalf("非本机明文连接未拒绝: %d", w.Code)
	}
	type session struct {
		Account m.CustomerAccount
		Token   string
	}
	var first, second, restored session
	decode(t, call(f.router, "POST", base+"/register", credentials, "", ""), &first)
	if !strings.HasPrefix(first.Account.ID, "SU-") || first.Token == "" {
		t.Fatal("注册缺少顾客编号或会话")
	}
	if w := call(f.router, "POST", base+"/register", credentials, "", ""); w.Code != 409 {
		t.Fatal("允许重复注册")
	}
	bad := credentials
	bad.Password = "incorrect-password"
	if w := call(f.router, "POST", base+"/login", bad, "", ""); w.Code != 401 {
		t.Fatal("错误密码被接受")
	}
	decode(t, call(f.router, "POST", base+"/login", credentials, "", ""), &restored)
	if restored.Account.ID != first.Account.ID || restored.Token == first.Token {
		t.Fatal("同一账号登录后顾客编号变化或会话重复")
	}
	credentials.Contact = "test_wechat_02"
	decode(t, call(f.router, "POST", base+"/register", credentials, "", ""), &second)
	var profile m.CustomerAccount
	decode(t, call(f.router, "PUT", base+"/profile", m.CustomerProfileInput{Name: "更新称呼", Dorm: "3 栋 508"}, restored.Token, ""), &profile)
	if profile.ID != first.Account.ID || profile.Dorm != "3 栋 508" {
		t.Fatal("账号资料未保存")
	}
	req := orderInput("customer-integration-order")
	req.Customer, req.Contact, req.Pickup = "伪造称呼", "", "明日喜提"
	var order m.OrderView
	decode(t, call(f.router, "POST", base+"/orders", req, restored.Token, ""), &order)
	if order.CustomerID != first.Account.ID || order.Customer != "更新称呼" || order.Contact != "test_wechat_01" || order.Token != "" {
		t.Fatal("订单没有关联已登录的顾客资料，或泄露了旧版查询凭证")
	}
	var orders []m.Order
	decode(t, call(f.router, "GET", base+"/orders", nil, second.Token, ""), &orders)
	if len(orders) != 0 {
		t.Fatal("另一账号能看见订单列表")
	}
	if w := call(f.router, "GET", base+"/orders/"+order.ID, nil, second.Token, ""); w.Code != 404 {
		t.Fatal("另一账号能读取订单详情")
	}
	decode(t, call(f.router, "GET", base+"/orders/"+order.ID, nil, first.Token, ""), &order)
	decode(t, call(f.router, "POST", base+"/logout", map[string]any{}, restored.Token, ""), nil)
	if w := call(f.router, "GET", base+"/profile", nil, restored.Token, ""); w.Code != 401 {
		t.Fatal("退出登录未撤销会话")
	}
	adminToken := login(t, f)
	var catalog struct{ Settings m.ShopSetting }
	decode(t, call(f.router, "GET", "/api/v1/admin/catalog", nil, adminToken, ""), &catalog)
	catalog.Settings.QuickCover = "/images/bag.jpg"
	decode(t, call(f.router, "PUT", "/api/v1/admin/settings", catalog.Settings, adminToken, ""), nil)
	decode(t, call(f.router, "GET", "/api/v1/catalog", nil, "", ""), &catalog)
	if catalog.Settings.QuickCover != "/images/bag.jpg" {
		t.Fatal("快速商城未获得已保存的封面")
	}
}

func TestCheckoutExpectedTotalIntegration(t *testing.T) {
	if os.Getenv("RUN_MYSQL_TESTS") != "1" {
		t.Skip("需要临时 MySQL 实例")
	}
	f := setup(t)
	input := orderInput("price-confirmation-integration")
	oldTotal := int64(69900)
	input.ExpectedTotal = &oldTotal
	input.Fulfillment = "delivery"
	input.Contact = "fixture_contact"
	input.Address = "测试寝室"
	response := call(f.router, "POST", "/api/v1/orders", input, "", "")
	if response.Code != 409 || !strings.Contains(response.Body.String(), "请重新确认金额") {
		t.Fatalf("旧金额应返回 409，实际 %d", response.Code)
	}
	var sku m.ProductSKU
	if err := f.db.Where("product_id = ? AND variant = ? AND color = ?", "p2", input.Items[0].Variant, input.Items[0].Color).First(&sku).Error; err != nil || sku.Stock != 3 {
		t.Fatalf("价格拒绝后库存应回滚，当前 %d，错误 %v", sku.Stock, err)
	}
	var movements int64
	if err := f.db.Model(&m.StockMovement{}).Where("reason = ?", "order_created").Count(&movements).Error; err != nil || movements != 0 {
		t.Fatalf("价格拒绝后不应留下订单流水，当前 %d，错误 %v", movements, err)
	}
	confirmedTotal := int64(70100)
	input.ExpectedTotal = &confirmedTotal
	var first, retry m.OrderView
	decode(t, call(f.router, "POST", "/api/v1/orders", input, "", ""), &first)
	if err := f.db.Model(&m.ShopSetting{}).Where("id = ?", 1).Update("open", false).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Model(&m.Product{}).Where("id = ?", "p2").Update("available", false).Error; err != nil {
		t.Fatal(err)
	}
	input.ExpectedTotal = &oldTotal
	decode(t, call(f.router, "POST", "/api/v1/orders", input, "", ""), &retry)
	if first.ID == "" || first.ID != retry.ID || retry.Total != confirmedTotal {
		t.Fatal("重复请求应取得原预约和原成交金额")
	}
}
