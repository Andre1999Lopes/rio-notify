package notification

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"rio-notify/internal/logger"
)

type NotificationHandler struct {
	service *NotificationService
	logger  *logger.Logger
}

func NewNotificationHandler(service *NotificationService, log *logger.Logger) *NotificationHandler {
	return &NotificationHandler{
		service: service,
		logger:  log,
	}
}

func (handler *NotificationHandler) ListNotifications(c *gin.Context) {
	userHash := c.GetString("user_hash")
	page, _ := strconv.Atoi(c.DefaultQuery("pagina", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limite", "20"))
	result, err := handler.service.ListNotifications(c.Request.Context(), userHash, page, limit)

	if err != nil {
		handler.logger.Error("Falha ao listar notificações",
			"user_hash", userHash[:8]+"...",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"erro": "Falha ao buscar notificações",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (handler *NotificationHandler) MarkAsRead(c *gin.Context) {
	userHash := c.GetString("user_hash")
	notificationId := c.Param("id")
	updated, err := handler.service.MarkAsRead(c.Request.Context(), notificationId, userHash)

	if err != nil {
		handler.logger.Error("Falha ao marcar notificação como lida",
			"id", notificationId,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"erro": "Falha ao marcar notificação como lida",
		})
		return
	}

	if !updated {
		c.JSON(http.StatusNotFound, gin.H{
			"erro": "Notificação não encontrada",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mensagem": "Notificação marcada como lida",
	})
}

func (handler *NotificationHandler) CountUnread(c *gin.Context) {
	userHash := c.GetString("user_hash")
	count, err := handler.service.CountUnread(c.Request.Context(), userHash)

	if err != nil {
		handler.logger.Error("Falha ao contar notificações não lidas",
			"user_hash", userHash[:8]+"...",
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"erro": "Falha ao contar notificações",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nao_lidas": count,
	})
}
