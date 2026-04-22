package webhook

import (
	"time"
)

type WebhookPayload struct {
	ChamadoID      string    `json:"chamado_id" binding:"required"`
	Tipo           string    `json:"tipo" binding:"required"`
	CPF            string    `json:"cpf" binding:"required"`
	StatusAnterior *string   `json:"status_anterior"`
	StatusNovo     string    `json:"status_novo" binding:"required"`
	Titulo         string    `json:"titulo" binding:"required"`
	Descricao      string    `json:"descricao" binding:"required"`
	Timestamp      time.Time `json:"timestamp" binding:"required"`
}
