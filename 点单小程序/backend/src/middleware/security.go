package middleware

import (
	"Boutique/src/api"
	"Boutique/src/config"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

func CorsMiddleware(cfg *config.AppConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			allowed := origin == "http://"+c.Request.Host || origin == "https://"+c.Request.Host
			if origin == cfg.Server.AllowOrigin || (cfg.Server.AllowOrigin == "*" && cfg.Server.Mode == "debug") {
				allowed = true
			}
			for _, v := range cfg.Server.AllowOrigins {
				if origin == v {
					allowed = true
				}
			}
			if !allowed {
				api.Fail(c, 403, "请求来源不被允许")
				return
			}
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			methods := cfg.Server.AllowMethods
			if methods == "" {
				methods = "GET,POST,PUT,PATCH,DELETE,OPTIONS"
			}
			c.Header("Access-Control-Allow-Methods", methods)
			headers := cfg.Server.AllowHeaders
			if headers == "" {
				headers = "Content-Type,Authorization,X-Order-Token"
			}
			c.Header("Access-Control-Allow-Headers", headers)
			age := cfg.Server.CorsMaxAge
			if age <= 0 {
				age = 600
			}
			c.Header("Access-Control-Max-Age", strconv.Itoa(age))
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Cache-Control", "no-store")
		if c.Request.Method == http.MethodPost && c.Request.URL.Path == "/api/v1/admin/media" {
			if !strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data;") {
				api.Fail(c, 415, "请选择要上传的图片")
				return
			}
			c.Next()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.Request.Method != "OPTIONS" && !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
			api.Fail(c, 415, "请求需使用 application/json")
			return
		}
		c.Next()
	}
}

type bucket struct {
	tokens  float64
	updated time.Time
}

func RateLimit(cfg config.RateLimitConfig) gin.HandlerFunc {
	var mu sync.Mutex
	buckets := map[string]bucket{}
	return func(c *gin.Context) {
		if !cfg.Enable {
			c.Next()
			return
		}
		now := time.Now()
		mu.Lock()
		if len(buckets) > 10000 {
			for key, b := range buckets {
				if now.Sub(b.updated) > 10*time.Minute {
					delete(buckets, key)
				}
			}
		}
		key := c.ClientIP()
		b, ok := buckets[key]
		if !ok {
			b = bucket{float64(cfg.Capacity), now}
		}
		b.tokens += now.Sub(b.updated).Seconds() * cfg.Rate
		if b.tokens > float64(cfg.Capacity) {
			b.tokens = float64(cfg.Capacity)
		}
		b.updated = now
		allowed := b.tokens >= 1
		if allowed {
			b.tokens--
		}
		buckets[key] = b
		mu.Unlock()
		if !allowed {
			api.Fail(c, 429, "请求过于频繁，请稍后重试")
			return
		}
		c.Next()
	}
}
