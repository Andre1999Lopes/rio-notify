package notification

import (
	"context"

	"rio-notify/internal/logger"
)

type NotificationService struct {
	repository *NotificationRepository
	logger     *logger.Logger
}

func NewNotificationService(repo *NotificationRepository, log *logger.Logger) *NotificationService {
	return &NotificationService{
		repository: repo,
		logger:     log,
	}
}

func (service *NotificationService) ListNotifications(
	ctx context.Context,
	userHash string,
	page, limit int,
) (*NotificationListResponse, error) {
	if page < 1 {
		page = 1
	}

	if limit < 1 || limit > 100 {
		limit = 20
	}

	notifications, total, err := service.repository.FindByUserHash(ctx, userHash, page, limit)

	if err != nil {
		service.logger.Error("Falha ao listar notificações",
			"user_hash", userHash[:8]+"...",
			"error", err,
		)
		return nil, err
	}

	return &NotificationListResponse{
		Notifications: notifications,
		Page:          page,
		Limit:         limit,
		Total:         total,
	}, nil
}

func (service *NotificationService) MarkAsRead(ctx context.Context, id, userHash string) (bool, error) {
	updated, err := service.repository.MarkAsRead(ctx, id, userHash)

	if err != nil {
		service.logger.Error("Falha ao marcar notificação como lida",
			"id", id,
			"error", err,
		)
		return false, err
	}

	return updated, nil
}

func (service *NotificationService) CountUnread(ctx context.Context, userHash string) (int, error) {
	count, err := service.repository.CountUnread(ctx, userHash)

	if err != nil {
		service.logger.Error("Falha ao contar notificações não lidas",
			"user_hash", userHash[:8]+"...",
			"error", err,
		)
		return 0, err
	}

	return count, nil
}
