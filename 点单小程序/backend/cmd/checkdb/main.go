package main

import (
	"Boutique/src/config"
	"Boutique/src/global"
)

func main() {
	global.Logger = config.InitLogger()
	defer global.Logger.Sync()
	cfg, e := config.LoadConfig("src/config/config.yaml")
	if e != nil {
		global.Logger.Fatalw("配置失败", "err", e)
	}
	db, e := config.InitMysql(cfg.MySQL)
	if e != nil {
		global.Logger.Fatalw("数据库验证失败", "err", e)
	}
	var count int64
	e = db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", cfg.MySQL.DBName).Scan(&count).Error
	if e != nil {
		global.Logger.Fatalw("表结构验证失败", "err", e)
	}
	global.Logger.Infow("业务表验证完成", "database", cfg.MySQL.DBName, "table_count", count)
}
