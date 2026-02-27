package game

type PlayerView struct {
	Phase         Phase           `json:"phase"`
	DayNumber     int             `json:"day_number"`
	YourID        uint            `json:"your_id"`
	IsYourTurn    bool            `json:"is_your_turn"`
	Hand          []*Card         `json:"hand"`
	Identities    []*IdentityCard `json:"identities"`
	IsWitch       bool            `json:"is_witch"`
	Players       []*PublicPlayer `json:"players"`
	Actions       []string        `json:"actions"`
	Events        []Event         `json:"events"`
	DrawPileCount int             `json:"draw_pile_count"`
	Winner        string          `json:"winner,omitempty"`
	FellowWitches []uint          `json:"fellow_witches,omitempty"`
	Trial         *TrialInfo      `json:"trial,omitempty"`
	ValidTargets  []uint          `json:"valid_targets,omitempty"`
}

type PublicPlayer struct {
	UserID             uint           `json:"user_id"`
	Username           string         `json:"username"`
	Alive              bool           `json:"alive"`
	IdentityCount      int            `json:"identity_count"`
	UnrevealedCount    int            `json:"unrevealed_count"`
	RevealedIdentities []IdentityType `json:"revealed_identities"`
	Equipment          []*Card        `json:"equipment"`
	AccuseTotal        int            `json:"accuse_total"`
	Detained           int            `json:"detained"`
	HasHammer          bool           `json:"has_hammer"`
	IsWitch            bool           `json:"is_witch,omitempty"`
}

func (g *Game) viewForPlayer(userID uint) *PlayerView {
	me := g.getPlayer(userID)
	if me == nil {
		return nil
	}

	v := &PlayerView{
		Phase:         g.Phase,
		DayNumber:     g.DayNumber,
		YourID:        userID,
		IsYourTurn:    g.Phase == PhaseDay && g.currentPlayer().UserID == userID,
		Hand:          me.Hand,
		Identities:    me.Identities,
		IsWitch:       me.IsWitch,
		DrawPileCount: len(g.DrawPile),
		Winner:        g.Winner,
		Trial:         g.Trial,
		Events:        g.Events,
		Actions:       g.availableActions(me),
	}

	if me.IsWitch && me.Alive {
		for _, p := range g.Players {
			if p.IsWitch && p.UserID != userID && p.Alive {
				v.FellowWitches = append(v.FellowWitches, p.UserID)
			}
		}
	}

	v.ValidTargets = g.validTargets(me)

	for _, p := range g.Players {
		pp := &PublicPlayer{
			UserID:          p.UserID,
			Username:        p.Username,
			Alive:           p.Alive,
			IdentityCount:   len(p.Identities),
			UnrevealedCount: len(p.Unrevealed()),
			Equipment:       p.Equipment,
			AccuseTotal:     p.AccuseTotal,
			Detained:        p.Detained,
			HasHammer:       p.HasHammer,
		}
		for _, id := range p.Identities {
			if id.Revealed {
				pp.RevealedIdentities = append(pp.RevealedIdentities, id.Type)
			}
		}
		if g.Phase == PhaseGameOver {
			pp.IsWitch = p.IsWitch
		}
		v.Players = append(v.Players, pp)
	}

	return v
}

func (g *Game) availableActions(p *Player) []string {
	if !p.Alive {
		return nil
	}
	switch g.Phase {
	case PhaseDay:
		if g.currentPlayer().UserID != p.UserID {
			return nil
		}
		var a []string
		if !g.TurnActed {
			a = append(a, "draw")
		}
		if len(p.Hand) > 0 {
			a = append(a, "play_card")
		}
		a = append(a, "end_turn")
		return a

	case PhaseNightWitch:
		if p.IsWitch {
			if _, voted := g.WitchVotes[p.UserID]; !voted {
				return []string{"witch_kill"}
			}
		}
	case PhaseNightSheriff:
		if sh := g.findSheriff(); sh != nil && sh.UserID == p.UserID {
			return []string{"sheriff_protect"}
		}
	case PhaseNightResult:
		if _, done := g.ConfessChoices[p.UserID]; !done {
			return []string{"confess", "pass"}
		}
	case PhaseTrial:
		if g.Trial != nil && g.Trial.FlipperID == p.UserID {
			return []string{"flip_identity"}
		}
	}
	return nil
}

func (g *Game) validTargets(me *Player) []uint {
	var ids []uint
	switch g.Phase {
	case PhaseNightWitch:
		if !me.IsWitch {
			return nil
		}
		for _, p := range g.Players {
			if p.Alive && !p.IsWitch {
				ids = append(ids, p.UserID)
			}
		}
	case PhaseNightSheriff:
		if sh := g.findSheriff(); sh == nil || sh.UserID != me.UserID {
			return nil
		}
		for _, p := range g.Players {
			if p.Alive && p.UserID != me.UserID {
				ids = append(ids, p.UserID)
			}
		}
	case PhaseDay:
		if g.currentPlayer().UserID != me.UserID {
			return nil
		}
		for _, p := range g.Players {
			if p.Alive && p.UserID != me.UserID {
				ids = append(ids, p.UserID)
			}
		}
	}
	return ids
}
