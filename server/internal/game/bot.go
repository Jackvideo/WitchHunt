package game

import (
	"log"
	"math/rand"
)

// RunBotStep executes a single bot action if applicable.
// Returns events generated and whether a bot acted.
func (e *SalemEngine) RunBotStep(roomCode string) ([]Event, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	g, ok := e.games[roomCode]
	if !ok || g.Phase == PhaseGameOver {
		return nil, false
	}

	var result *ActionResult
	var err error
	var acted bool

	switch g.Phase {
	case PhaseDay:
		p := g.currentPlayer()
		if p.IsBot && p.Alive {
			action := g.generateBotDayAction(p)
			result, err = g.handleDayAction(p.UserID, action)
			if err != nil {
				log.Printf("Bot %s day action error: %v", p.Username, err)
				// If action fails, just end turn to avoid loop
				g.nextTurn()
				acted = true
			} else {
				acted = true
			}
		}
	case PhaseNightWitch:
		// Need to return result if any witch voted
		res, didAct := g.handleBotWitchAction()
		if didAct {
			result = res
			acted = true
		}
	case PhaseNightSheriff:
		sheriff := g.findSheriff()
		if sheriff != nil && sheriff.IsBot {
			action := g.generateBotSheriffAction(sheriff)
			result, err = g.handleNightSheriffAction(sheriff.UserID, action)
			if err != nil {
				log.Printf("Bot Sheriff %s action error: %v", sheriff.Username, err)
			}
			acted = true
		}
	case PhaseNightResult:
		res, didAct := g.handleBotNightResult()
		if didAct {
			result = res
			acted = true
		}
	case PhaseTrial:
		if g.Trial != nil {
			flipper := g.getPlayer(g.Trial.FlipperID)
			if flipper != nil && flipper.IsBot {
				action := g.generateBotTrialAction(flipper)
				result, err = g.handleTrialAction(flipper.UserID, action)
				if err != nil {
					log.Printf("Bot %s trial action error: %v", flipper.Username, err)
				}
				acted = true
			}
		}
	}

	if result != nil {
		return result.Events, acted
	}
	return nil, acted
}

func (g *Game) generateBotDayAction(p *Player) Action {
	// Simple logic:
	// 1. If has accusation cards, use them on random other player
	// 2. Otherwise pass (end turn)

	// Try to play accusation
	for i, c := range p.Hand {
		if c.Type == CTAccuse1 || c.Type == CTAccuse2 || c.Type == CTAccuse3 {
			// Find a random target that is alive and not self
			var targets []uint
			for _, other := range g.Players {
				if other.Alive && other.UserID != p.UserID {
					targets = append(targets, other.UserID)
				}
			}

			if len(targets) > 0 {
				targetID := targets[rand.Intn(len(targets))]
				return Action{
					Type:      "play_card",
					CardID:    c.ID,
					TargetID:  targetID,
					CardIndex: i,
				}
			}
		}
	}

	// If no playable cards and hasn't acted, draw
	if !g.TurnActed {
		return Action{Type: "draw"}
	}

	return Action{Type: "end_turn"}
}

func (g *Game) handleBotWitchAction() (*ActionResult, bool) {
	// Find alive witches
	witches := g.aliveWitches()
	for _, w := range witches {
		if !w.IsBot {
			continue
		}
		// Check if already voted
		if _, ok := g.WitchVotes[w.UserID]; ok {
			continue
		}

		// Vote for a random non-witch player
		var targets []uint
		for _, p := range g.Players {
			if p.Alive && !p.IsWitch {
				targets = append(targets, p.UserID)
			}
		}

		if len(targets) > 0 {
			targetID := targets[rand.Intn(len(targets))]
			res, _ := g.handleNightWitchAction(w.UserID, Action{
				Type:     "witch_kill",
				TargetID: targetID,
			})
			return res, true
		}
	}
	return nil, false
}

func (g *Game) generateBotSheriffAction(p *Player) Action {
	// Check random player
	var targets []uint
	for _, other := range g.Players {
		if other.Alive && other.UserID != p.UserID {
			targets = append(targets, other.UserID)
		}
	}

	if len(targets) > 0 {
		targetID := targets[rand.Intn(len(targets))]
		return Action{
			Type:     "sheriff_protect",
			TargetID: targetID,
		}
	}
	return Action{} // Should not happen
}

func (g *Game) handleBotNightResult() (*ActionResult, bool) {
	for _, p := range g.Players {
		if !p.Alive || !p.IsBot {
			continue
		}
		if _, ok := g.ConfessChoices[p.UserID]; ok {
			continue
		}
		// Bot always passes
		res, _ := g.handleNightResultAction(p.UserID, Action{Type: "pass"})
		return res, true
	}
	return nil, false
}

func (g *Game) generateBotTrialAction(p *Player) Action {
	accused := g.getPlayer(g.Trial.AccusedID)
	if accused == nil {
		return Action{}
	}

	// Pick random card index
	ur := accused.Unrevealed()
	if len(ur) == 0 {
		return Action{}
	}
	idx := rand.Intn(len(ur))

	return Action{
		Type:      "flip_identity",
		CardIndex: idx,
	}
}
