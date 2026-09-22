package database

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/ziwenx1973/GoArena/internal/config"
	"net"
	"time"
)

func OpenRedis(c config.Config) (*redis.Client, error) {
	r := redis.NewClient(&redis.Options{Addr: net.JoinHostPort(c.RedisHost, c.RedisPort), Password: c.RedisPassword, DialTimeout: 2 * time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second, MaxRetries: -1, ContextTimeoutEnabled: true})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := r.Ping(ctx).Err(); err != nil {
		r.Close()
		return nil, err
	}
	return r, nil
}
