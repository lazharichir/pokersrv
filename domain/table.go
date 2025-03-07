package domain

import "github.com/lazharichir/poker/cards"

// Table represents a poker table
type Table struct {
	ID                        string
	Name                      string
	Players                   map[string]*Player
	ButtonPlayerID            string
	CommunityCards            []cards.Card
	Deck                      []cards.Card
	Pot                       int
	Ante                      int
	ContinuationBetMultiplier int
	DiscardPhaseDuration      int
	DiscardCostType           string
	DiscardCostValue          int
}

// NewTable creates a new poker table
func NewTable(id string, name string, ante int, continuationBetMultiplier int,
	discardPhaseDuration int, discardCostType string, discardCostValue int) *Table {
	return &Table{
		ID:                        id,
		Name:                      name,
		Players:                   make(map[string]*Player),
		CommunityCards:            make([]cards.Card, 0),
		Deck:                      make([]cards.Card, 0),
		Pot:                       0,
		Ante:                      ante,
		ContinuationBetMultiplier: continuationBetMultiplier,
		DiscardPhaseDuration:      discardPhaseDuration,
		DiscardCostType:           discardCostType,
		DiscardCostValue:          discardCostValue,
	}
}

// AddPlayer adds a player to the table
func (t *Table) AddPlayer(player *Player) bool {
	if _, exists := t.Players[player.ID]; exists {
		return false
	}
	t.Players[player.ID] = player
	return true
}

// RemovePlayer removes a player from the table
func (t *Table) RemovePlayer(playerID string) bool {
	if _, exists := t.Players[playerID]; !exists {
		return false
	}
	delete(t.Players, playerID)
	return true
}

// StartNewHand prepares the table for a new hand
func (t *Table) StartNewHand() {
	t.CommunityCards = t.CommunityCards[:0]
	t.Pot = 0
	for _, player := range t.Players {
		player.ResetForNewHand()
	}
}
