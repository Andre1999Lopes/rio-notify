package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"rio-notify/internal/crypto"
	"rio-notify/internal/database"
	"rio-notify/internal/logger"
)

type WebhookService struct {
	repository *WebhookRepository
	redis      *database.RedisClient
	hasher     *crypto.Hasher
	logger     *logger.Logger
}

func NewWebhookService(
	repo *WebhookRepository,
	redis *database.RedisClient,
	hasher *crypto.Hasher,
	log *logger.Logger,
) *WebhookService {
	return &WebhookService{
		repository: repo,
		redis:      redis,
		hasher:     hasher,
		logger:     log,
	}
}

func (service *WebhookService) IsDuplicate(ctx context.Context, callId, statusNew string) bool {
	key := service.buildIdempotencyKey(callId, statusNew)
	service.logger.Info("🔍 Verificando idempotência",
		"key", key,
	)
	exists, err := service.redis.Exists(ctx, key)

	if err != nil {
		service.logger.Warn("Falha ao verificar idempotência no Redis",
			"key", key,
			"error", err,
		)
		return false
	}

	service.logger.Info("🔍 Resultado Exists",
		"key", key,
		"exists", exists,
	)

	return exists
}

func (service *WebhookService) MarkAsProcessed(ctx context.Context, callId, statusNew string) {
	key := service.buildIdempotencyKey(callId, statusNew)

	if err := service.redis.Set(ctx, key, "1", 5*time.Minute); err != nil {
		service.logger.Warn("Falha ao marcar evento no Redis",
			"key", key,
			"error", err,
		)
	}
}

func (service *WebhookService) ProcessWebhook(ctx context.Context, payload WebhookPayload) (string, error) {
	userHash := service.hasher.HashCpf(payload.Cpf)
	params := CreateNotificationParams{
		UserHash:       userHash,
		CallId:         payload.ChamadoId,
		Title:          payload.Titulo,
		Description:    payload.Descricao,
		StatusOld:      payload.StatusAnterior,
		StatusNew:      payload.StatusNovo,
		EventTimestamp: payload.Timestamp,
	}
	return service.repository.Create(ctx, params)
}

func (service *WebhookService) buildIdempotencyKey(callId, statusNew string) string {
	return fmt.Sprintf("event:%s:%s", callId, statusNew)
}

func (service *WebhookService) PublishToRedis(ctx context.Context, notificationId string, payload WebhookPayload) {
	publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	userHash := service.hasher.HashCpf(payload.Cpf)
	channel := fmt.Sprintf("user:%s", userHash)

	notificationJson, err := json.Marshal(map[string]any{
		"id":              notificationId,
		"chamado_id":      payload.ChamadoId,
		"titulo":          payload.Titulo,
		"descricao":       payload.Descricao,
		"status_anterior": payload.StatusAnterior,
		"status_novo":     payload.StatusNovo,
		"data_evento":     payload.Timestamp,
		"lida":            false,
	})

	if err != nil {
		service.logger.Error("Falha ao serializar notificação para Redis",
			"error", err,
		)
		return
	}

	if err := service.redis.Publish(publishCtx, channel, notificationJson); err != nil {
		service.logger.Error("Falha ao publicar no Redis Pub/Sub",
			"channel", channel,
			"error", err,
		)
	}
}
