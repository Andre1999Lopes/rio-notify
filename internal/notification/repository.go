package notification

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"rio-notify/internal/database"
	"rio-notify/internal/logger"
)

type NotificationRepository struct {
	db     *database.PostgresDB
	logger *logger.Logger
}

func NewNotificationRepository(db *database.PostgresDB, log *logger.Logger) *NotificationRepository {
	return &NotificationRepository{
		db:     db,
		logger: log,
	}
}

func (repository *NotificationRepository) FindByUserHash(
	ctx context.Context,
	userHash string,
	page, limit int,
) ([]Notification, int, error) {
	offset := (page - 1) * limit
	var total int
	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_hash = $1`
	err := repository.db.QueryRow(ctx, countQuery, userHash).Scan(&total)

	if err != nil {
		repository.logger.Error("Falha ao contar notificações",
			"user_hash", userHash[:8]+"...",
			"error", err,
		)
		return nil, 0, err
	}

	query := `
		SELECT 
			id, user_hash, call_id, title, description,
			status_old, status_new, event_timestamp,
			read, created_at, updated_at
		FROM notifications
		WHERE user_hash = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := repository.db.Query(ctx, query, userHash, limit, offset)

	if err != nil {
		repository.logger.Error("Falha ao buscar notificações",
			"user_hash", userHash[:8]+"...",
			"error", err,
		)
		return nil, 0, err
	}

	defer rows.Close()
	notifications := make([]Notification, 0)

	for rows.Next() {
		var n Notification
		err := rows.Scan(
			&n.Id,
			&n.UserHash,
			&n.CallId,
			&n.Title,
			&n.Description,
			&n.StatusOld,
			&n.StatusNew,
			&n.EventTimestamp,
			&n.Read,
			&n.CreatedAt,
			&n.UpdatedAt,
		)

		if err != nil {
			repository.logger.Error("Falha ao escanear notificação", "error", err)
			return nil, 0, err
		}

		notifications = append(notifications, n)
	}

	return notifications, total, nil
}

func (repository *NotificationRepository) MarkAsRead(
	ctx context.Context,
	id string,
	userHash string,
) (bool, error) {
	query := `
		UPDATE notifications
		SET read = TRUE
		WHERE id = $1 AND user_hash = $2
	`
	result, err := repository.db.Exec(ctx, query, id, userHash)

	if err != nil {
		repository.logger.Error("Falha ao marcar notificação como lida",
			"id", id,
			"user_hash", userHash[:8]+"...",
			"error", err,
		)
		return false, err
	}

	updated := result.RowsAffected() > 0

	if updated {
		repository.logger.Debug("Notificação marcada como lida",
			"id", id,
			"user_hash", userHash[:8]+"...",
		)
	}

	return updated, nil
}

func (repository *NotificationRepository) CountUnread(
	ctx context.Context,
	userHash string,
) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM notifications WHERE user_hash = $1 AND read = FALSE`
	err := repository.db.QueryRow(ctx, query, userHash).Scan(&count)

	if err != nil {
		repository.logger.Error("Falha ao contar notificações não lidas",
			"user_hash", userHash[:8]+"...",
			"error", err,
		)
		return 0, err
	}

	return count, nil
}

func (repository *NotificationRepository) FindById(
	ctx context.Context,
	id string,
	userHash string,
) (*Notification, error) {
	query := `
		SELECT 
			id, user_hash, call_id, title, description,
			status_old, status_new, event_timestamp,
			read, created_at, updated_at
		FROM notifications
		WHERE id = $1 AND user_hash = $2
	`
	var n Notification
	err := repository.db.QueryRow(ctx, query, id, userHash).Scan(
		&n.Id,
		&n.UserHash,
		&n.CallId,
		&n.Title,
		&n.Description,
		&n.StatusOld,
		&n.StatusNew,
		&n.EventTimestamp,
		&n.Read,
		&n.CreatedAt,
		&n.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, database.ErrNotFound
		}

		repository.logger.Error("Falha ao buscar notificação por ID",
			"id", id,
			"error", err,
		)
		return nil, err
	}

	return &n, nil
}

func (repository *NotificationRepository) FindRecentByUserHash(
	ctx context.Context,
	userHash string,
	since time.Time,
) ([]Notification, error) {
	query := `
		SELECT 
			id, user_hash, call_id, title, description,
			status_old, status_new, event_timestamp,
			read, created_at, updated_at
		FROM notifications
		WHERE user_hash = $1 AND created_at > $2
		ORDER BY created_at DESC
		LIMIT 10
	`
	rows, err := repository.db.Query(ctx, query, userHash, since)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	notifications := make([]Notification, 0)

	for rows.Next() {
		var n Notification
		err := rows.Scan(
			&n.Id,
			&n.UserHash,
			&n.CallId,
			&n.Title,
			&n.Description,
			&n.StatusOld,
			&n.StatusNew,
			&n.EventTimestamp,
			&n.Read,
			&n.CreatedAt,
			&n.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		notifications = append(notifications, n)
	}

	return notifications, nil
}
