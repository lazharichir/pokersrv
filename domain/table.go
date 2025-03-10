package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lazharichir/poker/domain/cards"
	"github.com/lazharichir/poker/domain/events"
	"github.com/lazharichir/poker/domain/hands"
	"github.com/sanity-io/litter"
)

func NewTable(name string, rules TableRules) *Table {
	return &Table{
		ID:            uuid.NewString(),
		Name:          name,
		Status:        TableStatusWaiting,
		BuyIns:        make(map[string]int),
		Events:        []events.Event{},
		eventHandlers: []events.EventHandler{},
		Rules:         rules,
		Players:       []Player{},
		Hands:         []Hand{},
		ActiveHand:    nil,
	}
}

// Table represents a poker table
type Table struct {
	ID         string
	Name       string
	Rules      TableRules
	Players    []Player
	Hands      []Hand
	ActiveHand *Hand
	Status     TableStatus
	BuyIns     map[string]int

	// events
	Events        []events.Event
	eventHandlers []events.EventHandler
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

// SeatPlayer adds a player to the table
func (t *Table) SeatPlayer(player Player) error {
	if t.Status != TableStatusWaiting && t.Status != TableStatusPlaying {
		return errors.New("can only add players when table is waiting or playing")
	}

	// Check if player already exists
	for _, p := range t.Players {
		if p.ID == player.ID {
			return errors.New("player already at table")
		}
	}

	t.Players = append(t.Players, player)

	t.emitEvent(events.PlayerJoinedTable{
		TableID: t.ID,
		UserID:  player.ID,
		At:      time.Now(),
	})

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
		return errors.New("player does not have enough balance")
	}

	t.Players[playerIndex].RemoveFromBalance(chips)
	t.IncreasePlayerBuyIn(playerID, chips)

	return nil
}

func (t *Table) GetPlayerBuyIn(playerID string) int {
	if _, exists := t.BuyIns[playerID]; !exists {
		return 0
	}
	return t.BuyIns[playerID]
}

func (t *Table) IncreasePlayerBuyIn(playerID string, amount int) {
	if _, exists := t.BuyIns[playerID]; !exists {
		t.BuyIns[playerID] = 0
	}

	before := t.BuyIns[playerID]
	t.BuyIns[playerID] += amount
	after := t.BuyIns[playerID]

	t.emitEvent(events.PlayerChipsChanged{
		TableID: t.ID,
		UserID:  playerID,
		At:      time.Now(),
		Before:  before,
		After:   after,
		Change:  after - before,
	})
}

func (t *Table) DecreasePlayerBuyIn(playerID string, amount int) {
	if _, exists := t.BuyIns[playerID]; !exists {
		t.BuyIns[playerID] = 0
	}

	before := t.BuyIns[playerID]
	t.BuyIns[playerID] -= amount
	after := t.BuyIns[playerID]

	t.emitEvent(events.PlayerChipsChanged{
		TableID: t.ID,
		UserID:  playerID,
		At:      time.Now(),
		Before:  before,
		After:   after,
		Change:  after - before,
	})
}

func (t *Table) removePlayerFromBuyIns(playerID string) {
	delete(t.BuyIns, playerID)
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
	t.removePlayerFromBuyIns(playerID)

	t.emitEvent(events.PlayerLeftTable{
		TableID: t.ID,
		UserID:  playerID,
		At:      time.Now(),
	})

	return nil
}

// AllowPlaying starts the table if there are enough players
func (t *Table) AllowPlaying() error {
	if len(t.Players) < 2 {
		return errors.New("need at least 2 players to start")
	}

	if t.Status != TableStatusWaiting {
		return errors.New("table must be in waiting status to start playing")
	}

	t.Status = TableStatusPlaying

	return nil
}

// StartNewHand starts a new hand at the table
func (t *Table) StartNewHand() (*Hand, error) {
	if t.Status != TableStatusPlaying {
		return nil, errors.New("table must be in playing status to start a new hand")
	}

	// Check if there is an active hand
	if t.ActiveHand != nil {
		return nil, errors.New("there is already an active hand: " + t.ActiveHand.ID)
	}

	// Create the first hand
	hand := &Hand{
		ID:                          uuid.NewString(),
		Table:                       t,
		TableID:                     t.ID,
		Players:                     t.Players,
		Phase:                       HandPhase_Start,
		CommunityCards:              []cards.Card{},
		HoleCards:                   make(map[string]cards.Stack),
		Pot:                         0,
		Events:                      []events.Event{},
		eventHandlers:               []events.EventHandler{},
		TableRules:                  t.Rules,
		Deck:                        cards.NewDeck52(),
		Results:                     []hands.HandComparisonResult{},
		CurrentBettor:               "",
		CommunitySelections:         make(map[string]cards.Stack),
		CommunitySelectionStartedAt: time.Time{},
		// Initialize new tracking fields
		AntesPaid:        make(map[string]int),
		ContinuationBets: make(map[string]int),
		ActivePlayers:    make(map[string]bool),
		ButtonPosition:   t.findButtonPosition(), // Implement this method to track button
		StartedAt:        time.Time{},
	}

	hand.RegisterEventHandler(t.handleHandEvent)

	t.setActiveHand(hand)

	return hand, nil
}

func (t *Table) handleHandEvent(event events.Event) {
	fmt.Println("---")
	fmt.Println("Table received event:", event.Name())
	litter.D(event)

	t.emitEvent(event)

	switch ev := event.(type) {
	case events.HandEnded:
		fmt.Println("Hand ended with pot = ", ev.FinalPot)
		t.ActiveHand = nil
		t.StartNewHand()
	}
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

// RegisterEventHandler registers a callback function that will be called when events occur
func (t *Table) RegisterEventHandler(handler events.EventHandler) {
	t.eventHandlers = append(t.eventHandlers, handler)
}

// emitEvent notifies all registered handlers of a new event
func (t *Table) emitEvent(event events.Event) {
	// Add event to hand's event log
	t.Events = append(t.Events, event)

	// Notify all handlers
	for _, handler := range t.eventHandlers {
		handler(event)
	}
}
