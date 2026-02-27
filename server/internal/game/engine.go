package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
)

type SalemEngine struct {
	mu    sync.Mutex
	games map[string]*Game
}

func NewEngine() *SalemEngine {
	return &SalemEngine{games: make(map[string]*Game)}
}

func (e *SalemEngine) StartGame(roomCode string, players []PlayerInfo) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.games[roomCode]; ok {
		return errors.New("game already in progress")
	}
	g, err := newGame(players)
	if err != nil {
		return err
	}
	e.games[roomCode] = g
	return nil
}

func (e *SalemEngine) HandleAction(roomCode string, userID uint, raw json.RawMessage) (*ActionResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	g, ok := e.games[roomCode]
	if !ok {
		return nil, errors.New("no active game")
	}
	var action Action
	if err := json.Unmarshal(raw, &action); err != nil {
		return nil, err
	}
	return g.handleAction(userID, action)
}

func (e *SalemEngine) GetViewForPlayer(roomCode string, userID uint) *PlayerView {
	e.mu.Lock()
	defer e.mu.Unlock()
	g, ok := e.games[roomCode]
	if !ok {
		return nil
	}
	return g.viewForPlayer(userID)
}

type GameResult struct {
	Winner      string
	DayNumber   int
	PlayerCount int
	Players     []PlayerResult
}

type PlayerResult struct {
	UserID   uint
	Username string
	IsWitch  bool
	Alive    bool
	IsBot    bool
	Won      bool
}

func (e *SalemEngine) GetGameResult(roomCode string) *GameResult {
	e.mu.Lock()
	defer e.mu.Unlock()
	g, ok := e.games[roomCode]
	if !ok || g.Phase != PhaseGameOver {
		return nil
	}
	r := &GameResult{
		Winner:      g.Winner,
		DayNumber:   g.DayNumber,
		PlayerCount: len(g.Players),
	}
	for _, p := range g.Players {
		won := (g.Winner == "witch" && p.IsWitch) || (g.Winner == "villager" && !p.IsWitch)
		r.Players = append(r.Players, PlayerResult{
			UserID:   p.UserID,
			Username: p.Username,
			IsWitch:  p.IsWitch,
			Alive:    p.Alive,
			IsBot:    p.IsBot,
			Won:      won,
		})
	}
	return r
}

func (e *SalemEngine) EndGame(roomCode string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.games, roomCode)
}

// --------------- Game ---------------

type Game struct {
	Phase           Phase
	Players         []*Player
	DrawPile        []*Card
	DiscardPile     []*Card
	CurrentTurn     int
	TurnActed       bool
	Events          []Event
	Winner          string
	TotalWitchCards int
	DayNumber       int

	WitchVotes     map[uint]uint
	MurderTarget   uint
	HammerTarget   uint
	ConfessChoices map[uint]bool

	Trial *TrialInfo
}

func newGame(players []PlayerInfo) (*Game, error) {
	n := len(players)
	if n < 4 || n > 12 {
		return nil, errors.New("need 4-12 players")
	}

	ids := newIdentityCards(n)
	perPlayer := 5
	if n >= 10 {
		perPlayer = 3
	}

	gp := make([]*Player, n)
	idx := 0
	for i, pi := range players {
		gp[i] = &Player{
			UserID:     pi.UserID,
			Username:   pi.Username,
			Identities: ids[idx : idx+perPlayer],
			Alive:      true,
			IsBot:      pi.IsBot,
		}
		idx += perPlayer
	}

	for _, p := range gp {
		for _, id := range p.Identities {
			if id.Type == IDWitch {
				p.IsWitch = true
				break
			}
		}
	}

	allCards := newDeck()

	var nightCard, contagionCard, blackCatCard *Card
	var deck []*Card
	for _, c := range allCards {
		switch c.Type {
		case CTNight:
			nightCard = c
		case CTContagion:
			contagionCard = c
		case CTBlackCat:
			blackCatCard = c
		default:
			deck = append(deck, c)
		}
	}

	shuffleCards(deck)

	for _, p := range gp {
		take := min(3, len(deck))
		p.Hand = append(p.Hand, deck[:take]...)
		deck = deck[take:]
	}

	if contagionCard != nil {
		deck = append(deck, contagionCard)
		shuffleCards(deck)
	}
	if nightCard != nil {
		deck = append(deck, nightCard)
	}
	if blackCatCard != nil {
		holder := gp[rand.Intn(n)]
		holder.Equipment = append(holder.Equipment, blackCatCard)
	}

	_, wc, _ := identityCounts(n)

	return &Game{
		Phase:           PhaseDay,
		Players:         gp,
		DrawPile:        deck,
		CurrentTurn:     0,
		TotalWitchCards: wc,
		DayNumber:       0,
		WitchVotes:      make(map[uint]uint),
		ConfessChoices:  make(map[uint]bool),
		Events:          []Event{{Message: "第0天，游戏开始！"}},
	}, nil
}

func (g *Game) handleAction(userID uint, action Action) (*ActionResult, error) {
	switch g.Phase {
	case PhaseDay:
		return g.handleDayAction(userID, action)
	case PhaseNightWitch:
		return g.handleNightWitchAction(userID, action)
	case PhaseNightSheriff:
		return g.handleNightSheriffAction(userID, action)
	case PhaseNightResult:
		return g.handleNightResultAction(userID, action)
	case PhaseTrial:
		return g.handleTrialAction(userID, action)
	default:
		return nil, errors.New("game not active")
	}
}

// --------------- helpers ---------------

func (g *Game) getPlayer(id uint) *Player {
	for _, p := range g.Players {
		if p.UserID == id {
			return p
		}
	}
	return nil
}

func (g *Game) currentPlayer() *Player { return g.Players[g.CurrentTurn] }

func (g *Game) nextTurn() {
	n := len(g.Players)
	for range n {
		g.CurrentTurn = (g.CurrentTurn + 1) % n
		p := g.Players[g.CurrentTurn]
		if !p.Alive {
			continue
		}
		if p.Detained > 0 {
			p.Detained--
			g.evt("%s 被拘留，跳过回合", p.Username)
			continue
		}
		g.TurnActed = false
		return
	}
}

func (g *Game) drawCard() *Card {
	if len(g.DrawPile) == 0 {
		g.reshuffleDeck()
	}
	if len(g.DrawPile) == 0 {
		return nil
	}
	c := g.DrawPile[0]
	g.DrawPile = g.DrawPile[1:]
	return c
}

func (g *Game) reshuffleDeck() {
	g.DrawPile = append(g.DrawPile, g.DiscardPile...)
	g.DiscardPile = nil
	var night *Card
	var rest []*Card
	for _, c := range g.DrawPile {
		if c.Type == CTNight {
			night = c
		} else {
			rest = append(rest, c)
		}
	}
	shuffleCards(rest)
	if night != nil {
		rest = append(rest, night)
	}
	g.DrawPile = rest
}

func (g *Game) killPlayer(p *Player) {
	p.Alive = false
	for _, id := range p.Identities {
		id.Revealed = true
	}
	g.DiscardPile = append(g.DiscardPile, p.Hand...)
	g.DiscardPile = append(g.DiscardPile, p.Equipment...)
	g.DiscardPile = append(g.DiscardPile, p.Accusations...)
	p.Hand, p.Equipment, p.Accusations = nil, nil, nil
	p.AccuseTotal = 0

	faction := "村民阵营"
	if p.IsWitch {
		faction = "女巫阵营"
	}
	g.evtTyped("kill", map[string]interface{}{"player_id": p.UserID},
		"%s 死亡，属于%s", p.Username, faction)
}

func (g *Game) checkPlayerDeath(p *Player) bool {
	if p.Alive && len(p.Unrevealed()) == 0 {
		g.killPlayer(p)
		return true
	}
	return false
}

func (g *Game) checkWin() string {
	revealed := 0
	for _, p := range g.Players {
		for _, id := range p.Identities {
			if id.Type == IDWitch && id.Revealed {
				revealed++
			}
		}
	}
	if revealed >= g.TotalWitchCards {
		return "villager"
	}

	allWitch := true
	for _, p := range g.Players {
		if p.Alive && !p.IsWitch {
			allWitch = false
			break
		}
	}
	if allWitch {
		return "witch"
	}
	return ""
}

func (g *Game) aliveWitches() []*Player {
	var out []*Player
	for _, p := range g.Players {
		if p.Alive && p.IsWitch {
			out = append(out, p)
		}
	}
	return out
}

func (g *Game) findSheriff() *Player {
	for _, p := range g.Players {
		if !p.Alive {
			continue
		}
		for _, id := range p.Identities {
			if id.Type == IDSheriff && !id.Revealed {
				return p
			}
		}
	}
	return nil
}

func (g *Game) clearNightState() {
	g.WitchVotes = make(map[uint]uint)
	g.MurderTarget = 0
	g.HammerTarget = 0
	g.ConfessChoices = make(map[uint]bool)
	for _, p := range g.Players {
		p.HasHammer = false
	}
}

func (g *Game) updateWitchStatus() {
	for _, p := range g.Players {
		if p.IsWitch {
			continue
		}
		for _, id := range p.Identities {
			if id.Type == IDWitch {
				p.IsWitch = true
				break
			}
		}
	}
}

func (g *Game) endWithWinner(winner string) {
	g.Phase = PhaseGameOver
	g.Winner = winner
	if winner == "villager" {
		g.evt("所有女巫身份已暴露，村民阵营获胜！")
	} else {
		g.evt("所有存活玩家都是女巫阵营，女巫获胜！")
	}
}

func (g *Game) evt(format string, args ...any) {
	g.Events = append(g.Events, Event{Message: fmt.Sprintf(format, args...)})
}

func (g *Game) evtTyped(typ string, data map[string]interface{}, format string, args ...any) {
	g.Events = append(g.Events, Event{
		Message: fmt.Sprintf(format, args...),
		Type:    typ,
		Data:    data,
	})
}
