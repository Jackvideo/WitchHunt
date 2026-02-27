package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/jackvidyu/witchhunt/server/internal/game"
	"github.com/jackvidyu/witchhunt/server/internal/model"
	"github.com/jackvidyu/witchhunt/server/internal/room"
	"gorm.io/gorm"
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
	DB         *gorm.DB
	recorded   map[string]bool
}

func NewHub(engine *game.SalemEngine, rm *room.Manager, db *gorm.DB) *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Incoming:   make(chan *IncomingMessage, 256),
		Engine:     engine,
		RoomMgr:    rm,
		DB:         db,
		recorded:   make(map[string]bool),
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
	case "send_emoji":
		h.handleSendEmoji(sender, msg)
	case "chat_message":
		h.handleChatMessage(sender, msg)
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
		Type: "player_join",
		Payload: mustJSON(map[string]any{
			"user_id":  newBot.ID,
			"username": newBot.Username,
			"is_bot":   true,
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

func (h *Hub) handleChatMessage(sender *Client, msg Message) {
	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil || payload.Text == "" {
		return
	}

	text := []rune(payload.Text)
	if len(text) > 200 {
		text = text[:200]
	}

	now := time.Now()
	if now.Sub(sender.LastChat) < time.Second {
		return
	}
	sender.LastChat = now

	h.broadcastToRoom(sender.RoomCode, Message{
		Type: "chat_message",
		Payload: mustJSON(map[string]any{
			"user_id":  sender.UserID,
			"username": sender.Username,
			"text":     string(text),
			"ts":       now.UnixMilli(),
		}),
	})
}

func (h *Hub) handleSendEmoji(sender *Client, msg Message) {
	var payload struct {
		Emoji string `json:"emoji"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil || payload.Emoji == "" {
		return
	}
	emojis := []rune(payload.Emoji)
	if len(emojis) > 2 {
		return
	}
	h.broadcastToRoom(sender.RoomCode, Message{
		Type: "player_emoji",
		Payload: mustJSON(map[string]any{
			"user_id": sender.UserID,
			"emoji":   payload.Emoji,
		}),
	})
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
	h.tryRecordGameResult(roomCode)
}

func (h *Hub) tryRecordGameResult(roomCode string) {
	if h.recorded[roomCode] {
		return
	}
	result := h.Engine.GetGameResult(roomCode)
	if result == nil {
		return
	}
	h.recorded[roomCode] = true

	record := model.GameRecord{
		RoomCode:    roomCode,
		Winner:      result.Winner,
		PlayerCount: result.PlayerCount,
		DayCount:    result.DayNumber,
	}
	for _, p := range result.Players {
		record.Participants = append(record.Participants, model.GameParticipant{
			UserID:   p.UserID,
			Username: p.Username,
			IsWitch:  p.IsWitch,
			Alive:    p.Alive,
			Won:      p.Won,
			IsBot:    p.IsBot,
		})
	}

	go func() {
		if err := h.DB.Create(&record).Error; err != nil {
			log.Printf("failed to record game result: %v", err)
		} else {
			log.Printf("game result recorded: room=%s winner=%s", roomCode, result.Winner)
		}
	}()
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
