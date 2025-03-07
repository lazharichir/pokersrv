package domain

import "github.com/lazharichir/poker/cards"

// Player represents a poker player at a table
type Player struct {
	ID                     string
	Name                   string
	Chips                  int
	HoleCards              []cards.Card
	SelectedCommunityCards []cards.Card
	CurrentBet             int
	Folded                 bool
}

// NewPlayer creates a new player with the given ID and name
func NewPlayer(id string, name string, startingChips int) *Player {
	return &Player{
		ID:                     id,
		Name:                   name,
		Chips:                  startingChips,
		HoleCards:              make([]cards.Card, 0, 2),
		SelectedCommunityCards: make([]cards.Card, 0, 3),
		CurrentBet:             0,
		Folded:                 false,
	}
}

// ResetForNewHand resets the player's state for a new hand
func (p *Player) ResetForNewHand() {
	p.HoleCards = p.HoleCards[:0]
	p.SelectedCommunityCards = p.SelectedCommunityCards[:0]
	p.CurrentBet = 0
	p.Folded = false
}
