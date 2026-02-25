package game

import (
	"fmt"
	"math/rand"
)

func newDeck() []*Card {
	var cards []*Card
	id := 0
	add := func(ct CardType, color CardColor, value, count int) {
		for range count {
			id++
			cards = append(cards, &Card{
				ID: fmt.Sprintf("%s_%d", ct, id), Type: ct, Color: color, Value: value,
			})
		}
	}

	add(CTAccuse1, ColorRed, 1, 16)
	add(CTAccuse2, ColorRed, 2, 8)
	add(CTAccuse3, ColorRed, 3, 4)
	add(CTNight, ColorBlack, 0, 1)
	add(CTContagion, ColorBlack, 0, 1)
	add(CTBlackCat, ColorBlue, 0, 1)
	add(CTSanctuary, ColorBlue, 0, 2)
	add(CTDevotee, ColorBlue, 0, 2)
	add(CTFrame, ColorGreen, 0, 4)
	add(CTArson, ColorGreen, 0, 3)
	add(CTDetention, ColorGreen, 0, 4)
	add(CTDefense, ColorGreen, 0, 4)
	add(CTRobbery, ColorGreen, 0, 3)
	add(CTCurse, ColorGreen, 0, 4)

	return cards
}

func shuffleCards(cards []*Card) {
	rand.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
}

func identityCounts(n int) (villager, witch, sheriff int) {
	sheriff = 1
	if n <= 5 {
		witch = 1
	} else {
		witch = 2
	}
	perPlayer := 5
	if n >= 10 {
		perPlayer = 3
	}
	villager = n*perPlayer - witch - sheriff
	return
}

func newIdentityCards(numPlayers int) []*IdentityCard {
	v, w, s := identityCounts(numPlayers)
	cards := make([]*IdentityCard, 0, v+w+s)
	for range v {
		cards = append(cards, &IdentityCard{Type: IDVillager})
	}
	for range w {
		cards = append(cards, &IdentityCard{Type: IDWitch})
	}
	for range s {
		cards = append(cards, &IdentityCard{Type: IDSheriff})
	}
	rand.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	return cards
}
