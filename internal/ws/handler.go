package ws

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"rio-notify/internal/crypto"
	"rio-notify/internal/logger"
	"rio-notify/internal/middleware"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHandler struct {
	hub       *Hub
	hasher    *crypto.Hasher
	jwtSecret string
	logger    *logger.Logger
}

func NewWebSocketHandler(hub *Hub, hasher *crypto.Hasher, jwtSecret string, log *logger.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:       hub,
		hasher:    hasher,
		jwtSecret: jwtSecret,
		logger:    log,
	}
}

func (handler *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	token := c.Query("token")

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"erro": "Token JWT é obrigatório como query param (?token=...)",
		})
		return
	}

	cpf, err := middleware.ValidateJwtToken(token, handler.jwtSecret, handler.hasher)

	if err != nil {
		handler.logger.Warn("Token WebSocket inválido",
			"error", err,
			"client_ip", c.ClientIP(),
		)
		c.JSON(http.StatusUnauthorized, gin.H{
			"erro": "Token inválido ou expirado",
		})
		return
	}

	userHash := handler.hasher.HashCpf(cpf)
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)

	if err != nil {
		handler.logger.Error("Falha ao fazer upgrade WebSocket",
			"error", err,
			"user_hash", userHash[:8]+"...",
		)
		return
	}

	client := &Client{
		hub:      handler.hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userHash: userHash,
		logger:   handler.logger,
	}
	handler.hub.register <- client
	handler.logger.Info("Conexão WebSocket estabelecida",
		"user_hash", userHash[:8]+"...",
		"client_ip", c.ClientIP(),
	)

	go client.writePump()
	go client.readPump()
}
