package room

import (
	"crypto/rand"
	"errors"
	"math/big"
	"sync"
)

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrRoomFull     = errors.New("room is full")
	ErrAlreadyIn    = errors.New("already in room")
)

type Player struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Ready    bool   `json:"ready"`
}

type Room struct {
	Code       string    `json:"code"`
	HostID     uint      `json:"host_id"`
	Players    []*Player `json:"players"`
	MaxPlayers int       `json:"max_players"`
	Status     string    `json:"status"` // waiting, playing, finished
}

type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewManager() *Manager {
	return &Manager{rooms: make(map[string]*Room)}
}

func (m *Manager) Create(hostID uint, hostName string, maxPlayers int) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()

	code := m.generateCode()
	r := &Room{
		Code:       code,
		HostID:     hostID,
		MaxPlayers: maxPlayers,
		Status:     "waiting",
		Players: []*Player{
			{ID: hostID, Username: hostName},
		},
	}
	m.rooms[code] = r
	return r
}

func (m *Manager) Join(code string, playerID uint, username string) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, ok := m.rooms[code]
	if !ok {
		return nil, ErrRoomNotFound
	}
	if len(r.Players) >= r.MaxPlayers {
		return nil, ErrRoomFull
	}
	for _, p := range r.Players {
		if p.ID == playerID {
			return r, ErrAlreadyIn
		}
	}
	r.Players = append(r.Players, &Player{ID: playerID, Username: username})
	return r, nil
}

func (m *Manager) Leave(code string, playerID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, ok := m.rooms[code]
	if !ok {
		return
	}
	for i, p := range r.Players {
		if p.ID == playerID {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			break
		}
	}
	if len(r.Players) == 0 {
		delete(m.rooms, code)
	}
}

func (m *Manager) Get(code string) (*Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[code]
	return r, ok
}

func (m *Manager) PlayerIDs(code string) []uint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[code]
	if !ok {
		return nil
	}
	ids := make([]uint, len(r.Players))
	for i, p := range r.Players {
		ids[i] = p.ID
	}
	return ids
}

const codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func (m *Manager) generateCode() string {
	for {
		b := make([]byte, 6)
		for i := range b {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(codeChars))))
			b[i] = codeChars[n.Int64()]
		}
		code := string(b)
		if _, exists := m.rooms[code]; !exists {
			return code
		}
	}
}
