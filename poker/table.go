package poker

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lazharichir/poker/cards"
)

// Table represents a poker table
type Table struct {
	ID         string
	Name       string
	Rules      TableRules
	Players    []Player
	Hands      []Hand
	ActiveHand *Hand
	Status     TableStatus
}

type TableStatus string

const (
	TableStatusWaiting TableStatus = "waiting"
	TableStatusPlaying TableStatus = "playing"
	TableStatusEnded   TableStatus = "ended"
)

// TableRules defines the rules for a poker table
type TableRules struct {
	AnteValue                 int
	ContinuationBetMultiplier int
	DiscardPhaseDuration      int
	DiscardCostType           string
	DiscardCostValue          int
	PlayerTimeout             time.Duration
}

// PlayerSeats adds a player to the table
func (t *Table) PlayerSeats(player Player) error {
	if t.Status != TableStatusWaiting {
		return errors.New("can only add players when table is waiting")
	}

	// Check if player already exists
	for _, p := range t.Players {
		if p.ID == player.ID {
			return errors.New("player already at table")
		}
	}

	// Add chips to player
	playerWithChips := player

	t.Players = append(t.Players, playerWithChips)

	// Publish player added event
	// This would be implemented with a proper message type in a real system

	return nil
}

// PlayerBuysIn adds chips to a player's balance at the table, and removes them from the player's global balance
func (t *Table) PlayerBuysIn(playerID string, chips int) error {
	if t.Status != TableStatusWaiting {
		return errors.New("can only add chips when table is waiting")
	}

	playerIndex := -1
	for i, p := range t.Players {
		if p.ID == playerID {
			playerIndex = i
			break
		}
	}

	if playerIndex == -1 {
		return errors.New("player not found")
	}

	if t.Players[playerIndex].Balance < chips {
		return errors.New("player does not have enough chips")
	}

	t.Players[playerIndex].Balance -= chips
	t.Players[playerIndex].Chips += chips

	return nil
}

// PlayerLeaves removes a player from the table
func (t *Table) PlayerLeaves(playerID string) error {
	playerIndex := -1
	for i, p := range t.Players {
		if p.ID == playerID {
			playerIndex = i
			break
		}
	}

	if playerIndex == -1 {
		return errors.New("player not found")
	}

	t.Players = append(t.Players[:playerIndex], t.Players[playerIndex+1:]...)

	return nil
}

// AllowPlaying starts the table if there are enough players
func (t *Table) AllowPlaying() error {
	if len(t.Players) < 2 {
		return errors.New("need at least  players to start")
	}

	if t.Status != TableStatusWaiting {
		return errors.New("table must be in waiting status to start playing")
	}

	t.Status = TableStatusPlaying

	return nil
}

// StartNewHand starts a new hand at the table
func (t *Table) StartNewHand() error {
	if t.Status != TableStatusPlaying {
		return errors.New("table must be in playing status to start a new hand")
	}

	// Create the first hand
	hand := Hand{
		ID:             uuid.NewString(),
		TableID:        t.ID,
		Players:        t.Players,
		CommunityCards: []cards.Card{},
		HoleCards:      make(map[string]cards.Stack),
		Pot:            0,
		Events:         []Event{},
		TableRules:     t.Rules,
		StartedAt:      time.Now(),
		Phase:          HandPhase_Start,
		// Initialize new tracking fields
		AntesPaid:        make(map[string]int),
		ContinuationBets: make(map[string]int),
		DiscardCosts:     make(map[string]int),
		ActivePlayers:    make(map[string]bool),
		ButtonPosition:   t.findButtonPosition(), // Implement this method to track button
	}

	t.setActiveHand(&hand)

	return nil
}

// findButtonPosition gets the current button position or sets it to 0 if not yet defined
func (t *Table) findButtonPosition() int {
	// Implementation depends on how you want to track the button
	// For first hand, it could be random or set to 0
	// For subsequent hands, it would move clockwise

	// Simplified version: return 0 or current position + 1
	if t.ActiveHand == nil {
		return 0
	}

	return (t.ActiveHand.ButtonPosition + 1) % len(t.Players)
}

func (t *Table) setActiveHand(hand *Hand) {
	t.ActiveHand = hand
	t.Hands = append(t.Hands, *hand)
}

func (t *Table) publish(msg Message) {
	// This would be implemented with a proper message queue in a real system
}
