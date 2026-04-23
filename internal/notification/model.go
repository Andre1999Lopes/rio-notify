package notification

import (
	"time"
)

type Notification struct {
	Id             string     `json:"id"`
	UserHash       string     `json:"-"`
	CallId         string     `json:"call_id"`
	Title          string     `json:"titulo"`
	Description    string     `json:"descricao"`
	StatusOld      *string    `json:"status_anterior,omitempty"`
	StatusNew      string     `json:"status_novo"`
	EventTimestamp time.Time  `json:"data_evento"`
	Read           bool       `json:"lida"`
	CreatedAt      time.Time  `json:"criada_em"`
	UpdatedAt      *time.Time `json:"atualizada_em,omitempty"`
}

type NotificationListResponse struct {
	Notifications []Notification `json:"notificacoes"`
	Page          int            `json:"pagina"`
	Limit         int            `json:"limite"`
	Total         int            `json:"total"`
}

type UnreadCountResponse struct {
	Count int `json:"nao_lidas"`
}
