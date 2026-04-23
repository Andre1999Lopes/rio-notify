package webhook

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"rio-notify/internal/database"
	"rio-notify/internal/logger"
)

type WebhookRepository struct {
	db     *database.PostgresDB
	logger *logger.Logger
}

func NewWebhookRepository(db *database.PostgresDB, log *logger.Logger) *WebhookRepository {
	return &WebhookRepository{
		db:     db,
		logger: log,
	}
}

var ErrDuplicateEvent = errors.New("duplicate event")

type CreateNotificationParams struct {
	UserHash       string
	CallId         string
	Title          string
	Description    string
	StatusOld      *string
	StatusNew      string
	EventTimestamp time.Time
}

func (repository *WebhookRepository) Create(ctx context.Context, params CreateNotificationParams) (string, error) {
	query := `
		INSERT INTO notifications (
			user_hash, call_id, title, description,
			status_old, status_new, event_timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	var id string
	err := repository.db.QueryRow(ctx, query,
		params.UserHash,
		params.CallId,
		params.Title,
		params.Description,
		params.StatusOld,
		params.StatusNew,
		params.EventTimestamp,
	).Scan(&id)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			repository.logger.Debug("Evento duplicado detectado",
				"call_id", params.CallId,
				"status_new", params.StatusNew,
			)
			return "", ErrDuplicateEvent
		}
		repository.logger.Error("Falha ao inserir notificação",
			"error", err,
			"call_id", params.CallId,
		)
		return "", err
	}

	repository.logger.Debug("Notificação criada com sucesso",
		"call_id", params.CallId,
		"status_new", params.StatusNew,
	)
	return id, nil
}
