package game

import (
	"errors"
	"fmt"
	"math/rand"
)

// ===================== Day =====================

func (g *Game) handleDayAction(userID uint, action Action) (*ActionResult, error) {
	if g.currentPlayer().UserID != userID {
		return nil, errors.New("not your turn")
	}
	switch action.Type {
	case "draw":
		if g.TurnActed {
			return nil, errors.New("already acted, cannot draw")
		}
		return g.handleDraw()
	case "play_card":
		return g.handlePlayCard(action)
	case "end_turn":
		g.nextTurn()
		return &ActionResult{}, nil
	default:
		return nil, fmt.Errorf("unknown day action: %s", action.Type)
	}
}

func (g *Game) handleDraw() (*ActionResult, error) {
	p := g.currentPlayer()
	result := &ActionResult{}

	for range 2 {
		card := g.drawCard()
		if card == nil {
			break
		}
		if card.Color == ColorBlack {
			evts := g.processBlackCard(card)
			result.Events = append(result.Events, evts...)
			if g.Phase != PhaseDay {
				return result, nil
			}
		} else {
			p.Hand = append(p.Hand, card)
		}
	}

	if g.Phase == PhaseDay {
		g.nextTurn()
	}
	return result, nil
}

func (g *Game) handlePlayCard(action Action) (*ActionResult, error) {
	p := g.currentPlayer()
	card := p.RemoveHandCard(action.CardID)
	if card == nil {
		return nil, errors.New("card not in hand")
	}

	target := g.getPlayer(action.TargetID)
	if target == nil || !target.Alive || target.UserID == p.UserID {
		p.Hand = append(p.Hand, card)
		return nil, errors.New("invalid target")
	}

	g.TurnActed = true
	result := &ActionResult{}

	switch card.Color {
	case ColorRed:
		if target.HasEquipment(CTDevotee) {
			g.evt("%s 有信徒保护，%s 的指控无效", target.Username, p.Username)
			g.DiscardPile = append(g.DiscardPile, card)
			return result, nil
		}
		target.Accusations = append(target.Accusations, card)
		target.AccuseTotal += card.Value
		g.evt("%s 对 %s 发起 %d 点指控（累计 %d/7）", p.Username, target.Username, card.Value, target.AccuseTotal)

		if target.AccuseTotal >= 7 {
			g.Phase = PhaseTrial
			g.Trial = &TrialInfo{AccusedID: target.UserID, FlipperID: p.UserID}
			g.evt("审判触发！%s 将翻开 %s 的一张身份牌", p.Username, target.Username)
		}

	case ColorGreen:
		evts := g.processGreenCard(p, target, card, action)
		result.Events = append(result.Events, evts...)
		g.DiscardPile = append(g.DiscardPile, card)

	case ColorBlue:
		target.Equipment = append(target.Equipment, card)
		g.evt("%s 为 %s 装备了[%s]", p.Username, target.Username, CardNames[card.Type])
	}

	return result, nil
}

// ===================== Black cards =====================

func (g *Game) processBlackCard(card *Card) []Event {
	g.DiscardPile = append(g.DiscardPile, card)
	switch card.Type {
	case CTNight:
		g.enterNight()
		return []Event{{Message: "夜幕降临..."}}
	case CTContagion:
		return g.processContagion()
	}
	return nil
}

func (g *Game) enterNight() {
	g.clearNightState()
	if len(g.aliveWitches()) == 0 {
		g.evt("没有存活的女巫，跳过夜晚")
		g.reshuffleDeck()
		g.Phase = PhaseDay
		g.nextTurn()
		return
	}
	g.Phase = PhaseNightWitch
	g.evt("女巫请睁眼，选择要杀害的目标")
}

func (g *Game) processContagion() []Event {
	events := []Event{{Message: "传染！所有玩家从左边玩家获取一张身份牌"}}

	for _, p := range g.Players {
		if p.Alive && p.HasEquipment(CTBlackCat) {
			if ur := p.Unrevealed(); len(ur) > 0 {
				ur[0].Revealed = true
				events = append(events, Event{
					Message: fmt.Sprintf("%s 持有黑猫，被迫翻开一张[%s]身份牌", p.Username, IdentityNames[ur[0].Type]),
				})
				g.checkPlayerDeath(p)
			}
			break
		}
	}

	var alive []*Player
	for _, p := range g.Players {
		if p.Alive {
			alive = append(alive, p)
		}
	}
	n := len(alive)
	if n < 2 {
		return events
	}

	type xfer struct {
		card *IdentityCard
		from *Player
	}
	takes := make([]xfer, n)
	for i := range alive {
		left := alive[(i-1+n)%n]
		ur := left.Unrevealed()
		if len(ur) > 0 {
			takes[i] = xfer{card: ur[rand.Intn(len(ur))], from: left}
		}
	}

	for _, t := range takes {
		if t.card == nil {
			continue
		}
		for j, id := range t.from.Identities {
			if id == t.card {
				t.from.Identities = append(t.from.Identities[:j], t.from.Identities[j+1:]...)
				break
			}
		}
	}
	for i, t := range takes {
		if t.card != nil {
			alive[i].Identities = append(alive[i].Identities, t.card)
		}
	}

	g.updateWitchStatus()
	return events
}

// ===================== Green cards =====================

func (g *Game) processGreenCard(player, target *Player, card *Card, action Action) []Event {
	switch card.Type {
	case CTDefense:
		for target.AccuseTotal > 0 && len(target.Accusations) > 0 {
			last := target.Accusations[len(target.Accusations)-1]
			target.AccuseTotal -= last.Value
			target.Accusations = target.Accusations[:len(target.Accusations)-1]
			g.DiscardPile = append(g.DiscardPile, last)
			if target.AccuseTotal <= target.AccuseTotal-3 {
				break
			}
		}
		if target.AccuseTotal < 0 {
			target.AccuseTotal = 0
		}
		return []Event{{Message: fmt.Sprintf("%s 为 %s 辩护（剩余 %d/7）", player.Username, target.Username, target.AccuseTotal)}}

	case CTArson:
		cnt := len(target.Hand)
		g.DiscardPile = append(g.DiscardPile, target.Hand...)
		target.Hand = nil
		return []Event{{Message: fmt.Sprintf("%s 纵火烧毁了 %s 的 %d 张手牌", player.Username, target.Username, cnt)}}

	case CTDetention:
		target.Detained++
		return []Event{{Message: fmt.Sprintf("%s 拘留了 %s", player.Username, target.Username)}}

	case CTCurse:
		if len(target.Equipment) > 0 {
			eq := target.Equipment[0]
			target.Equipment = target.Equipment[1:]
			g.DiscardPile = append(g.DiscardPile, eq)
			return []Event{{Message: fmt.Sprintf("%s 诅咒移除了 %s 的[%s]", player.Username, target.Username, CardNames[eq.Type])}}
		}
		return []Event{{Message: fmt.Sprintf("%s 诅咒 %s，但无装备可移除", player.Username, target.Username)}}

	case CTRobbery:
		extra := g.getPlayer(action.ExtraTargetID)
		if extra == nil || !extra.Alive || extra.UserID == player.UserID {
			return []Event{{Message: "抢劫：第二目标无效"}}
		}
		cnt := len(target.Hand)
		extra.Hand = append(extra.Hand, target.Hand...)
		target.Hand = nil
		return []Event{{Message: fmt.Sprintf("%s 抢劫了 %s 的 %d 张手牌给 %s", player.Username, target.Username, cnt, extra.Username)}}

	case CTFrame:
		extra := g.getPlayer(action.ExtraTargetID)
		if extra == nil || !extra.Alive || extra.UserID == player.UserID {
			return []Event{{Message: "嫁祸：第二目标无效"}}
		}
		extra.Accusations = append(extra.Accusations, target.Accusations...)
		extra.AccuseTotal += target.AccuseTotal
		target.Accusations = nil
		target.AccuseTotal = 0
		evts := []Event{{Message: fmt.Sprintf("%s 将 %s 的指控嫁祸给 %s（累计 %d/7）", player.Username, target.Username, extra.Username, extra.AccuseTotal)}}
		if extra.AccuseTotal >= 7 {
			g.Phase = PhaseTrial
			g.Trial = &TrialInfo{AccusedID: extra.UserID, FlipperID: player.UserID}
			evts = append(evts, Event{Message: fmt.Sprintf("审判触发！%s 将翻开 %s 的一张身份牌", player.Username, extra.Username)})
		}
		return evts
	}
	return nil
}

// ===================== Night: Witch =====================

func (g *Game) handleNightWitchAction(userID uint, action Action) (*ActionResult, error) {
	p := g.getPlayer(userID)
	if p == nil || !p.IsWitch || !p.Alive {
		return nil, errors.New("not a witch")
	}
	if action.Type != "witch_kill" {
		return nil, errors.New("must choose a target")
	}
	target := g.getPlayer(action.TargetID)
	if target == nil || !target.Alive || target.IsWitch {
		return nil, errors.New("invalid target")
	}

	g.WitchVotes[userID] = action.TargetID

	witches := g.aliveWitches()
	if len(g.WitchVotes) < len(witches) {
		return &ActionResult{}, nil
	}

	counts := make(map[uint]int)
	for _, tid := range g.WitchVotes {
		counts[tid]++
	}
	best, bestN := uint(0), 0
	var candidates []uint
	for tid, n := range counts {
		if n > bestN {
			bestN = n
			candidates = []uint{tid}
			best = tid
		} else if n == bestN {
			candidates = append(candidates, tid)
		}
	}
	if len(candidates) > 1 {
		best = candidates[rand.Intn(len(candidates))]
	}
	g.MurderTarget = best

	if sh := g.findSheriff(); sh != nil {
		g.Phase = PhaseNightSheriff
		g.evt("警长请睁眼，选择要保护的玩家")
	} else {
		g.Phase = PhaseNightResult
		g.evt("天亮了，玩家可以选择自首或跳过")
	}
	return &ActionResult{}, nil
}

// ===================== Night: Sheriff =====================

func (g *Game) handleNightSheriffAction(userID uint, action Action) (*ActionResult, error) {
	sh := g.findSheriff()
	if sh == nil || sh.UserID != userID {
		return nil, errors.New("not the sheriff")
	}
	if action.Type != "sheriff_protect" {
		return nil, errors.New("must choose a target")
	}
	target := g.getPlayer(action.TargetID)
	if target == nil || !target.Alive || target.UserID == userID {
		return nil, errors.New("invalid target")
	}

	g.HammerTarget = target.UserID
	target.HasHammer = true
	g.Phase = PhaseNightResult
	g.evt("天亮了，玩家可以选择自首或跳过")
	return &ActionResult{}, nil
}

// ===================== Night: Result =====================

func (g *Game) handleNightResultAction(userID uint, action Action) (*ActionResult, error) {
	p := g.getPlayer(userID)
	if p == nil || !p.Alive {
		return nil, errors.New("not alive")
	}
	if _, done := g.ConfessChoices[userID]; done {
		return nil, errors.New("already chose")
	}

	switch action.Type {
	case "confess":
		ur := p.Unrevealed()
		if len(ur) == 0 {
			return nil, errors.New("no identity to reveal")
		}
		ur[0].Revealed = true
		g.ConfessChoices[userID] = true
		g.evt("%s 自首，翻开了一张[%s]身份牌", p.Username, IdentityNames[ur[0].Type])
		g.checkPlayerDeath(p)
	case "pass":
		g.ConfessChoices[userID] = false
	default:
		return nil, errors.New("use confess or pass")
	}

	for _, ap := range g.Players {
		if ap.Alive {
			if _, ok := g.ConfessChoices[ap.UserID]; !ok {
				return &ActionResult{}, nil
			}
		}
	}
	return g.resolveNight()
}

func (g *Game) resolveNight() (*ActionResult, error) {
	target := g.getPlayer(g.MurderTarget)
	if target != nil && target.Alive {
		confessed := g.ConfessChoices[target.UserID]
		switch {
		case confessed:
			g.evt("%s 因自首免于被杀", target.Username)
		case target.HasEquipment(CTSanctuary):
			g.evt("%s 受到避难所保护", target.Username)
		case target.HasHammer:
			g.evt("%s 受到警长保护", target.Username)
		default:
			g.killPlayer(target)
		}
	}

	if w := g.checkWin(); w != "" {
		g.endWithWinner(w)
		return &ActionResult{}, nil
	}

	g.clearNightState()
	g.reshuffleDeck()
	g.Phase = PhaseDay
	g.nextTurn()
	g.evt("新的一天开始了")
	return &ActionResult{}, nil
}

// ===================== Trial =====================

func (g *Game) handleTrialAction(userID uint, action Action) (*ActionResult, error) {
	if g.Trial == nil || g.Trial.FlipperID != userID {
		return nil, errors.New("not the accuser")
	}
	if action.Type != "flip_identity" {
		return nil, errors.New("must flip an identity card")
	}

	accused := g.getPlayer(g.Trial.AccusedID)
	if accused == nil {
		return nil, errors.New("accused not found")
	}
	ur := accused.Unrevealed()
	if action.CardIndex < 0 || action.CardIndex >= len(ur) {
		return nil, errors.New("invalid card index")
	}

	card := ur[action.CardIndex]
	card.Revealed = true
	g.evt("审判：翻开了 %s 的一张[%s]身份牌", accused.Username, IdentityNames[card.Type])

	g.DiscardPile = append(g.DiscardPile, accused.Accusations...)
	accused.Accusations = nil
	accused.AccuseTotal = 0

	g.checkPlayerDeath(accused)
	g.Trial = nil

	if w := g.checkWin(); w != "" {
		g.endWithWinner(w)
		return &ActionResult{}, nil
	}

	g.Phase = PhaseDay
	return &ActionResult{}, nil
}
