package poker

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lazharichir/poker/cards"
)

// Message represents a game event message
type Message interface {
	MessageName() string
}

// Game represents the poker game server
type Game struct {
	tables       map[string]*Table
	messageQueue []Message
	listeners    []func(Message)
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

// Player represents a player in the game
type Player struct {
	ID      string
	Name    string
	Balance int
	Status  string
	Chips   int // chips brought to the table
}

// HandState represents the current state of a hand
type HandState struct {
	// State information
}

// ActionName represents a type of action a player can take
type ActionName string

// Event represents something that happened during a hand
type Event struct {
	Type      string
	PlayerID  string
	Timestamp time.Time
	Data      interface{}
}

type HandPhase string

const (
	HandPhase_Start           HandPhase = "start"
	HandPhase_Antes           HandPhase = "antes"
	HandPhase_Hole            HandPhase = "hole"
	HandPhase_Continuation    HandPhase = "continuation"
	HandPhase_CommunityDeal   HandPhase = "community.deal"
	HandPhase_Discard         HandPhase = "discard"
	HandPhase_CommunityReveal HandPhase = "community.reveal"
	HandPhase_HandReveal      HandPhase = "hand.reveal"
	HandPhase_Decision        HandPhase = "decision"
	HandPhase_Payout          HandPhase = "payout"
	HandPhase_Ended           HandPhase = "ended"
)

// Hand represents a hand of poker being played
type Hand struct {
	ID             string
	Phase          HandPhase
	TableID        string
	Players        []Player
	Deck           cards.Stack
	CommunityCards cards.Stack
	HoleCards      map[string]cards.Stack
	Pot            int
	Events         []Event
	TableRules     TableRules
	StartedAt      time.Time
	// New fields for tracking bets
	AntesPaid        map[string]int  // Maps player IDs to ante amounts
	ContinuationBets map[string]int  // Maps player IDs to continuation bet amounts
	DiscardCosts     map[string]int  // Maps player IDs to discard costs
	ActivePlayers    map[string]bool // Maps player IDs to active status (still in the hand)
	CurrentBettor    string          // ID of player who should act next
	ButtonPosition   int             // Index of button player in the Players slice

}

func (h *Hand) IsPlayerTheCurrentBettor(playerID string) bool {
	return h.CurrentBettor == playerID
}

func (h *Hand) IsInPhase(phase HandPhase) bool {
	return h.Phase == phase
}

func (h *Hand) HasEnded() bool {
	return h.IsInPhase(HandPhase_Ended)
}

func (h *Hand) TransitionToAntesPhase() {
	if !h.IsInPhase(HandPhase_Start) {
		return
	}

	h.Phase = HandPhase_Antes

	// Record event for phase transition
	h.Events = append(h.Events, Event{
		Type:      "phase_transition",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"phase":      string(HandPhase_Antes),
			"ante_value": h.TableRules.AnteValue,
		},
	})

	// The actual ante collection would happen in the game loop,
	// giving each player the specified timeout to respond.
	// Starting from the player left of the dealer (would need dealer position tracking)
	// If a player doesn't respond within the timeout, they would be folded automatically
}

func (h *Hand) TransitionToHolePhase() {
	if !h.IsInPhase(HandPhase_Antes) {
		return
	}

	h.Phase = HandPhase_Hole
}

func (h *Hand) TransitionToContinuationPhase() {
	if !h.IsInPhase(HandPhase_Hole) {
		return
	}

	h.Phase = HandPhase_Continuation
}

func (h *Hand) TransitionToCommunityDealPhase() {
	if !h.IsInPhase(HandPhase_Continuation) {
		return
	}

	h.Phase = HandPhase_CommunityDeal
}

func (h *Hand) TransitionToCommunityRevealPhase() {
	if !h.IsInPhase(HandPhase_CommunityDeal) {
		return
	}

	h.Phase = HandPhase_CommunityReveal
}

func (h *Hand) TransitionToDiscardPhase() {
	if !h.IsInPhase(HandPhase_CommunityDeal) {
		return
	}

	h.Phase = HandPhase_Discard
}

func (h *Hand) TransitionToHandRevealPhase() {
	if !h.IsInPhase(HandPhase_CommunityReveal) {
		return
	}

	h.Phase = HandPhase_HandReveal
}

func (h *Hand) TransitionToDecisionPhase() {
	if !h.IsInPhase(HandPhase_HandReveal) {
		return
	}

	h.Phase = HandPhase_Decision
}

func (h *Hand) TransitionToPayoutPhase() {
	if !h.IsInPhase(HandPhase_Decision) {
		return
	}

	h.Phase = HandPhase_Payout
}

func (h *Hand) TransitionToEndedPhase() {
	if !h.IsInPhase(HandPhase_Payout) {
		return
	}

	h.Phase = HandPhase_Ended
}

// PlayerPlacesAnte records a player placing an ante
func (h *Hand) PlayerPlacesAnte(playerID string, amount int) error {
	// Check if in the correct phase
	if !h.IsInPhase(HandPhase_Antes) {
		return errors.New("not in antes phase")
	}

	// Check if it's the player's turn to act
	if h.CurrentBettor != playerID {
		return errors.New("not this player's turn to act")
	}

	// Check if player already paid ante
	if _, paid := h.AntesPaid[playerID]; paid {
		return errors.New("player already paid ante")
	}

	// Record the ante
	h.AntesPaid[playerID] = amount
	h.Pot += amount

	// Add event
	h.Events = append(h.Events, Event{
		Type:      "ante_placed",
		PlayerID:  playerID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"amount": amount,
		},
	})

	// Find next player to act
	h.CurrentBettor = h.getNextActiveBettor(playerID)

	// Check if all antes have been paid
	if len(h.AntesPaid) == len(h.ActivePlayers) {
		h.TransitionToHolePhase()
		// Reset CurrentBettor for next phase
		h.CurrentBettor = h.getPlayerLeftOfButton() // Implement this method
	}

	return nil
}

// PlayerPlacesContinuationBet records a player placing a continuation bet
func (h *Hand) PlayerPlacesContinuationBet(playerID string, amount int) error {
	// Check if in the correct phase
	if !h.IsInPhase(HandPhase_Continuation) {
		return errors.New("not in continuation bet phase")
	}

	// Check if it's the player's turn to act
	if h.CurrentBettor != playerID {
		return errors.New("not this player's turn to act")
	}

	// Check if player already made decision
	if _, decided := h.ContinuationBets[playerID]; decided {
		return errors.New("player already made continuation bet decision")
	}

	// Record the bet
	h.ContinuationBets[playerID] = amount
	h.Pot += amount

	// Add event
	h.Events = append(h.Events, Event{
		Type:      "continuation_bet_placed",
		PlayerID:  playerID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"amount": amount,
		},
	})

	// Find next player to act
	h.CurrentBettor = h.getNextActiveBettor(playerID)

	// Check if all continuation bets are in
	allDecided := true
	for playerID := range h.ActivePlayers {
		if _, decided := h.ContinuationBets[playerID]; !decided {
			allDecided = false
			break
		}
	}

	if allDecided {
		h.TransitionToCommunityDealPhase()
	}

	return nil
}

func (h *Hand) IsPlayerActive(playerID string) bool {
	return h.ActivePlayers[playerID]
}

func (h *Hand) SetPlayerAsActive(playerID string) {
	h.ActivePlayers[playerID] = true
}

func (h *Hand) SetPlayerAsInactive(playerID string) {
	h.ActivePlayers[playerID] = false
}

// PlayerFolds handles a player folding
func (h *Hand) PlayerFolds(playerID string) error {
	// Check if player is active
	if !h.IsPlayerActive(playerID) {
		return errors.New("player is not active in this hand")
	}

	// Check if it's appropriate phase for folding (continuation or discard)
	if !h.IsInPhase(HandPhase_Continuation) && !h.IsInPhase(HandPhase_Discard) {
		return errors.New("cannot fold in current phase")
	}

	// Mark player as inactive
	h.SetPlayerAsInactive(playerID)

	// Add event
	h.Events = append(h.Events, Event{
		Type:      "player_folded",
		PlayerID:  playerID,
		Timestamp: time.Now(),
	})

	// If current bettor folded, move to next player
	if h.IsPlayerTheCurrentBettor(playerID) {
		h.CurrentBettor = h.getNextActiveBettor(playerID)
	}

	// Check if only one player remains
	activePlayers := 0
	var lastActivePlayer string
	for id, active := range h.ActivePlayers {
		if active {
			activePlayers++
			lastActivePlayer = id
		}
	}

	if activePlayers == 1 {
		// Hand is over, last active player wins
		h.handleSinglePlayerWin(lastActivePlayer)
	}

	return nil
}

func (h *Hand) PlayerDiscardsCard(playerID string, card cards.Card) error {

	return nil
}

// handleSinglePlayerWin handles case where only one player remains
func (h *Hand) handleSinglePlayerWin(playerID string) {
	// Skip to the payout phase directly
	h.Phase = HandPhase_Payout

	// Add event for single winner
	h.Events = append(h.Events, Event{
		Type:      "single_winner",
		PlayerID:  playerID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"pot_amount": h.Pot,
		},
	})

	// Handle payout logic
	h.Payout()

	// End the hand
	h.TransitionToEndedPhase()
}

func (h *Hand) Payout() error {
	// Check if in the correct phase
	if !h.IsInPhase(HandPhase_Payout) {
		return errors.New("not in payout phase")
	}

	// Payout logic
	// ...

	// Transition
	h.TransitionToEndedPhase()

	return nil
}

// getPlayerLeftOfButton returns the player ID to the left of the button
func (h *Hand) getPlayerLeftOfButton() string {
	if len(h.Players) == 0 {
		return ""
	}

	pos := (h.ButtonPosition + 1) % len(h.Players)
	return h.Players[pos].ID
}

// CanPlayerActNow checks if a given player can act right now
func (h *Hand) CanPlayerActNow(playerID string) bool {
	// First check if player is active
	if !h.IsPlayerActive(playerID) {
		return false
	}

	// Check if it's this player's turn to act
	return h.IsPlayerTheCurrentBettor(playerID)
}

// IsWaitingForBet checks if the hand is waiting for a player to bet
func (h *Hand) IsWaitingForBet() bool {
	// Check the current phase
	switch h.Phase {
	case HandPhase_Antes:
		return len(h.AntesPaid) < len(h.ActivePlayers)
	case HandPhase_Continuation:
		for playerID, active := range h.ActivePlayers {
			if active {
				if _, decided := h.ContinuationBets[playerID]; !decided {
					return true
				}
			}
		}
		return false
	case HandPhase_Discard:
		// Implementation depends on your discard phase rules
		return h.CurrentBettor != ""
	default:
		return false
	}
}

// AddTable adds a new table to the game
func (g *Game) AddTable(table Table) error {
	if g.tables == nil {
		g.tables = make(map[string]*Table)
	}

	if _, exists := g.tables[table.ID]; exists {
		return errors.New("table with this ID already exists")
	}

	table.Status = TableStatusWaiting
	g.tables[table.ID] = &table

	return nil
}

// GetTable retrieves a table by ID
func (g *Game) GetTable(tableID string) (*Table, error) {
	if g.tables == nil {
		return nil, errors.New("no tables exist")
	}

	table, exists := g.tables[tableID]
	if !exists {
		return nil, errors.New("table not found")
	}

	return table, nil
}

// listen registers a message listener
func (g *Game) listen(listener func(Message)) {
	g.listeners = append(g.listeners, listener)
}

// publish sends a message to all listeners
func (g *Game) publish(msg Message) {
	for _, listener := range g.listeners {
		listener(msg)
	}
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

// StartPlaying starts the table if there are enough players
func (t *Table) StartPlaying() error {
	if len(t.Players) < 2 {
		return errors.New("need at least 2 players to start")
	}

	if t.Status != TableStatusWaiting {
		return errors.New("table must be in waiting status to start playing")
	}

	t.Status = TableStatusPlaying

	t.publish(TableStartedMessage{TableID: t.ID})

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

	t.publish(TableStartedMessage{TableID: t.ID})

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

// getFirstActivePlayerAfterButton returns the ID of the first active player after the button
func (t *Table) getFirstActivePlayerAfterButton(buttonPos int) string {
	if len(t.Players) == 0 {
		return ""
	}

	pos := (buttonPos + 1) % len(t.Players)
	// Find the first active player
	for i := 0; i < len(t.Players); i++ {
		playerID := t.Players[pos].ID
		if t.ActiveHand.ActivePlayers[playerID] {
			return playerID
		}
		pos = (pos + 1) % len(t.Players)
	}
	return ""
}

// getNextActiveBettor returns the next active player who should bet
func (h *Hand) getNextActiveBettor(currentBettorID string) string {
	if len(h.Players) == 0 {
		return ""
	}

	// Find current bettor's position
	currentPos := -1
	for i, player := range h.Players {
		if player.ID == currentBettorID {
			currentPos = i
			break
		}
	}

	if currentPos == -1 {
		return ""
	}

	// Find next active player
	pos := (currentPos + 1) % len(h.Players)
	for i := 0; i < len(h.Players); i++ {
		if pos == currentPos {
			break // We've come full circle
		}

		playerID := h.Players[pos].ID
		if h.ActivePlayers[playerID] {
			return playerID
		}
		pos = (pos + 1) % len(h.Players)
	}

	return ""
}

func (t *Table) setActiveHand(hand *Hand) {
	t.ActiveHand = hand
}

func (t *Table) publish(msg Message) {
	// This would be implemented with a proper message queue in a real system
}

// TableAddedMessage represents a message when a table is added
type TableAddedMessage struct {
	Table Table
}

func (m TableAddedMessage) MessageName() string {
	return "TableAdded"
}

// PlayerAddedMessage represents a message when a player is added to a table
type PlayerAddedMessage struct {
	TableID  string
	PlayerID string
}

func (m PlayerAddedMessage) MessageName() string {
	return "PlayerAdded"
}

// TableStartedMessage represents a message when a table starts playing
type TableStartedMessage struct {
	TableID string
}

func (m TableStartedMessage) MessageName() string {
	return "TableStarted"
}
