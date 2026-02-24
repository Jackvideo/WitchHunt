package game

import "encoding/json"

// Engine defines the interface for game logic.
// Implement this interface to plug in actual game rules (e.g. Werewolf, Mafia).
type Engine interface {
	OnPlayerJoin(roomCode string, userID uint)
	OnPlayerLeave(roomCode string, userID uint)
	OnPlayerAction(roomCode string, userID uint, action json.RawMessage) (broadcast json.RawMessage, err error)
	GetState(roomCode string) json.RawMessage
}

// NoopEngine is a placeholder that does nothing.
// Replace with real game logic when ready.
type NoopEngine struct{}

func (NoopEngine) OnPlayerJoin(string, uint)  {}
func (NoopEngine) OnPlayerLeave(string, uint) {}
func (NoopEngine) OnPlayerAction(string, uint, json.RawMessage) (json.RawMessage, error) {
	return nil, nil
}
func (NoopEngine) GetState(string) json.RawMessage { return nil }
