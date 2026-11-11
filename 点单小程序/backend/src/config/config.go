package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

type MySQLConfig struct {
	Host               string `mapstructure:"host"`
	Port               string `mapstructure:"port"`
	User               string `mapstructure:"user"`
	Password           string `mapstructure:"password"`
	DBName             string `mapstructure:"dbname"`
	Charset            string `mapstructure:"charset"`
	MaxIdle            int    `mapstructure:"max_idle"`
	MaxOpen            int    `mapstructure:"max_open"`
	AutoCreateDatabase bool   `mapstructure:"auto_create_database"`
}
type RedisConfig struct {
	Enable   bool   `mapstructure:"enable"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}
type LoggerConfig struct {
	MaxSize    int `mapstructure:"MaxSize"`
	MaxBackups int `mapstructure:"MaxBackups"`
	MaxAge     int `mapstructure:"MaxAge"`
}
type ServerConfig struct {
	Host         string   `mapstructure:"host"`
	Port         string   `mapstructure:"port"`
	Mode         string   `mapstructure:"mode"`
	Timeout      int      `mapstructure:"timeout"`
	AllowOrigins []string `mapstructure:"allow_origins"`
	AllowOrigin  string   `mapstructure:"allow_origin"`
	AllowMethods string   `mapstructure:"allow_methods"`
	AllowHeaders string   `mapstructure:"allow_headers"`
	CorsMaxAge   int      `mapstructure:"cors_max_age"`
}
type JWTConfig struct {
	Secret    string   `mapstructure:"secret"`
	Expire    int      `mapstructure:"expire"`
	Issuer    string   `mapstructure:"issuer"`
	Whitelist []string `mapstructure:"whitelist"`
}
type BootstrapConfig struct {
	AdminUsername string `mapstructure:"admin_username"`
	AdminPassword string `mapstructure:"admin_password"`
	SeedCatalog   bool   `mapstructure:"seed_catalog"`
}
type RateLimitConfig struct {
	Enable   bool    `mapstructure:"enable"`
	Rate     float64 `mapstructure:"rate"`
	Capacity int     `mapstructure:"capacity"`
}
type AppConfig struct {
	Logger    LoggerConfig    `mapstructure:"logger"`
	Server    ServerConfig    `mapstructure:"server"`
	MySQL     MySQLConfig     `mapstructure:"mysql"`
	Redis     RedisConfig     `mapstructure:"redis"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Bootstrap BootstrapConfig `mapstructure:"bootstrap"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

func LoadConfig(path string) (*AppConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	local := filepath.Join(filepath.Dir(path), "config.local.yaml")
	if _, err := os.Stat(local); err == nil {
		v.SetConfigFile(local)
		if err = v.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("读取本地配置失败: %w", err)
		}
	}
	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	if cfg.MySQL.Host == "" || cfg.MySQL.DBName == "" || cfg.MySQL.User == "" {
		return nil, fmt.Errorf("请在 src/config/config.yaml 或 config.local.yaml 配置 MySQL")
	}
	if len(cfg.JWT.Secret) < 32 {
		return nil, fmt.Errorf("请在配置文件设置至少32字符的 jwt.secret")
	}
	if cfg.JWT.Expire < 1 || cfg.JWT.Expire > 168 {
		return nil, fmt.Errorf("jwt.expire 应为1至168小时")
	}
	if cfg.Server.Port == "" {
		return nil, fmt.Errorf("server.port 不能为空")
	}
	if cfg.Server.Mode != "debug" && cfg.Server.Mode != "release" {
		return nil, fmt.Errorf("server.mode 应为debug或release")
	}
	if cfg.Server.Mode == "release" && cfg.Server.AllowOrigin == "*" {
		return nil, fmt.Errorf("发布模式请显式配置跨域来源")
	}
	if cfg.Server.Mode == "release" && (strings.Contains(cfg.JWT.Secret, "change-me") || cfg.Bootstrap.AdminPassword == "stall2026!") {
		return nil, fmt.Errorf("发布前请替换开发密钥及初始密码")
	}
	if cfg.Server.Timeout < 1 || cfg.Server.Timeout > 120 {
		return nil, fmt.Errorf("server.timeout 应为1至120秒")
	}
	if cfg.MySQL.Port == "" || cfg.MySQL.Charset != "utf8mb4" || cfg.MySQL.MaxIdle < 0 || cfg.MySQL.MaxOpen < 1 || cfg.MySQL.MaxIdle > cfg.MySQL.MaxOpen {
		return nil, fmt.Errorf("MySQL 端口、字符集或连接池配置无效")
	}
	if cfg.RateLimit.Enable && (cfg.RateLimit.Rate <= 0 || cfg.RateLimit.Capacity < 1) {
		return nil, fmt.Errorf("限流速率和容量必须大于零")
	}
	allowed := map[string]bool{"/api/v1/catalog": true, "/api/v1/orders": true, "/api/v1/admin/login": true}
	for _, path := range cfg.JWT.Whitelist {
		if !allowed[path] {
			return nil, fmt.Errorf("jwt.whitelist 不能开放未登记的接口或管理组")
		}
	}
	return &cfg, nil
}
