package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/jackvidyu/witchhunt/server/internal/game"
	"github.com/jackvidyu/witchhunt/server/internal/room"
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
	Engine     *game.SalemEngine
	RoomMgr    *room.Manager
}

func NewHub(engine *game.SalemEngine, rm *room.Manager) *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Incoming:   make(chan *IncomingMessage, 256),
		Engine:     engine,
		RoomMgr:    rm,
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

			if view := h.Engine.GetViewForPlayer(client.RoomCode, client.UserID); view != nil {
				h.sendToClient(client, Message{Type: "state_update", Payload: mustJSON(view)})
			}

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				h.broadcastToRoom(client.RoomCode, Message{
					Type:    "player_leave",
					Payload: mustJSON(map[string]any{"user_id": client.UserID, "username": client.Username}),
				})
				closed := h.RoomMgr.Leave(client.RoomCode, client.UserID)
				log.Printf("client unregistered: user=%s room=%s", client.Username, client.RoomCode)
				if closed {
					log.Printf("room %s destroyed (no real players)", client.RoomCode)
					h.Engine.EndGame(client.RoomCode)
				}
			}

		case incoming := <-h.Incoming:
			var msg Message
			if err := json.Unmarshal(incoming.Data, &msg); err != nil {
				log.Printf("invalid message: %v", err)
				continue
			}
			h.handleMessage(incoming.Client, msg)
		}
	}
}

func (h *Hub) handleMessage(sender *Client, msg Message) {
	switch msg.Type {
	case "start_game":
		h.handleStartGame(sender)
	case "add_bot":
		h.handleAddBot(sender)
	case "game_action":
		h.handleGameAction(sender, msg)
	default:
		log.Printf("unknown message type: %s", msg.Type)
	}
}

func (h *Hub) handleAddBot(sender *Client) {
	rm, ok := h.RoomMgr.Get(sender.RoomCode)
	if !ok {
		h.sendError(sender, "room not found")
		return
	}
	if rm.HostID != sender.UserID {
		h.sendError(sender, "only host can add bot")
		return
	}
	r, err := h.RoomMgr.AddBot(sender.RoomCode)
	if err != nil {
		h.sendError(sender, err.Error())
		return
	}
	
	// Broadcast new bot joined
	newBot := r.Players[len(r.Players)-1]
	h.broadcastToRoom(sender.RoomCode, Message{
		Type:    "player_join",
		Payload: mustJSON(map[string]any{
			"user_id": newBot.ID, 
			"username": newBot.Username,
			"is_bot": true,
		}),
	})
}

func (h *Hub) handleStartGame(sender *Client) {
	rm, ok := h.RoomMgr.Get(sender.RoomCode)
	if !ok {
		h.sendError(sender, "room not found")
		return
	}
	if rm.HostID != sender.UserID {
		h.sendError(sender, "only host can start")
		return
	}
	var players []game.PlayerInfo
	for _, p := range rm.Players {
		players = append(players, game.PlayerInfo{
			UserID:   p.ID,
			Username: p.Username,
			IsBot:    p.IsBot,
		})
	}
	if err := h.Engine.StartGame(sender.RoomCode, players); err != nil {
		h.sendError(sender, err.Error())
		return
	}
	h.broadcastToRoom(sender.RoomCode, Message{Type: "game_started"})
	h.sendGameState(sender.RoomCode)

	go h.runBots(sender.RoomCode)
}

func (h *Hub) handleGameAction(sender *Client, msg Message) {
	result, err := h.Engine.HandleAction(sender.RoomCode, sender.UserID, msg.Payload)
	if err != nil {
		h.sendError(sender, err.Error())
		return
	}
	if len(result.Events) > 0 {
		h.broadcastToRoom(sender.RoomCode, Message{
			Type:    "game_events",
			Payload: mustJSON(result.Events),
		})
	}
	h.sendGameState(sender.RoomCode)

	go h.runBots(sender.RoomCode)
}

func (h *Hub) runBots(roomCode string) {
	// Loop to allow consecutive bot actions. 
	// Limit set to 100 to prevent infinite loops, but allow for full rounds of bot actions.
	for i := 0; i < 100; i++ {
		// Check if next player is bot before sleeping to avoid unnecessary delay for human players
		// We do a peek without locking whole engine, relying on RunBotStep to do safe check
		
		// Sleep first to give a natural pause before bot acts
		time.Sleep(800 * time.Millisecond)

		events, acted := h.Engine.RunBotStep(roomCode)
		if !acted {
			break
		}

		if len(events) > 0 {
			h.broadcastToRoom(roomCode, Message{
				Type:    "game_events",
				Payload: mustJSON(events),
			})
		}
		h.sendGameState(roomCode)
	}
}

func (h *Hub) sendGameState(roomCode string) {
	for client := range h.Clients {
		if client.RoomCode != roomCode {
			continue
		}
		view := h.Engine.GetViewForPlayer(roomCode, client.UserID)
		if view == nil {
			continue
		}
		h.sendToClient(client, Message{Type: "state_update", Payload: mustJSON(view)})
	}
}

func (h *Hub) sendToClient(c *Client, msg Message) {
	data, _ := json.Marshal(msg)
	select {
	case c.Send <- data:
	default:
	}
}

func (h *Hub) sendError(c *Client, text string) {
	h.sendToClient(c, Message{Type: "error", Payload: mustJSON(map[string]string{"message": text})})
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

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
