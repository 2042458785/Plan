package config

import (
	"Boutique/src/global"
	m "Boutique/src/model/http_model"
	"context"
	"database/sql"
	"fmt"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"net"
	"regexp"
	"time"
)

func InitMysql(cfg MySQLConfig) (*gorm.DB, error) {
	if !regexp.MustCompile(`^suxu_[a-z0-9_]+$`).MatchString(cfg.DBName) {
		return nil, fmt.Errorf("仅允许连接 suxu_ 前缀的独立商城库，禁止对参考项目原库执行迁移")
	}
	c := driver.NewConfig()
	c.User = cfg.User
	c.Passwd = cfg.Password
	c.Net = "tcp"
	c.Addr = net.JoinHostPort(cfg.Host, cfg.Port)
	c.DBName = cfg.DBName
	c.ParseTime = true
	c.Loc = time.Local
	c.Timeout = 5 * time.Second
	c.ReadTimeout = 15 * time.Second
	c.WriteTimeout = 15 * time.Second
	c.Params = map[string]string{"charset": cfg.Charset}
	if cfg.AutoCreateDatabase {
		if !regexp.MustCompile(`^suxu_[a-z0-9_]+$`).MatchString(cfg.DBName) {
			return nil, fmt.Errorf("自动建库仅允许 suxu_ 前缀的独立数据库")
		}
		server := *c
		server.DBName = ""
		raw, e := sql.Open("mysql", server.FormatDSN())
		if e != nil {
			return nil, e
		}
		defer raw.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, e = raw.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+cfg.DBName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); e != nil {
			return nil, fmt.Errorf("创建独立数据库失败: %w", e)
		}
	}
	db, e := gorm.Open(mysql.Open(c.FormatDSN()), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true, TablePrefix: "sys_"}, Logger: logger.Default.LogMode(logger.Silent), SkipDefaultTransaction: true, PrepareStmt: true})
	if e != nil {
		return nil, fmt.Errorf("连接MySQL失败，请检查配置文件中的 mysql 段: %w", e)
	}
	pool, e := db.DB()
	if e != nil {
		return nil, e
	}
	pool.SetMaxIdleConns(cfg.MaxIdle)
	pool.SetMaxOpenConns(cfg.MaxOpen)
	pool.SetConnMaxLifetime(time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e = pool.PingContext(ctx); e != nil {
		return nil, e
	}
	if e = Migrate(db); e != nil {
		return nil, e
	}
	if global.Logger != nil {
		global.Logger.Infow("MySQL连接与业务表迁移完成", "module", "config/mysql", "op", "InitMysql", "database", cfg.DBName)
	}
	return db, nil
}
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&m.Category{}, &m.Product{}, &m.ProductSKU{}, &m.ProductImage{}, &m.ShopSetting{}, &m.Order{}, &m.OrderItem{}, &m.StockMovement{}, &m.AdminUser{}, &m.AdminSession{}, &m.AdminAuditLog{}, &m.MigrationRecord{}, &m.LoginAttempt{}, &m.CustomerAccount{}, &m.CustomerSession{})
}
