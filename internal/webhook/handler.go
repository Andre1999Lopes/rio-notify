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

func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.logger.Warn("Payload inválido",
			"error", err,
			"client_ip", c.ClientIP(),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Payload inválido",
		})
		return
	}

	h.logger.Info("Webhook recebido",
		"call_id", payload.ChamadoID,
		"status_new", payload.StatusNovo,
		"cpf_masked", maskCPF(payload.CPF),
	)

	if h.service.IsDuplicate(ctx, payload.ChamadoID, payload.StatusNovo) {
		h.logger.Info("Evento já processado (Redis)",
			"call_id", payload.ChamadoID,
			"status_new", payload.StatusNovo,
		)
		c.JSON(http.StatusOK, gin.H{
			"status": "processado",
		})
		return
	}

	err := h.service.ProcessWebhook(ctx, payload)

	if err != nil && errors.Is(err, ErrDuplicateEvent) {
		h.service.MarkAsProcessed(ctx, payload.ChamadoID, payload.StatusNovo)

		h.logger.Info("Evento já processado (banco)",
			"call_id", payload.ChamadoID,
			"status_new", payload.StatusNovo,
		)
		c.JSON(http.StatusOK, gin.H{
			"status": "processado",
		})
		return
	}

	if err != nil {
		h.logger.Error("Falha ao processar webhook",
			"error", err,
			"call_id", payload.ChamadoID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Falha ao processar webhook",
		})
		return
	}

	h.service.MarkAsProcessed(ctx, payload.ChamadoID, payload.StatusNovo)

	go h.service.PublishToRedis(ctx, payload)

	h.logger.Info("Webhook processado com sucesso",
		"call_id", payload.ChamadoID,
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
