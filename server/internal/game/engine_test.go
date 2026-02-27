package game

import (
	"testing"
)

// mkGame creates a controlled 4-player game for testing.
// Player 1 = witch, Player 2 = sheriff holder, Players 3-4 = villagers.
// No random deck — hand cards and draw pile are set manually per test.
func mkGame() *Game {
	g := &Game{
		Phase:           PhaseDay,
		CurrentTurn:     0,
		TotalWitchCards: 1,
		WitchVotes:      make(map[uint]uint),
		ConfessChoices:  make(map[uint]bool),
		Events:          []Event{{Message: "test game"}},
		Players: []*Player{
			{UserID: 1, Username: "Witch", Alive: true, IsWitch: true,
				Identities: []*IdentityCard{{Type: IDWitch}, {Type: IDVillager}, {Type: IDVillager}}},
			{UserID: 2, Username: "Sheriff", Alive: true,
				Identities: []*IdentityCard{{Type: IDSheriff}, {Type: IDVillager}, {Type: IDVillager}}},
			{UserID: 3, Username: "Alice", Alive: true,
				Identities: []*IdentityCard{{Type: IDVillager}, {Type: IDVillager}, {Type: IDVillager}}},
			{UserID: 4, Username: "Bob", Alive: true,
				Identities: []*IdentityCard{{Type: IDVillager}, {Type: IDVillager}, {Type: IDVillager}}},
		},
	}
	return g
}

func redCard(id string, value int) *Card {
	ct := CTAccuse1
	if value == 2 {
		ct = CTAccuse2
	} else if value == 3 {
		ct = CTAccuse3
	}
	return &Card{ID: id, Type: ct, Color: ColorRed, Value: value}
}

func greenCard(id string, ct CardType) *Card {
	return &Card{ID: id, Type: ct, Color: ColorGreen}
}

func blueCard(id string, ct CardType) *Card {
	return &Card{ID: id, Type: ct, Color: ColorBlue}
}

// ======================== Game Setup ========================

func TestNewGamePlayerCount(t *testing.T) {
	_, err := newGame([]PlayerInfo{{1, "a", false}, {2, "b", false}, {3, "c", false}})
	if err == nil {
		t.Fatal("expected error for <4 players")
	}

	pls := make([]PlayerInfo, 4)
	for i := range pls {
		pls[i] = PlayerInfo{uint(i + 1), "p", false}
	}
	g, err := newGame(pls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Phase != PhaseDay {
		t.Fatalf("expected day phase, got %s", g.Phase)
	}
	for _, p := range g.Players {
		if len(p.Identities) != 5 {
			t.Fatalf("expected 5 identities, got %d", len(p.Identities))
		}
		if len(p.Hand) != 3 {
			t.Fatalf("expected 3 hand cards, got %d", len(p.Hand))
		}
	}
}

func TestNewGameWitchAssignment(t *testing.T) {
	pls := make([]PlayerInfo, 5)
	for i := range pls {
		pls[i] = PlayerInfo{uint(i + 1), "p", false}
	}
	g, _ := newGame(pls)

	witchCount := 0
	for _, p := range g.Players {
		if p.IsWitch {
			witchCount++
		}
	}
	// 4-5 players = 1 witch card, so at least 1 witch player
	if witchCount < 1 {
		t.Fatalf("expected at least 1 witch player, got %d", witchCount)
	}
}

// ======================== Day: Draw ========================

func TestDayDraw(t *testing.T) {
	g := mkGame()
	g.DrawPile = []*Card{
		redCard("r1", 1), redCard("r2", 2), redCard("r3", 1),
	}
	g.Players[0].Hand = nil

	_, err := g.handleAction(1, Action{Type: "draw"})
	if err != nil {
		t.Fatalf("draw failed: %v", err)
	}

	if len(g.Players[0].Hand) != 2 {
		t.Fatalf("expected 2 cards in hand, got %d", len(g.Players[0].Hand))
	}
	if len(g.DrawPile) != 1 {
		t.Fatalf("expected 1 card in draw pile, got %d", len(g.DrawPile))
	}
	// After drawing, turn should advance to player 2
	if g.currentPlayer().UserID != 2 {
		t.Fatalf("expected player 2 turn, got player %d", g.currentPlayer().UserID)
	}
}

func TestDayDrawNotYourTurn(t *testing.T) {
	g := mkGame()
	g.DrawPile = []*Card{redCard("r1", 1), redCard("r2", 1)}

	_, err := g.handleAction(2, Action{Type: "draw"})
	if err == nil {
		t.Fatal("expected error: not your turn")
	}
}

func TestDayDrawNightCard(t *testing.T) {
	g := mkGame()
	nightCard := &Card{ID: "night_1", Type: CTNight, Color: ColorBlack}
	g.DrawPile = []*Card{nightCard}
	g.Players[0].Hand = nil

	_, err := g.handleAction(1, Action{Type: "draw"})
	if err != nil {
		t.Fatalf("draw failed: %v", err)
	}

	if g.Phase != PhaseNightWitch {
		t.Fatalf("expected night_witch phase, got %s", g.Phase)
	}
	if len(g.Players[0].Hand) != 0 {
		t.Fatalf("night card should not go to hand, got %d cards", len(g.Players[0].Hand))
	}
}

// ======================== Day: Play Card ========================

func TestPlayRedCard(t *testing.T) {
	g := mkGame()
	card := redCard("r1", 2)
	g.Players[0].Hand = []*Card{card}

	_, err := g.handleAction(1, Action{Type: "play_card", CardID: "r1", TargetID: 3})
	if err != nil {
		t.Fatalf("play card failed: %v", err)
	}

	if g.Players[2].AccuseTotal != 2 {
		t.Fatalf("expected accuse_total=2, got %d", g.Players[2].AccuseTotal)
	}
	if len(g.Players[0].Hand) != 0 {
		t.Fatalf("card should be removed from hand")
	}
}

func TestPlayRedCardCannotTargetSelf(t *testing.T) {
	g := mkGame()
	g.Players[0].Hand = []*Card{redCard("r1", 1)}

	_, err := g.handleAction(1, Action{Type: "play_card", CardID: "r1", TargetID: 1})
	if err == nil {
		t.Fatal("expected error: cannot target self")
	}
}

func TestPlayRedCardDevoteeBlocks(t *testing.T) {
	g := mkGame()
	g.Players[0].Hand = []*Card{redCard("r1", 2)}
	g.Players[2].Equipment = []*Card{blueCard("devotee_1", CTDevotee)}

	_, err := g.handleAction(1, Action{Type: "play_card", CardID: "r1", TargetID: 3})
	if err != nil {
		t.Fatalf("play card failed: %v", err)
	}
	if g.Players[2].AccuseTotal != 0 {
		t.Fatal("devotee should block accusation")
	}
}

// ======================== Trial ========================

func TestTrialTrigger(t *testing.T) {
	g := mkGame()
	g.Players[2].AccuseTotal = 5
	g.Players[2].Accusations = []*Card{redCard("old1", 3), redCard("old2", 2)}
	g.Players[0].Hand = []*Card{redCard("r1", 3)}

	_, err := g.handleAction(1, Action{Type: "play_card", CardID: "r1", TargetID: 3})
	if err != nil {
		t.Fatalf("play card failed: %v", err)
	}

	if g.Phase != PhaseTrial {
		t.Fatalf("expected trial phase, got %s", g.Phase)
	}
	if g.Trial.AccusedID != 3 || g.Trial.FlipperID != 1 {
		t.Fatalf("wrong trial info: %+v", g.Trial)
	}
}

func TestTrialFlipIdentity(t *testing.T) {
	g := mkGame()
	g.Phase = PhaseTrial
	g.Trial = &TrialInfo{AccusedID: 3, FlipperID: 1}
	g.Players[2].AccuseTotal = 7
	g.Players[2].Accusations = []*Card{redCard("old1", 3), redCard("old2", 2), redCard("old3", 2)}

	_, err := g.handleAction(1, Action{Type: "flip_identity", CardIndex: 0})
	if err != nil {
		t.Fatalf("flip failed: %v", err)
	}

	if !g.Players[2].Identities[0].Revealed {
		t.Fatal("first identity should be revealed")
	}
	if g.Players[2].AccuseTotal != 0 {
		t.Fatal("accusations should be cleared after trial")
	}
	if g.Phase != PhaseDay {
		t.Fatalf("expected return to day, got %s", g.Phase)
	}
}

func TestTrialFlipWrongUser(t *testing.T) {
	g := mkGame()
	g.Phase = PhaseTrial
	g.Trial = &TrialInfo{AccusedID: 3, FlipperID: 1}

	_, err := g.handleAction(2, Action{Type: "flip_identity", CardIndex: 0})
	if err == nil {
		t.Fatal("expected error: not the accuser")
	}
}

// ======================== Night: Full Cycle ========================

func TestNightFullCycle(t *testing.T) {
	g := mkGame()
	g.Phase = PhaseNightWitch

	// Witch votes to kill Alice (player 3)
	_, err := g.handleAction(1, Action{Type: "witch_kill", TargetID: 3})
	if err != nil {
		t.Fatalf("witch_kill failed: %v", err)
	}

	// Sheriff exists (player 2) → should go to night_sheriff
	if g.Phase != PhaseNightSheriff {
		t.Fatalf("expected night_sheriff, got %s", g.Phase)
	}

	// Sheriff protects Bob (player 4), not Alice
	_, err = g.handleAction(2, Action{Type: "sheriff_protect", TargetID: 4})
	if err != nil {
		t.Fatalf("sheriff_protect failed: %v", err)
	}
	if g.Phase != PhaseNightResult {
		t.Fatalf("expected night_result, got %s", g.Phase)
	}

	// Everyone passes (no confess)
	for _, uid := range []uint{1, 2, 3, 4} {
		_, err = g.handleAction(uid, Action{Type: "pass"})
		if err != nil {
			t.Fatalf("pass failed for uid=%d: %v", uid, err)
		}
	}

	// Alice should be dead (not protected by hammer, didn't confess)
	if g.Players[2].Alive {
		t.Fatal("Alice should be dead")
	}
	if g.Phase != PhaseDay {
		t.Fatalf("expected day after night, got %s", g.Phase)
	}
}

func TestNightSheriffProtects(t *testing.T) {
	g := mkGame()
	g.Phase = PhaseNightWitch

	// Witch targets Alice
	g.handleAction(1, Action{Type: "witch_kill", TargetID: 3})
	// Sheriff protects Alice
	g.handleAction(2, Action{Type: "sheriff_protect", TargetID: 3})

	// Everyone passes
	for _, uid := range []uint{1, 2, 3, 4} {
		g.handleAction(uid, Action{Type: "pass"})
	}

	if !g.Players[2].Alive {
		t.Fatal("Alice should be alive (protected by sheriff)")
	}
}

func TestNightConfessSaves(t *testing.T) {
	g := mkGame()
	g.Phase = PhaseNightWitch

	// No sheriff for this test — reveal the sheriff identity
	for _, id := range g.Players[1].Identities {
		if id.Type == IDSheriff {
			id.Revealed = true
		}
	}

	g.handleAction(1, Action{Type: "witch_kill", TargetID: 3})

	// Since sheriff is revealed, should skip to night_result
	if g.Phase != PhaseNightResult {
		t.Fatalf("expected night_result (no sheriff), got %s", g.Phase)
	}

	// Alice confesses, others pass
	g.handleAction(3, Action{Type: "confess"})
	g.handleAction(1, Action{Type: "pass"})
	g.handleAction(2, Action{Type: "pass"})
	g.handleAction(4, Action{Type: "pass"})

	if !g.Players[2].Alive {
		t.Fatal("Alice should be alive (confessed)")
	}
	// But she should have one revealed identity
	revealedCount := 0
	for _, id := range g.Players[2].Identities {
		if id.Revealed {
			revealedCount++
		}
	}
	if revealedCount != 1 {
		t.Fatalf("expected 1 revealed identity, got %d", revealedCount)
	}
}

func TestNightSanctuaryProtects(t *testing.T) {
	g := mkGame()
	g.Phase = PhaseNightWitch
	g.Players[2].Equipment = []*Card{blueCard("sanc_1", CTSanctuary)}

	// Remove sheriff so we skip to night_result
	for _, id := range g.Players[1].Identities {
		if id.Type == IDSheriff {
			id.Revealed = true
		}
	}

	g.handleAction(1, Action{Type: "witch_kill", TargetID: 3})

	for _, uid := range []uint{1, 2, 3, 4} {
		g.handleAction(uid, Action{Type: "pass"})
	}

	if !g.Players[2].Alive {
		t.Fatal("Alice should be alive (sanctuary)")
	}
}

func TestWitchCannotTargetWitch(t *testing.T) {
	g := mkGame()
	g.Phase = PhaseNightWitch

	_, err := g.handleAction(1, Action{Type: "witch_kill", TargetID: 1})
	if err == nil {
		t.Fatal("expected error: cannot target self (witch)")
	}
}

func TestSheriffCannotProtectSelf(t *testing.T) {
	g := mkGame()
	g.Phase = PhaseNightSheriff
	g.MurderTarget = 3

	_, err := g.handleAction(2, Action{Type: "sheriff_protect", TargetID: 2})
	if err == nil {
		t.Fatal("expected error: cannot protect self")
	}
}

// ======================== Win Conditions ========================

func TestVillagerWin(t *testing.T) {
	g := mkGame()

	// Reveal all witch identity cards → villager wins
	for _, p := range g.Players {
		for _, id := range p.Identities {
			if id.Type == IDWitch {
				id.Revealed = true
			}
		}
	}

	winner := g.checkWin()
	if winner != "villager" {
		t.Fatalf("expected villager win, got '%s'", winner)
	}
}

func TestWitchWin(t *testing.T) {
	g := mkGame()

	// Kill all non-witch players
	for _, p := range g.Players {
		if !p.IsWitch {
			p.Alive = false
		}
	}

	winner := g.checkWin()
	if winner != "witch" {
		t.Fatalf("expected witch win, got '%s'", winner)
	}
}

func TestNoWinYet(t *testing.T) {
	g := mkGame()
	if w := g.checkWin(); w != "" {
		t.Fatalf("expected no winner yet, got '%s'", w)
	}
}

// ======================== Green Cards ========================

func TestArson(t *testing.T) {
	g := mkGame()
	arson := greenCard("arson_1", CTArson)
	g.Players[0].Hand = []*Card{arson}
	g.Players[2].Hand = []*Card{redCard("v1", 1), redCard("v2", 2)}

	g.handleAction(1, Action{Type: "play_card", CardID: "arson_1", TargetID: 3})

	if len(g.Players[2].Hand) != 0 {
		t.Fatal("Alice's hand should be empty after arson")
	}
}

func TestDetention(t *testing.T) {
	g := mkGame()
	g.Players[0].Hand = []*Card{greenCard("det_1", CTDetention)}

	g.handleAction(1, Action{Type: "play_card", CardID: "det_1", TargetID: 2})

	if g.Players[1].Detained != 1 {
		t.Fatal("Sheriff should be detained")
	}

	// End turn, should skip detained player
	g.handleAction(1, Action{Type: "end_turn"})
	if g.currentPlayer().UserID == 2 {
		t.Fatal("detained player should be skipped")
	}
	if g.Players[1].Detained != 0 {
		t.Fatal("detained count should decrement")
	}
}

func TestCurseRemovesEquipment(t *testing.T) {
	g := mkGame()
	g.Players[0].Hand = []*Card{greenCard("curse_1", CTCurse)}
	g.Players[2].Equipment = []*Card{blueCard("sanc_1", CTSanctuary)}

	g.handleAction(1, Action{Type: "play_card", CardID: "curse_1", TargetID: 3})

	if len(g.Players[2].Equipment) != 0 {
		t.Fatal("equipment should be removed by curse")
	}
}

func TestRobbery(t *testing.T) {
	g := mkGame()
	g.Players[0].Hand = []*Card{greenCard("rob_1", CTRobbery)}
	g.Players[2].Hand = []*Card{redCard("v1", 1), redCard("v2", 2)}
	initialBobHand := len(g.Players[3].Hand)

	g.handleAction(1, Action{Type: "play_card", CardID: "rob_1", TargetID: 3, ExtraTargetID: 4})

	if len(g.Players[2].Hand) != 0 {
		t.Fatal("Alice's hand should be empty after robbery")
	}
	if len(g.Players[3].Hand) != initialBobHand+2 {
		t.Fatalf("Bob should have +2 cards, got %d", len(g.Players[3].Hand))
	}
}

func TestFrame(t *testing.T) {
	g := mkGame()
	g.Players[0].Hand = []*Card{greenCard("frame_1", CTFrame)}
	g.Players[2].AccuseTotal = 4
	g.Players[2].Accusations = []*Card{redCard("a1", 2), redCard("a2", 2)}

	g.handleAction(1, Action{Type: "play_card", CardID: "frame_1", TargetID: 3, ExtraTargetID: 4})

	if g.Players[2].AccuseTotal != 0 {
		t.Fatal("Alice's accusations should be transferred")
	}
	if g.Players[3].AccuseTotal != 4 {
		t.Fatalf("Bob should have 4 accuse points, got %d", g.Players[3].AccuseTotal)
	}
}

func TestFrameTriggersTrial(t *testing.T) {
	g := mkGame()
	g.Players[0].Hand = []*Card{greenCard("frame_1", CTFrame)}
	g.Players[2].AccuseTotal = 4
	g.Players[2].Accusations = []*Card{redCard("a1", 2), redCard("a2", 2)}
	g.Players[3].AccuseTotal = 4
	g.Players[3].Accusations = []*Card{redCard("a3", 2), redCard("a4", 2)}

	g.handleAction(1, Action{Type: "play_card", CardID: "frame_1", TargetID: 3, ExtraTargetID: 4})

	if g.Phase != PhaseTrial {
		t.Fatalf("expected trial (Bob has 8 points), got %s", g.Phase)
	}
	if g.Trial.AccusedID != 4 {
		t.Fatal("Bob should be accused")
	}
}

func TestDefenseRemovesUpTo3Points(t *testing.T) {
	g := mkGame()
	g.Players[0].Hand = []*Card{greenCard("def_1", CTDefense)}
	// Target has 5 points: [1, 1, 1, 2]
	g.Players[2].Accusations = []*Card{redCard("a1", 1), redCard("a2", 1), redCard("a3", 1), redCard("a4", 2)}
	g.Players[2].AccuseTotal = 5

	g.handleAction(1, Action{Type: "play_card", CardID: "def_1", TargetID: 3})

	// Should remove from the end: first the 2-pointer (removed=2), then a 1-pointer (removed=3), stop
	// Remaining: [1, 1] = 2 points
	if g.Players[2].AccuseTotal != 2 {
		t.Fatalf("expected 2 accuse points remaining, got %d", g.Players[2].AccuseTotal)
	}
	if len(g.Players[2].Accusations) != 2 {
		t.Fatalf("expected 2 accusations remaining, got %d", len(g.Players[2].Accusations))
	}
}

func TestDefenseOnZeroAccusations(t *testing.T) {
	g := mkGame()
	g.Players[0].Hand = []*Card{greenCard("def_1", CTDefense)}
	g.Players[2].AccuseTotal = 0

	_, err := g.handleAction(1, Action{Type: "play_card", CardID: "def_1", TargetID: 3})
	if err != nil {
		t.Fatalf("defense on 0 accusations should work: %v", err)
	}
	if g.Players[2].AccuseTotal != 0 {
		t.Fatal("should still be 0")
	}
}

// ======================== Blue Cards ========================

func TestEquipBlueCard(t *testing.T) {
	g := mkGame()
	g.Players[0].Hand = []*Card{blueCard("sanc_1", CTSanctuary)}

	g.handleAction(1, Action{Type: "play_card", CardID: "sanc_1", TargetID: 3})

	if !g.Players[2].HasEquipment(CTSanctuary) {
		t.Fatal("Alice should have sanctuary")
	}
}

// ======================== End Turn ========================

func TestEndTurn(t *testing.T) {
	g := mkGame()
	g.Players[0].Hand = []*Card{redCard("r1", 1)}

	g.handleAction(1, Action{Type: "end_turn"})

	if g.currentPlayer().UserID != 2 {
		t.Fatalf("turn should advance to player 2, got %d", g.currentPlayer().UserID)
	}
}

func TestCannotDrawAfterPlaying(t *testing.T) {
	g := mkGame()
	g.DrawPile = []*Card{redCard("d1", 1), redCard("d2", 1)}
	g.Players[0].Hand = []*Card{redCard("r1", 1)}

	g.handleAction(1, Action{Type: "play_card", CardID: "r1", TargetID: 3})

	_, err := g.handleAction(1, Action{Type: "draw"})
	if err == nil {
		t.Fatal("expected error: cannot draw after playing")
	}
}

// ======================== View ========================

func TestViewHidesOtherPlayerInfo(t *testing.T) {
	g := mkGame()
	g.Players[0].Hand = []*Card{redCard("r1", 1)}
	g.Players[1].Hand = []*Card{redCard("r2", 2)}

	view := g.viewForPlayer(1)
	if view == nil {
		t.Fatal("view should not be nil")
	}
	if len(view.Hand) != 1 || view.Hand[0].ID != "r1" {
		t.Fatal("should see own hand")
	}
	// Other players shouldn't expose hand
	for _, pp := range view.Players {
		if pp.UserID == 2 {
			// PublicPlayer doesn't have Hand field — good
			break
		}
	}
}

func TestViewWitchSeesFellows(t *testing.T) {
	g := mkGame()
	// Add a second witch
	g.Players[3].IsWitch = true
	g.Players[3].Identities[0].Type = IDWitch

	view := g.viewForPlayer(1) // Player 1 is witch
	if len(view.FellowWitches) != 1 || view.FellowWitches[0] != 4 {
		t.Fatalf("witch should see fellow witch: %v", view.FellowWitches)
	}

	viewNormal := g.viewForPlayer(2) // Player 2 is not witch
	if len(viewNormal.FellowWitches) != 0 {
		t.Fatal("non-witch should not see fellow witches")
	}
}

// ======================== Player Death ========================

func TestPlayerDeathRevealsAll(t *testing.T) {
	g := mkGame()
	g.killPlayer(g.Players[2])

	if g.Players[2].Alive {
		t.Fatal("should be dead")
	}
	for _, id := range g.Players[2].Identities {
		if !id.Revealed {
			t.Fatal("all identities should be revealed on death")
		}
	}
	if len(g.Players[2].Hand) != 0 {
		t.Fatal("hand should be empty on death")
	}
}

func TestDeathByAllRevealed(t *testing.T) {
	g := mkGame()
	p := g.Players[2]
	for _, id := range p.Identities {
		id.Revealed = true
	}
	died := g.checkPlayerDeath(p)
	if !died {
		t.Fatal("player with all identities revealed should die")
	}
}

// ======================== Skip Night Without Witches ========================

func TestNightSkipsWhenNoWitches(t *testing.T) {
	g := mkGame()
	g.Players[0].Alive = false // witch is dead

	g.enterNight()

	if g.Phase != PhaseDay {
		t.Fatalf("expected day (no witches), got %s", g.Phase)
	}
}

// ======================== Engine-level Tests ========================

func TestEngineStartAndAction(t *testing.T) {
	e := NewEngine()
	players := make([]PlayerInfo, 4)
	for i := range players {
		players[i] = PlayerInfo{uint(i + 1), "p", false}
	}

	if err := e.StartGame("room1", players); err != nil {
		t.Fatalf("start game failed: %v", err)
	}

	// Double start should fail
	if err := e.StartGame("room1", players); err == nil {
		t.Fatal("expected error on double start")
	}

	view := e.GetViewForPlayer("room1", 1)
	if view == nil {
		t.Fatal("view should not be nil")
	}
	if view.Phase != PhaseDay {
		t.Fatalf("expected day phase, got %s", view.Phase)
	}

	e.EndGame("room1")
	if v := e.GetViewForPlayer("room1", 1); v != nil {
		t.Fatal("view should be nil after EndGame")
	}
}
