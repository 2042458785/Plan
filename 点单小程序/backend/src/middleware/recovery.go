package middleware

import (
	"Boutique/src/api"
	"Boutique/src/global"
	"fmt"
	"github.com/gin-gonic/gin"
)

// Recovery 统一使用 Zap，避免把鉴权头、订单凭证和请求正文写入 panic 日志。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if value := recover(); value != nil {
				if global.Logger != nil {
					global.Logger.Errorw("请求处理异常", "module", "middleware", "op", "Recovery", "err", fmt.Errorf("%v", value))
				}
				api.Fail(c, 500, "服务处理异常，请稍后重试")
			}
		}()
		c.Next()
	}
}
