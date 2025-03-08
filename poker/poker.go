package poker

import (
	"errors"
	"fmt"
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
		return errors.New("need at least  players to start")
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
	t.Hands = append(t.Hands, *hand)
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

// InitializeHand initializes a new hand with a fresh deck and activates all players
func (h *Hand) InitializeHand() {
	// Initialize a new shuffled deck
	h.Deck = cards.NewDeck52()
	h.Deck.Shuffle()

	// Initialize the community cards as empty
	h.CommunityCards = []cards.Card{}

	// Initialize hole cards map for each player
	h.HoleCards = make(map[string]cards.Stack)

	// Set all players to active at the start of the hand
	h.ActivePlayers = make(map[string]bool)
	for _, player := range h.Players {
		h.ActivePlayers[player.ID] = true
		h.HoleCards[player.ID] = []cards.Card{}
	}

	// Initialize betting maps
	h.AntesPaid = make(map[string]int)
	h.ContinuationBets = make(map[string]int)
	h.DiscardCosts = make(map[string]int)

	// Set the current bettor to the player left of the button
	h.CurrentBettor = h.getPlayerLeftOfButton()

	// Record event for hand initialization
	h.Events = append(h.Events, Event{
		Type:      "hand_initialized",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"player_count": len(h.Players),
			"button_pos":   h.ButtonPosition,
		},
	})
}

// BurnCard removes the top card from the deck without revealing it
func (h *Hand) BurnCard() error {
	if len(h.Deck) == 0 {
		return errors.New("no cards left in deck to burn")
	}

	// Remove top card without using it
	h.Deck = h.Deck[1:]

	// Record event
	h.Events = append(h.Events, Event{
		Type:      "card_burned",
		Timestamp: time.Now(),
	})

	return nil
}

// PrintState is a debugging function to print the current state of the hand in a string, over multiple lines and in a human-readable structured format
func (h *Hand) PrintState() string {
	output := "Hand State:\n"
	output += "--------------------------------------------------\n"
	output += "ID: " + h.ID + "\n"
	output += "Phase: " + string(h.Phase) + "\n"
	output += "TableID: " + h.TableID + "\n"
	output += "Pot: " + fmt.Sprint(h.Pot) + "\n"
	output += "CurrentBettor: " + h.CurrentBettor + "\n"
	output += "ButtonPosition: " + fmt.Sprint(h.ButtonPosition) + "\n"
	output += "\n"

	output += "Players:\n"
	for _, player := range h.Players {
		output += "  - ID: " + player.ID + ", Name: " + player.Name + ", Chips: " + fmt.Sprint(player.Chips) + "\n"
	}
	output += "\n"

	output += "Active Players:\n"
	for playerID, active := range h.ActivePlayers {
		output += "  - ID: " + playerID + ", Active: " + fmt.Sprint(active) + "\n"
	}
	output += "\n"

	output += "Hole Cards:\n"
	for playerID, cards := range h.HoleCards {
		output += "  - Player: " + playerID + ", Cards: " + cards.String() + "\n"
	}
	output += "\n"

	output += "Community Cards: " + h.CommunityCards.String() + "\n"
	output += "\n"

	output += "Antes Paid:\n"
	for playerID, amount := range h.AntesPaid {
		output += "  - Player: " + playerID + ", Amount: " + fmt.Sprint(amount) + "\n"
	}
	output += "\n"

	output += "Continuation Bets:\n"
	for playerID, amount := range h.ContinuationBets {
		output += "  - Player: " + playerID + ", Amount: " + fmt.Sprint(amount) + "\n"
	}
	output += "\n"

	output += "Discard Costs:\n"
	for playerID, amount := range h.DiscardCosts {
		output += "  - Player: " + playerID + ", Amount: " + fmt.Sprint(amount) + "\n"
	}
	output += "\n"

	output += "Events:\n"
	for _, event := range h.Events {
		output += "  - Type: " + event.Type + ", PlayerID: " + event.PlayerID + ", Timestamp: " + event.Timestamp.String() + "\n"
	}
	output += "\n"

	output += "--------------------------------------------------\n"

	return output
}

// DealHoleCards deals two cards to each active player, one card at a time
func (h *Hand) DealHoleCards() error {
	if !h.IsInPhase(HandPhase_Hole) {
		return errors.New("not in hole card phase")
	}

	// First round of dealing (first card to each player)
	for i := 0; i < len(h.Players); i++ {
		// Start with player to left of button and go around
		pos := (h.ButtonPosition + 1 + i) % len(h.Players)
		playerID := h.Players[pos].ID

		// Only deal to active players
		if h.ActivePlayers[playerID] {
			if len(h.Deck) == 0 {
				return errors.New("no cards left in deck")
			}

			// Deal one card
			card := h.Deck[0]
			h.Deck = h.Deck[1:] // Remove card from deck
			h.HoleCards[playerID] = append(h.HoleCards[playerID], card)
		}
	}

	// Second round of dealing (second card to each player)
	for i := 0; i < len(h.Players); i++ {
		pos := (h.ButtonPosition + 1 + i) % len(h.Players)
		playerID := h.Players[pos].ID

		// Only deal to active players
		if h.ActivePlayers[playerID] {
			if len(h.Deck) == 0 {
				return errors.New("no cards left in deck")
			}

			// Deal one card
			card := h.Deck[0]
			h.Deck = h.Deck[1:] // Remove card from deck
			h.HoleCards[playerID] = append(h.HoleCards[playerID], card)
		}
	}

	// Record event for hole cards dealt
	h.Events = append(h.Events, Event{
		Type:      "hole_cards_dealt",
		Timestamp: time.Now(),
	})

	return nil
}

// DealCommunityCard deals a single community card
func (h *Hand) DealCommunityCard() error {
	if !h.IsInPhase(HandPhase_CommunityDeal) && !h.IsInPhase(HandPhase_CommunityReveal) {
		return errors.New("not in community card dealing phase")
	}

	if len(h.Deck) == 0 {
		return errors.New("no cards left in deck")
	}

	// Deal one card
	card := h.Deck[0]
	h.Deck = h.Deck[1:]
	h.CommunityCards = append(h.CommunityCards, card)

	// Record event
	h.Events = append(h.Events, Event{
		Type:      "community_card_dealt",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"card_index": len(h.CommunityCards) - 1,
		},
	})

	return nil
}

// PlayerDiscardsCard allows a player to discard and replace a specific hole card
func (h *Hand) PlayerDiscardsCard(playerID string, cardIndex int) error {
	// Check if in the correct phase
	if !h.IsInPhase(HandPhase_Discard) {
		return errors.New("not in discard phase")
	}

	// Check if player is active
	if !h.IsPlayerActive(playerID) {
		return errors.New("player is not active in this hand")
	}

	// Check if it's this player's turn to act
	if !h.IsPlayerTheCurrentBettor(playerID) {
		return errors.New("not this player's turn to act")
	}

	// Check if the card index is valid
	if cardIndex < 0 || cardIndex >= len(h.HoleCards[playerID]) {
		return errors.New("invalid card index")
	}

	// Check if player has already paid discard cost
	if _, paid := h.DiscardCosts[playerID]; !paid {
		return errors.New("player has not paid discard cost")
	}

	// Remove the card from player's hand
	_ = h.HoleCards[playerID][cardIndex]
	h.HoleCards[playerID] = append(h.HoleCards[playerID][:cardIndex], h.HoleCards[playerID][cardIndex+1:]...)

	// Deal a replacement card
	if len(h.Deck) == 0 {
		return errors.New("no cards left in deck")
	}

	newCard := h.Deck[0]
	h.Deck = h.Deck[1:]
	h.HoleCards[playerID] = append(h.HoleCards[playerID], newCard)

	// Record event
	h.Events = append(h.Events, Event{
		Type:      "card_discarded",
		PlayerID:  playerID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"discarded_card_index": cardIndex,
		},
	})

	// Move to next player
	h.CurrentBettor = h.getNextActiveBettor(playerID)

	return nil
}

// PlayerPaysDiscardCost records a player paying to discard a card
func (h *Hand) PlayerPaysDiscardCost(playerID string) error {
	// Check if in the correct phase
	if !h.IsInPhase(HandPhase_Discard) {
		return errors.New("not in discard phase")
	}

	// Check if player is active
	if !h.IsPlayerActive(playerID) {
		return errors.New("player is not active in this hand")
	}

	// Check if it's this player's turn to act
	if !h.IsPlayerTheCurrentBettor(playerID) {
		return errors.New("not this player's turn to act")
	}

	// Check if player has already paid
	if _, paid := h.DiscardCosts[playerID]; paid {
		return errors.New("player already paid discard cost")
	}

	// Record the discard cost
	cost := h.TableRules.DiscardCostValue
	h.DiscardCosts[playerID] = cost
	h.Pot += cost

	// Record event
	h.Events = append(h.Events, Event{
		Type:      "discard_cost_paid",
		PlayerID:  playerID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"amount": cost,
		},
	})

	return nil
}

// EvaluateHands evaluates all active players' hands and determines the winner(s)
func (h *Hand) EvaluateHands() ([]HandComparisonResult, error) {
	if !h.IsInPhase(HandPhase_HandReveal) {
		return nil, errors.New("not in hand reveal phase")
	}

	// Create a map of player ID to their combined hole and community cards
	playerCards := make(map[string]cards.Stack)
	for playerID, holeCards := range h.HoleCards {
		if h.IsPlayerActive(playerID) {
			// Combine hole cards and community cards
			combinedCards := append(cards.Stack{}, holeCards...)
			combinedCards = append(combinedCards, h.CommunityCards...)
			playerCards[playerID] = combinedCards
		}
	}

	// Use the hand evaluator to determine the best hand for each player
	// (This assumes we have access to the hands package)
	results, err := h.comparePlayerHands(playerCards)
	if err != nil {
		return nil, err
	}

	// Record event with the results
	resultData := make([]map[string]interface{}, len(results))
	for i, result := range results {
		resultData[i] = map[string]interface{}{
			"player_id":   result.PlayerID,
			"hand_rank":   result.HandRank,
			"is_winner":   result.IsWinner,
			"place_index": result.PlaceIndex,
		}
	}

	h.Events = append(h.Events, Event{
		Type:      "hands_evaluated",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"results": resultData,
		},
	})

	return results, nil
}

// Payout distributes the pot to the winner(s)
func (h *Hand) Payout() error {
	// Check if in the correct phase
	if !h.IsInPhase(HandPhase_Payout) {
		return errors.New("not in payout phase")
	}

	// Get hand evaluation results
	results, err := h.EvaluateHands()
	if err != nil {
		// If we can't evaluate hands, look for single remaining player
		var winnerID string
		winnerCount := 0

		for playerID, active := range h.ActivePlayers {
			if active {
				winnerID = playerID
				winnerCount++
			}
		}

		if winnerCount == 1 {
			// Single player remaining, they win by default
			return h.payoutToSingleWinner(winnerID)
		}

		return err
	}

	// Find winners
	var winners []string
	for _, result := range results {
		if result.IsWinner {
			winners = append(winners, result.PlayerID)
		}
	}

	// If no winners found (shouldn't happen), return error
	if len(winners) == 0 {
		return errors.New("no winners found")
	}

	// Calculate the amount each winner gets (split pot)
	winAmount := h.Pot / len(winners)
	remainder := h.Pot % len(winners)

	// Distribute the pot
	for _, winnerID := range winners {
		// Find player index
		playerIndex := -1
		for i, p := range h.Players {
			if p.ID == winnerID {
				playerIndex = i
				break
			}
		}

		if playerIndex != -1 {
			// Add winnings to player chips
			h.Players[playerIndex].Chips += winAmount

			// Record event
			h.Events = append(h.Events, Event{
				Type:      "pot_awarded",
				PlayerID:  winnerID,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"amount": winAmount,
				},
			})
		}
	}

	// If there's a remainder due to uneven split, give it to first winner
	// (usually the player closest to the left of the dealer)
	if remainder > 0 && len(winners) > 0 {
		playerIndex := -1
		for i, p := range h.Players {
			if p.ID == winners[0] {
				playerIndex = i
				break
			}
		}

		if playerIndex != -1 {
			h.Players[playerIndex].Chips += remainder
		}
	}

	// Empty the pot
	h.Pot = 0

	// Transition to ended state
	h.TransitionToEndedPhase()

	return nil
}

// payoutToSingleWinner distributes the pot to a single winner
func (h *Hand) payoutToSingleWinner(winnerID string) error {
	// Find player index
	playerIndex := -1
	for i, p := range h.Players {
		if p.ID == winnerID {
			playerIndex = i
			break
		}
	}

	if playerIndex == -1 {
		return errors.New("winner not found among players")
	}

	// Add pot to player's chips
	h.Players[playerIndex].Chips += h.Pot

	// Record event
	h.Events = append(h.Events, Event{
		Type:      "pot_awarded",
		PlayerID:  winnerID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"amount": h.Pot,
			"reason": "last_player_standing",
		},
	})

	// Empty the pot
	h.Pot = 0

	// Transition to ended state
	h.TransitionToEndedPhase()

	return nil
}
