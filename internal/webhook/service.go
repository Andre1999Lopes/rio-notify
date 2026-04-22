package webhook

import (
	"context"
	"fmt"
	"time"

	"rio-notify/internal/crypto"
	"rio-notify/internal/database"
	"rio-notify/internal/logger"
)

type WebhookService struct {
	repo   *WebhookRepository
	redis  *database.RedisClient
	hasher *crypto.Hasher
	logger *logger.Logger
}

func NewWebhookService(
	repo *WebhookRepository,
	redis *database.RedisClient,
	hasher *crypto.Hasher,
	log *logger.Logger,
) *WebhookService {
	return &WebhookService{
		repo:   repo,
		redis:  redis,
		hasher: hasher,
		logger: log,
	}
}

func (s *WebhookService) IsDuplicate(ctx context.Context, callID, statusNew string) bool {
	key := s.buildIdempotencyKey(callID, statusNew)
	exists, err := s.redis.Exists(ctx, key)
	if err != nil {
		s.logger.Warn("Falha ao verificar idempotência no Redis",
			"key", key,
			"error", err,
		)
		return false
	}
	return exists
}

func (s *WebhookService) MarkAsProcessed(ctx context.Context, callID, statusNew string) {
	key := s.buildIdempotencyKey(callID, statusNew)
	if err := s.redis.Set(ctx, key, "1", 5*time.Minute); err != nil {
		s.logger.Warn("Falha ao marcar evento no Redis",
			"key", key,
			"error", err,
		)
	}
}

func (s *WebhookService) ProcessWebhook(ctx context.Context, payload WebhookPayload) error {
	userHash := s.hasher.HashCPF(payload.CPF)

	params := CreateNotificationParams{
		UserHash:       userHash,
		CallID:         payload.ChamadoID,
		Title:          payload.Titulo,
		Description:    payload.Descricao,
		StatusOld:      payload.StatusAnterior,
		StatusNew:      payload.StatusNovo,
		EventTimestamp: payload.Timestamp,
	}

	return s.repo.Create(ctx, params)
}

func (s *WebhookService) buildIdempotencyKey(callID, statusNew string) string {
	return fmt.Sprintf("event:%s:%s", callID, statusNew)
}

func (s *WebhookService) PublishToRedis(ctx context.Context, payload WebhookPayload) {
	userHash := s.hasher.HashCPF(payload.CPF)
	channel := fmt.Sprintf("user:%s", userHash)

	s.logger.Debug("Publicando no Redis Pub/Sub",
		"channel", channel,
		"call_id", payload.ChamadoID,
	)
}
