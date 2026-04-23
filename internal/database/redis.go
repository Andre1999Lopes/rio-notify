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

func (redisClient *RedisClient) Close() error {
	if err := redisClient.Client.Close(); err != nil {
		redisClient.logger.Error("Falha ao fechar conexão com Redis", "error", err)
		return err
	}

	redisClient.logger.Info("Conexão Redis fechada")
	return nil
}

func (redisClient *RedisClient) Health(ctx context.Context) error {
	config := DefaultRedisConfig()
	pingCtx, cancel := context.WithTimeout(ctx, config.PingTimeout)
	defer cancel()

	if err := redisClient.Client.Ping(pingCtx).Err(); err != nil {
		redisClient.logger.Error("Health check do Redis falhou", "error", err)
		return errors.New("redis health check failed")
	}

	return nil
}

func (redisClient *RedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	if err := redisClient.Client.Publish(ctx, channel, message).Err(); err != nil {
		redisClient.logger.Error("Falha ao publicar mensagem no Redis",
			"channel", channel,
			"error", err,
		)
		return err
	}

	redisClient.logger.Debug("Mensagem publicada no Redis",
		"channel", channel,
	)
	return nil
}

func (redisClient *RedisClient) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	sub := redisClient.Client.Subscribe(ctx, channel)
	redisClient.logger.Debug("Inscrito no canal Redis",
		"channel", channel,
	)
	return sub
}

func (redisClient *RedisClient) Unsubscribe(ctx context.Context, pubsub *redis.PubSub, channel string) error {
	if err := pubsub.Unsubscribe(ctx, channel); err != nil {
		redisClient.logger.Error("Falha ao cancelar assinatura do canal Redis",
			"channel", channel,
			"error", err,
		)
		return err
	}

	redisClient.logger.Debug("Assinatura cancelada",
		"channel", channel,
	)
	return nil
}

func (redisClient *RedisClient) SetNXWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	result, err := redisClient.Client.SetArgs(ctx, key, value, redis.SetArgs{
		TTL:  ttl,
		Mode: "NX",
	}).Result()

	if err != nil {
		redisClient.logger.Error("Falha ao executar SetNX no Redis",
			"key", key,
			"error", err,
		)
		return false, err
	}

	ok := result == "OK"

	if ok {
		redisClient.logger.Debug("Chave definida no Redis",
			"key", key,
			"ttl", ttl,
		)
	}

	return ok, nil
}

func (redisClient *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	count, err := redisClient.Client.Exists(ctx, key).Result()

	if err != nil {
		redisClient.logger.Error("Falha ao verificar existência de chave no Redis",
			"key", key,
			"error", err,
		)
		return false, err
	}

	return count > 0, nil
}

func (redisClient *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := redisClient.Client.Get(ctx, key).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		redisClient.logger.Error("Falha ao obter valor do Redis",
			"key", key,
			"error", err,
		)
		return "", err
	}

	return val, nil
}

func (redisClient *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if err := redisClient.Client.Set(ctx, key, value, ttl).Err(); err != nil {
		redisClient.logger.Error("Falha ao definir valor no Redis",
			"key", key,
			"error", err,
		)
		return err
	}
	redisClient.logger.Debug("Valor definido no Redis",
		"key", key,
		"ttl", ttl,
	)
	return nil
}

func (redisClient *RedisClient) Del(ctx context.Context, key string) error {
	if err := redisClient.Client.Del(ctx, key).Err(); err != nil {
		redisClient.logger.Error("Falha ao remover chave do Redis",
			"key", key,
			"error", err,
		)
		return err
	}
	redisClient.logger.Debug("Chave removida do Redis",
		"key", key,
	)
	return nil
}
