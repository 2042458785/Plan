package app

import (
	"Boutique/src/config"
	"Boutique/src/dao"
	"Boutique/src/global"
	"Boutique/src/router"
	"Boutique/src/service"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Dependencies struct {
	DB    *gorm.DB
	Redis *redis.Client
}
type Runtime struct{ Handler *gin.Engine }

func New(cfg *config.AppConfig, deps Dependencies) (*Runtime, error) {
	if cfg == nil || deps.DB == nil {
		return nil, fmt.Errorf("配置及数据库依赖不能为空")
	}
	if cfg.Redis.Enable && deps.Redis == nil {
		return nil, fmt.Errorf("Redis已启用，必须提供可用连接")
	}
	global.DB = deps.DB
	global.Redis = deps.Redis
	catalog := &service.CatalogService{Dao: &dao.CatalogDao{Orm: deps.DB}}
	orders := &service.OrderService{Dao: &dao.OrderDao{Orm: deps.DB}}
	auth := &service.AuthService{Dao: &dao.AuthDao{Orm: deps.DB}, Config: cfg.JWT, Redis: deps.Redis, Namespace: cfg.MySQL.DBName}
	return &Runtime{Handler: router.SetupRouter(cfg, deps.DB, catalog, orders, auth)}, nil
}
