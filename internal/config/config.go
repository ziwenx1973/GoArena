package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct{ Port, MySQLHost, MySQLPort, MySQLUser, MySQLPassword, MySQLDatabase, RedisHost, RedisPort, RedisPassword, JWTSecret string }

func Load() (Config, error) {
	c := Config{env("SERVER_PORT", "8080"), env("MYSQL_HOST", "127.0.0.1"), env("MYSQL_PORT", "3306"), env("MYSQL_USER", "arena"), os.Getenv("MYSQL_PASSWORD"), env("MYSQL_DATABASE", "go_arena"), env("REDIS_HOST", "127.0.0.1"), env("REDIS_PORT", "6379"), os.Getenv("REDIS_PASSWORD"), os.Getenv("JWT_SECRET")}
	if len(c.JWTSecret) < 32 {
		return c, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	for _, p := range []string{c.Port, c.MySQLPort, c.RedisPort} {
		n, e := strconv.Atoi(p)
		if e != nil || n < 1 || n > 65535 {
			return c, fmt.Errorf("invalid port: %s", p)
		}
	}
	return c, nil
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
