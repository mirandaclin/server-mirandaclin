package cache

import (
	"context"
	"time"

	"github.com/Mirnda/mirandaclin/pkg/config"
	"github.com/Mirnda/mirandaclin/pkg/logger"
)

// Cache é a interface de acesso ao cache — usada por todos os domínios.
// Nunca importar redis diretamente fora deste pacote.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	DelWithIndex(ctx context.Context, prefix string) ([]string, error)
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

func New(log logger.Logger, rds config.Redis) Cache {

	if rds.Addr == "" {
		log.Debug("Redis não configurado — usando Noop cache")
		return NewNoop()
	}

	cache, err := NewRedis(rds.Addr, rds.Password, rds.DB)
	if err != nil {
		log.WithErr(err).Warn("Redis indisponível — usando Noop cache")
		return NewNoop()
	}

	return cache
}
