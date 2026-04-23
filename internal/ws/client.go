package ws

import (
	"encoding/json"
	"time"

	"rio-notify/internal/logger"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userHash string
	logger   *logger.Logger
}

func (client *Client) readPump() {
	defer func() {
		client.hub.unregister <- client
		client.conn.Close()
	}()

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				client.logger.Warn("WebSocket fechado inesperadamente",
					"user_hash", client.userHash[:8]+"...",
					"error", err,
				)
			}
			break
		}

		client.logger.Debug("Mensagem recebida do cliente",
			"user_hash", client.userHash[:8]+"...",
			"message", string(message),
		)
	}
}

func (client *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				client.logger.Warn("Falha ao enviar mensagem WebSocket",
					"user_hash", client.userHash[:8]+"...",
					"error", err,
				)
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (client *Client) SendNotification(notification interface{}) error {
	data, err := json.Marshal(notification)

	if err != nil {
		return err
	}

	select {
	case client.send <- data:
		return nil
	default:
		return nil
	}
}
