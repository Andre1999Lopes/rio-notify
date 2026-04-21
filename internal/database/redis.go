package database

import (
	"context"
	"errors"
	"rio-notify/internal/logger"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	*redis.Client
	logger *logger.Logger
}

type RedisConfig struct {
	PingTimeout time.Duration
}

func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		PingTimeout: 5 * time.Second,
	}
}

func NewRedisClient(ctx context.Context, redisURL string, logger *logger.Logger) (*RedisClient, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Error("Falha ao fazer parse da URL do Redis",
			"error", err,
			"url", redisURL,
		)
		return nil, errors.New("failed to parse redis URL")
	}

	client := redis.NewClient(opts)

	config := DefaultRedisConfig()
	pingCtx, cancel := context.WithTimeout(ctx, config.PingTimeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		client.Close()
		logger.Error("Falha ao pingar Redis",
			"error", err,
			"timeout", config.PingTimeout,
		)
		return nil, errors.New("failed to ping redis")
	}

	logger.Info("Redis conectado com sucesso",
		"addr", opts.Addr,
		"db", opts.DB,
	)

	return &RedisClient{
		Client: client,
		logger: logger,
	}, nil
}

func (r *RedisClient) Close() error {
	if err := r.Client.Close(); err != nil {
		r.logger.Error("Falha ao fechar conexão com Redis", "error", err)
		return err
	}
	r.logger.Info("Conexão Redis fechada")
	return nil
}

func (r *RedisClient) Health(ctx context.Context) error {
	config := DefaultRedisConfig()
	pingCtx, cancel := context.WithTimeout(ctx, config.PingTimeout)
	defer cancel()

	if err := r.Client.Ping(pingCtx).Err(); err != nil {
		r.logger.Error("Health check do Redis falhou", "error", err)
		return errors.New("redis health check failed")
	}
	return nil
}

func (r *RedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	if err := r.Client.Publish(ctx, channel, message).Err(); err != nil {
		r.logger.Error("Falha ao publicar mensagem no Redis",
			"channel", channel,
			"error", err,
		)
		return err
	}
	r.logger.Debug("Mensagem publicada no Redis",
		"channel", channel,
	)
	return nil
}

func (r *RedisClient) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	sub := r.Client.Subscribe(ctx, channel)
	r.logger.Debug("Inscrito no canal Redis",
		"channel", channel,
	)
	return sub
}

func (r *RedisClient) Unsubscribe(ctx context.Context, pubsub *redis.PubSub, channel string) error {
	if err := pubsub.Unsubscribe(ctx, channel); err != nil {
		r.logger.Error("Falha ao cancelar assinatura do canal Redis",
			"channel", channel,
			"error", err,
		)
		return err
	}
	r.logger.Debug("Assinatura cancelada",
		"channel", channel,
	)
	return nil
}

func (r *RedisClient) SetNXWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	ok, err := r.Client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		r.logger.Error("Falha ao executar SetNX no Redis",
			"key", key,
			"error", err,
		)
		return false, err
	}
	if ok {
		r.logger.Debug("Chave definida no Redis",
			"key", key,
			"ttl", ttl,
		)
	}
	return ok, nil
}

func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.Client.Exists(ctx, key).Result()
	if err != nil {
		r.logger.Error("Falha ao verificar existência de chave no Redis",
			"key", key,
			"error", err,
		)
		return false, err
	}
	return count > 0, nil
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		r.logger.Error("Falha ao obter valor do Redis",
			"key", key,
			"error", err,
		)
		return "", err
	}
	return val, nil
}

func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if err := r.Client.Set(ctx, key, value, ttl).Err(); err != nil {
		r.logger.Error("Falha ao definir valor no Redis",
			"key", key,
			"error", err,
		)
		return err
	}
	r.logger.Debug("Valor definido no Redis",
		"key", key,
		"ttl", ttl,
	)
	return nil
}

func (r *RedisClient) Del(ctx context.Context, key string) error {
	if err := r.Client.Del(ctx, key).Err(); err != nil {
		r.logger.Error("Falha ao remover chave do Redis",
			"key", key,
			"error", err,
		)
		return err
	}
	r.logger.Debug("Chave removida do Redis",
		"key", key,
	)
	return nil
}
