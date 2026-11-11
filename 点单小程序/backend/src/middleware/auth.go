package middleware

import (
	"Boutique/src/api"
	"Boutique/src/config"
	"Boutique/src/service"
	"github.com/gin-gonic/gin"
	"strings"
)

func AuthMiddleware(cfg *config.AppConfig, auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		public := path == "/api/v1/catalog" || path == "/api/v1/orders" || strings.HasPrefix(path, "/api/v1/orders/") || path == "/api/v1/admin/login" || path == "/healthz"
		if public {
			for _, prefix := range cfg.JWT.Whitelist {
				if path == prefix || strings.HasPrefix(path, prefix+"/") {
					c.Next()
					return
				}
			}
			if path == "/healthz" {
				c.Next()
				return
			}
			api.Fail(c, 401, "该接口未配置为开放接口")
			return
		}
		if !strings.HasPrefix(path, "/api/v1/admin/") {
			c.Next()
			return
		}
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") || len(header) < 8 {
			api.Fail(c, 401, "请先登录商家工作室")
			return
		}
		claims, e := auth.Validate(c.Request.Context(), strings.TrimPrefix(header, "Bearer "))
		if e != nil {
			api.RespondError(c, e)
			return
		}
		c.Set("adminID", claims.AdminID)
		c.Set("sessionID", claims.ID)
		c.Next()
	}
}
