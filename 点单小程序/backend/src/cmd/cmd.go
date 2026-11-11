package cmd

import (
	"Boutique/src/app"
	"Boutique/src/config"
	"Boutique/src/dao"
	"Boutique/src/global"
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Start() error {
	global.Logger = config.InitLogger()
	cfg, e := config.LoadConfig("src/config/config.yaml")
	if e != nil {
		global.Logger.Errorw("配置加载失败", "module", "cmd", "op", "Start", "err", e)
		return e
	}
	global.Logger = config.InitLogger(cfg.Logger)
	global.DB, e = config.InitMysql(cfg.MySQL)
	if e != nil {
		global.Logger.Errorw("数据库初始化失败", "module", "cmd", "op", "Start", "err", e)
		return e
	}
	global.Redis, e = config.InitRedis(cfg.Redis)
	if e != nil {
		global.Logger.Errorw("Redis初始化失败", "module", "cmd", "op", "Start", "err", e)
		return e
	}
	if e = dao.Bootstrap(context.Background(), global.DB, cfg.Bootstrap.AdminUsername, cfg.Bootstrap.AdminPassword, cfg.Bootstrap.SeedCatalog); e != nil {
		global.Logger.Errorw("初始数据校验失败", "module", "cmd", "op", "Start", "err", e)
		return e
	}
	runtime, e := app.New(cfg, app.Dependencies{DB: global.DB, Redis: global.Redis})
	if e != nil {
		global.Logger.Errorw("应用组装失败", "module", "cmd", "op", "Start", "err", e)
		return e
	}
	server := &http.Server{Addr: net.JoinHostPort(cfg.Server.Host, cfg.Server.Port), Handler: runtime.Handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: time.Duration(cfg.Server.Timeout) * time.Second, WriteTimeout: time.Duration(cfg.Server.Timeout) * time.Second, IdleTimeout: 60 * time.Second}
	errorsCh := make(chan error, 1)
	go func() {
		global.Logger.Infow("商城API服务已启动", "module", "cmd", "op", "Start", "address", server.Addr)
		errorsCh <- server.ListenAndServe()
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if e = server.Shutdown(shutdown); e != nil {
			global.Logger.Errorw("服务关闭失败", "module", "cmd", "op", "Start", "err", e)
		}
	case e = <-errorsCh:
		if !errors.Is(e, http.ErrServerClosed) {
			global.Logger.Errorw("服务监听失败", "module", "cmd", "op", "Start", "err", e)
			return e
		}
	}
	return nil
}
func Clean() {
	if global.Redis != nil {
		_ = global.Redis.Close()
	}
	if global.DB != nil {
		if db, e := global.DB.DB(); e == nil {
			_ = db.Close()
		}
	}
	if global.Logger != nil {
		_ = global.Logger.Sync()
	}
}
