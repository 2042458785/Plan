package main

import (
	"Boutique/src/config"
	"Boutique/src/dao"
	"Boutique/src/global"
	"context"
	"flag"
	"os"
)

func main() {
	file := flag.String("file", "data/boutique.json", "旧版JSON路径，导入仅允许空商城库")
	flag.Parse()
	global.Logger = config.InitLogger()
	defer global.Logger.Sync()
	cfg, e := config.LoadConfig("src/config/config.yaml")
	if e != nil {
		global.Logger.Fatalw("配置读取失败", "err", e)
	}
	db, e := config.InitMysql(cfg.MySQL)
	if e != nil {
		global.Logger.Fatalw("数据库初始化失败", "err", e)
	}
	raw, e := os.ReadFile(*file)
	if e != nil {
		global.Logger.Fatalw("旧数据读取失败", "err", e)
	}
	if e = dao.ImportLegacy(context.Background(), db, raw); e != nil {
		global.Logger.Fatalw("旧数据导入失败，事务已回滚", "err", e)
	}
	global.Logger.Infow("旧数据导入完成，原文件保留", "module", "migrate-legacy", "op", "main")
}
