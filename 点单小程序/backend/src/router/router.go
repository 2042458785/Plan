package router

import (
	"Boutique/src/api"
	ctl "Boutique/src/api/http_controller"
	"Boutique/src/config"
	"Boutique/src/middleware"
	"Boutique/src/service"
	"Boutique/src/theValidator"
	"context"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"time"
)

func SetupRouter(cfg *config.AppConfig, db *gorm.DB, catalog *service.CatalogService, orders *service.OrderService, auth *service.AuthService) *gin.Engine {
	if err := theValidator.RegisterCustValidator1(); err != nil {
		panic(err)
	}
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(middleware.Recovery(), middleware.SecurityHeaders(), middleware.CorsMiddleware(cfg), middleware.RateLimit(cfg.RateLimit), middleware.AuthMiddleware(cfg, auth))
	catalogController := ctl.NewCatalogController(catalog)
	orderController := ctl.NewOrderController(orders)
	authController := ctl.NewAuthController(auth)
	r.GET("/healthz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		pool, e := db.DB()
		if e == nil {
			e = pool.PingContext(ctx)
		}
		if e != nil {
			api.Fail(c, 503, "数据库未就绪")
			return
		}
		if auth.Redis != nil {
			if e = auth.Redis.Ping(ctx).Err(); e != nil {
				api.Fail(c, 503, "Redis未就绪")
				return
			}
		}
		api.Ok(c, gin.H{"status": "ok"})
	})
	v1 := r.Group("/api/v1")
	v1.GET("/catalog", catalogController.Catalog)
	v1.GET("/media/:id", catalogController.Image)
	v1.POST("/orders", orderController.Create)
	v1.GET("/orders/:id", orderController.Lookup)
	// 所有管理接口都经过服务端 JWT、角色与持久化会话校验。
	admin := v1.Group("/admin")
	admin.POST("/login", authController.Login)
	admin.GET("/session", authController.Session)
	admin.POST("/logout", authController.Logout)
	admin.GET("/catalog", catalogController.Catalog)
	admin.GET("/dashboard", catalogController.Dashboard)
	admin.GET("/orders", orderController.List)
	admin.PATCH("/orders/:id", orderController.Update)
	admin.PUT("/products/:id", catalogController.SaveProduct)
	admin.PUT("/categories/:id", catalogController.SaveCategory)
	admin.DELETE("/categories/:id", catalogController.DeleteCategory)
	admin.POST("/media", catalogController.UploadImage)
	admin.PUT("/settings", catalogController.SaveSettings)
	r.NoRoute(func(c *gin.Context) { api.Fail(c, 404, "接口不存在") })
	return r
}
