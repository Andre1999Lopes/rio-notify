package ws

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"

	"rio-notify/internal/logger"
)

type Hub struct {
	clients    map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	redis      *redis.Client
	logger     *logger.Logger
}

func NewHub(redisClient *redis.Client, log *logger.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		redis:      redisClient,
		logger:     log,
	}
}

func (hub *Hub) Run() {
	hub.logger.Info("Hub WebSocket iniciado")

	for {
		select {
		case client := <-hub.register:
			hub.registerClient(client)

		case client := <-hub.unregister:
			hub.unregisterClient(client)
		}
	}
}

func (hub *Hub) registerClient(client *Client) {
	if _, ok := hub.clients[client.userHash]; !ok {
		hub.clients[client.userHash] = make(map[*Client]bool)
		go hub.SubscribeToUserChannel(context.Background(), client.userHash)
	}

	hub.clients[client.userHash][client] = true
	hub.logger.Info("Cliente WebSocket conectado",
		"user_hash", client.userHash[:8]+"...",
		"total_clients", len(hub.clients[client.userHash]),
	)
}

func (hub *Hub) unregisterClient(client *Client) {
	if clients, ok := hub.clients[client.userHash]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			close(client.send)

			if len(clients) == 0 {
				delete(hub.clients, client.userHash)
			}
		}
	}

	hub.logger.Info("Cliente WebSocket desconectado",
		"user_hash", client.userHash[:8]+"...",
	)
}

func (hub *Hub) SubscribeToUserChannel(ctx context.Context, userHash string) {
	channel := "user:" + userHash
	pubsub := hub.redis.Subscribe(ctx, channel)
	defer pubsub.Close()

	hub.logger.Info("Assinando canal Redis para WebSocket",
		"channel", channel,
	)
	ch := pubsub.Channel()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				hub.logger.Warn("Canal Redis fechado",
					"channel", channel,
				)
				return
			}

			var notification map[string]interface{}

			if err := json.Unmarshal([]byte(msg.Payload), &notification); err != nil {
				hub.logger.Warn("Mensagem Redis não é JSON válido",
					"channel", channel,
					"error", err,
				)
				continue
			}

			hub.broadcastToUser(userHash, []byte(msg.Payload))

		case <-ctx.Done():
			hub.logger.Info("Contexto cancelado, parando assinatura Redis",
				"channel", channel,
			)
			return
		}
	}
}

func (hub *Hub) broadcastToUser(userHash string, message []byte) {
	clients, ok := hub.clients[userHash]

	if !ok {
		return
	}

	for client := range clients {
		select {
		case client.send <- message:
		default:
			hub.unregister <- client
		}
	}
}

func (hub *Hub) GetConnectedClients(userHash string) int {
	if clients, ok := hub.clients[userHash]; ok {
		return len(clients)
	}

	return 0
}
