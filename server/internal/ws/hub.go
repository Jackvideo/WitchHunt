package ws

import (
	"encoding/json"
	"log"
)

type IncomingMessage struct {
	Client *Client
	Data   []byte
}

type Hub struct {
	Clients    map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
	Incoming   chan *IncomingMessage
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Incoming:   make(chan *IncomingMessage, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			h.broadcastToRoom(client.RoomCode, Message{
				Type:    "player_join",
				Payload: mustJSON(map[string]any{"user_id": client.UserID, "username": client.Username}),
			})
			log.Printf("client registered: user=%s room=%s", client.Username, client.RoomCode)

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				h.broadcastToRoom(client.RoomCode, Message{
					Type:    "player_leave",
					Payload: mustJSON(map[string]any{"user_id": client.UserID, "username": client.Username}),
				})
				log.Printf("client unregistered: user=%s room=%s", client.Username, client.RoomCode)
			}

		case incoming := <-h.Incoming:
			var msg Message
			if err := json.Unmarshal(incoming.Data, &msg); err != nil {
				log.Printf("invalid message from user=%s: %v", incoming.Client.Username, err)
				continue
			}
			h.handleMessage(incoming.Client, msg)
		}
	}
}

func (h *Hub) handleMessage(sender *Client, msg Message) {
	switch msg.Type {
	case "game_action":
		// TODO: forward to GameEngine, then broadcast result
		h.broadcastToRoom(sender.RoomCode, Message{
			Type: "game_action",
			Payload: mustJSON(map[string]any{
				"user_id":  sender.UserID,
				"username": sender.Username,
				"action":   msg.Payload,
			}),
		})
	default:
		log.Printf("unknown message type: %s", msg.Type)
	}
}

func (h *Hub) broadcastToRoom(roomCode string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	for client := range h.Clients {
		if client.RoomCode == roomCode {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(h.Clients, client)
			}
		}
	}
}

func (h *Hub) SendTo(userID uint, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	for client := range h.Clients {
		if client.UserID == userID {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(h.Clients, client)
			}
		}
	}
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
