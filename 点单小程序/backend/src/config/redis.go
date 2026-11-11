package config

import (
	"context"
	"github.com/redis/go-redis/v9"
	"net"
	"time"
)

func InitRedis(cfg RedisConfig) (*redis.Client, error) {
	if !cfg.Enable {
		return nil, nil
	}
	r := redis.NewClient(&redis.Options{Addr: net.JoinHostPort(cfg.Host, cfg.Port), Password: cfg.Password, DB: cfg.DB, DialTimeout: 3 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if e := r.Ping(ctx).Err(); e != nil {
		r.Close()
		return nil, e
	}
	return r, nil
}
