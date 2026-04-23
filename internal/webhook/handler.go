package webhook

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"rio-notify/internal/logger"
)

type WebhookHandler struct {
	service *WebhookService
	logger  *logger.Logger
}

func NewWebhookHandler(service *WebhookService, log *logger.Logger) *WebhookHandler {
	return &WebhookHandler{
		service: service,
		logger:  log,
	}
}

func (handler *WebhookHandler) HandleWebhook(c *gin.Context) {
	ctx := c.Request.Context()
	var payload WebhookPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		handler.logger.Warn("Payload inválido",
			"error", err,
			"client_ip", c.ClientIP(),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Payload inválido",
		})
		return
	}

	handler.logger.Info("Webhook recebido",
		"call_id", payload.ChamadoId,
		"status_new", payload.StatusNovo,
		"cpf_masked", maskCPF(payload.Cpf),
	)

	if handler.service.IsDuplicate(ctx, payload.ChamadoId, payload.StatusNovo) {
		handler.logger.Info("Evento já processado (Redis)",
			"call_id", payload.ChamadoId,
			"status_new", payload.StatusNovo,
		)
		c.JSON(http.StatusOK, gin.H{
			"status": "processado",
		})
		return
	}

	notificationId, err := handler.service.ProcessWebhook(ctx, payload)

	if err != nil && errors.Is(err, ErrDuplicateEvent) {
		handler.service.MarkAsProcessed(ctx, payload.ChamadoId, payload.StatusNovo)

		handler.logger.Info("Evento já processado (banco)",
			"call_id", payload.ChamadoId,
			"status_new", payload.StatusNovo,
		)
		c.JSON(http.StatusOK, gin.H{
			"status": "processado",
		})
		return
	}

	if err != nil {
		handler.logger.Error("Falha ao processar webhook",
			"error", err,
			"call_id", payload.ChamadoId,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Falha ao processar webhook",
		})
		return
	}

	handler.service.MarkAsProcessed(ctx, payload.ChamadoId, payload.StatusNovo)
	go handler.service.PublishToRedis(ctx, notificationId, payload)
	handler.logger.Info("Webhook processado com sucesso",
		"call_id", payload.ChamadoId,
		"status_new", payload.StatusNovo,
	)
	c.JSON(http.StatusCreated, gin.H{
		"status": "criado",
	})
}

func maskCPF(cpf string) string {
	if len(cpf) < 4 {
		return "***"
	}
	return "***" + cpf[len(cpf)-4:]
}
