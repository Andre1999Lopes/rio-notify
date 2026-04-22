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
	CallID         string
	Title          string
	Description    string
	StatusOld      *string
	StatusNew      string
	EventTimestamp time.Time
}

func (r *WebhookRepository) Create(ctx context.Context, params CreateNotificationParams) error {
	query := `
		INSERT INTO notifications (
			user_hash, call_id, title, description,
			status_old, status_new, event_timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query,
		params.UserHash,
		params.CallID,
		params.Title,
		params.Description,
		params.StatusOld,
		params.StatusNew,
		params.EventTimestamp,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			r.logger.Debug("Evento duplicado detectado",
				"call_id", params.CallID,
				"status_new", params.StatusNew,
			)
			return ErrDuplicateEvent
		}
		r.logger.Error("Falha ao inserir notificação",
			"error", err,
			"call_id", params.CallID,
		)
		return err
	}

	r.logger.Debug("Notificação criada com sucesso",
		"call_id", params.CallID,
		"status_new", params.StatusNew,
	)

	return nil
}
