package redisconfig

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// Options is the common Redis connection configuration used by go-redis and
// Asynq. It accepts either a redis:// URL or a plain host:port address.
type Options struct {
	Network   string
	Address   string
	Username  string
	Password  string
	DB        int
	TLSConfig *tls.Config
}

func Parse(address string) (Options, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return Options{}, fmt.Errorf("redis address is required")
	}

	if strings.Contains(address, "://") {
		parsed, err := redis.ParseURL(address)
		if err != nil {
			return Options{}, fmt.Errorf("parse redis URL: %w", err)
		}
		return Options{
			Network:   parsed.Network,
			Address:   parsed.Addr,
			Username:  parsed.Username,
			Password:  parsed.Password,
			DB:        parsed.DB,
			TLSConfig: parsed.TLSConfig,
		}, nil
	}

	return Options{
		Network: "tcp",
		Address: address,
	}, nil
}

func (options Options) GoRedis() *redis.Options {
	return &redis.Options{
		Network:   options.Network,
		Addr:      options.Address,
		Username:  options.Username,
		Password:  options.Password,
		DB:        options.DB,
		TLSConfig: options.TLSConfig,
	}
}

func (options Options) Asynq() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Network:   options.Network,
		Addr:      options.Address,
		Username:  options.Username,
		Password:  options.Password,
		DB:        options.DB,
		TLSConfig: options.TLSConfig,
	}
}
