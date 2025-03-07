package game

import (
	"errors"
	"fmt"
	"time"

	"github.com/lazharichir/poker/cards"
	"github.com/lazharichir/poker/domain"
	"github.com/lazharichir/poker/domain/events"
)

// GamePhase represents the current phase of a hand
type GamePhase string

const (
	PhaseNotStarted      GamePhase = "not_started"
	PhaseAnteCollection  GamePhase = "ante_collection"
	PhaseContinuationBet GamePhase = "continuation_bet"
	PhaseInitialDeal     GamePhase = "initial_deal"
	PhaseDiscard         GamePhase = "discard"
	PhaseCardSelection   GamePhase = "card_selection"
	PhaseHandEvaluation  GamePhase = "hand_evaluation"
	PhaseHandCompleted   GamePhase = "hand_completed"
)

// TableEngine handles the game logic and generates domain events
type TableEngine struct {
	eventStore           events.EventStore
	tableState           *domain.Table
	phase                GamePhase
	deck                 []cards.Card
	activePlayers        []string // IDs of players still in the hand
	currentPlayerTurnIdx int      // Index in activePlayers for current turn
}

// NewTableEngine creates a new table engine with the given event store
func NewTableEngine(eventStore events.EventStore, tableID string) (*TableEngine, error) {
	// Load existing events to rebuild state
	tableEvents, err := eventStore.LoadEvents(tableID)
	if err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}
	_ = tableEvents

	// Create a default table as starting point
	table := &domain.Table{
		ID:      tableID,
		Players: make(map[string]*domain.Player),
	}

	// Apply events to rebuild state (this would be handled by event handlers in Phase 5)
	// For now, we'll just create a new table engine with default values

	return &TableEngine{
		eventStore:    eventStore,
		tableState:    table,
		phase:         PhaseNotStarted,
		deck:          []cards.Card{},
		activePlayers: []string{},
	}, nil
}

// StartHand starts a new hand at the table
func (te *TableEngine) StartHand() error {
	if te.phase != PhaseNotStarted {
		return errors.New("cannot start a new hand while another is in progress")
	}

	if len(te.tableState.Players) < 2 {
		return errors.New("need at least 2 players to start a hand")
	}

	// Prepare a new deck
	te.deck = cards.ShuffleDeck(cards.NewDeck())

	// Choose button position (in a real implementation, rotate from previous hand)
	// For now, pick the first player as the button
	playerIDs := make([]string, 0, len(te.tableState.Players))
	for id := range te.tableState.Players {
		playerIDs = append(playerIDs, id)
	}
	buttonPlayerID := playerIDs[0]
	te.tableState.ButtonPlayerID = buttonPlayerID

	// Initialize active players (everyone starts active)
	te.activePlayers = playerIDs

	// Create and store HandStarted event
	handStartedEvent := events.HandStarted{
		TableID:        te.tableState.ID,
		ButtonPlayerID: buttonPlayerID,
		AnteAmount:     te.tableState.Ante,
		PlayerIDs:      playerIDs,
	}

	if err := te.eventStore.Append(handStartedEvent); err != nil {
		return fmt.Errorf("failed to append HandStarted event: %w", err)
	}

	// Move to ante collection phase
	te.phase = PhaseAnteCollection
	return nil
}

// PlaceAnte handles a player placing an ante bet
func (te *TableEngine) PlaceAnte(playerID string) error {
	if te.phase != PhaseAnteCollection {
		return errors.New("not in ante collection phase")
	}

	player, exists := te.tableState.Players[playerID]
	if !exists {
		return fmt.Errorf("player %s not found at table", playerID)
	}

	if player.Chips < te.tableState.Ante {
		return errors.New("player doesn't have enough chips for ante")
	}

	// Create AntePlacedByPlayer event
	antePlacedEvent := events.AntePlacedByPlayer{
		TableID:  te.tableState.ID,
		PlayerID: playerID,
		Amount:   te.tableState.Ante,
	}

	if err := te.eventStore.Append(antePlacedEvent); err != nil {
		return fmt.Errorf("failed to append AntePlacedByPlayer event: %w", err)
	}

	// Check if all active players have placed antes
	// In a real implementation, this would be tracked via event handlers
	// For now, we'll assume all antes are collected when this function is called for all players
	// and the last ante triggers moving to the next phase

	// If this was the last player to place ante, deal hole cards
	if playerID == te.activePlayers[len(te.activePlayers)-1] {
		return te.dealHoleCards()
	}

	return nil
}

// dealHoleCards deals two hole cards to each player
func (te *TableEngine) dealHoleCards() error {
	te.phase = PhaseInitialDeal

	// Deal 2 cards to each active player
	for _, playerID := range te.activePlayers {
		// Deal first card
		card1, remainingDeck := cards.DealCard(te.deck)
		te.deck = remainingDeck

		event1 := events.PlayerHoleCardDealt{
			TableID:  te.tableState.ID,
			PlayerID: playerID,
			Card:     card1,
		}
		if err := te.eventStore.Append(event1); err != nil {
			return fmt.Errorf("failed to append PlayerHoleCardDealt event: %w", err)
		}

		// Deal second card
		card2, remainingDeck := cards.DealCard(te.deck)
		te.deck = remainingDeck

		event2 := events.PlayerHoleCardDealt{
			TableID:  te.tableState.ID,
			PlayerID: playerID,
			Card:     card2,
		}
		if err := te.eventStore.Append(event2); err != nil {
			return fmt.Errorf("failed to append PlayerHoleCardDealt event: %w", err)
		}
	}

	// Move to continuation bet phase
	te.phase = PhaseContinuationBet
	te.currentPlayerTurnIdx = 0 // Start with first active player
	return nil
}

// PlaceContinuationBet handles a player placing a continuation bet
func (te *TableEngine) PlaceContinuationBet(playerID string) error {
	if te.phase != PhaseContinuationBet {
		return errors.New("not in continuation bet phase")
	}

	if te.activePlayers[te.currentPlayerTurnIdx] != playerID {
		return errors.New("not your turn")
	}

	player, exists := te.tableState.Players[playerID]
	if !exists {
		return fmt.Errorf("player %s not found at table", playerID)
	}

	continuationBetAmount := te.tableState.Ante * te.tableState.ContinuationBetMultiplier
	if player.Chips < continuationBetAmount {
		return errors.New("player doesn't have enough chips for continuation bet")
	}

	// Create ContinuationBetPlaced event
	continuationBetEvent := events.ContinuationBetPlaced{
		TableID:  te.tableState.ID,
		PlayerID: playerID,
		Amount:   continuationBetAmount,
	}

	if err := te.eventStore.Append(continuationBetEvent); err != nil {
		return fmt.Errorf("failed to append ContinuationBetPlaced event: %w", err)
	}

	// Move to next player
	te.currentPlayerTurnIdx++

	// Check if all active players have acted
	if te.currentPlayerTurnIdx >= len(te.activePlayers) {
		return te.dealCommunityCards()
	}

	return nil
}

// Fold handles a player folding
func (te *TableEngine) Fold(playerID string) error {
	if te.phase != PhaseContinuationBet {
		return errors.New("can only fold during continuation bet phase")
	}

	if te.activePlayers[te.currentPlayerTurnIdx] != playerID {
		return errors.New("not your turn")
	}

	// Create PlayerFolded event
	foldedEvent := events.PlayerFolded{
		TableID:  te.tableState.ID,
		PlayerID: playerID,
	}

	if err := te.eventStore.Append(foldedEvent); err != nil {
		return fmt.Errorf("failed to append PlayerFolded event: %w", err)
	}

	// Remove player from active players
	// In event sourcing, this would be handled by an event handler
	// For now, we'll update the list directly
	te.activePlayers = append(te.activePlayers[:te.currentPlayerTurnIdx], te.activePlayers[te.currentPlayerTurnIdx+1:]...)

	// Check if only one player remains (everyone else folded)
	if len(te.activePlayers) == 1 {
		// Create HandCompleted event with single winner
		winnerID := te.activePlayers[0]
		handCompletedEvent := events.HandCompleted{
			TableID:       te.tableState.ID,
			FirstPlaceID:  winnerID,
			FirstPrize:    te.tableState.Pot,
			SecondPlaceID: "",
			SecondPrize:   0,
		}

		if err := te.eventStore.Append(handCompletedEvent); err != nil {
			return fmt.Errorf("failed to append HandCompleted event: %w", err)
		}

		te.phase = PhaseHandCompleted
		return nil
	}

	// If there are players left and this was the last player's turn, deal community cards
	if te.currentPlayerTurnIdx >= len(te.activePlayers) {
		return te.dealCommunityCards()
	}

	return nil
}

// dealCommunityCards deals the 8 community cards
func (te *TableEngine) dealCommunityCards() error {
	te.phase = PhaseInitialDeal

	// Deal 8 community cards
	communityCards, remainingDeck := cards.DealCards(te.deck, 8)
	te.deck = remainingDeck

	// Store community cards in table state
	te.tableState.CommunityCards = communityCards

	// Create CommunityCardsDealt event
	communityCardsEvent := events.CommunityCardsDealt{
		TableID: te.tableState.ID,
		Cards:   communityCards,
	}

	if err := te.eventStore.Append(communityCardsEvent); err != nil {
		return fmt.Errorf("failed to append CommunityCardsDealt event: %w", err)
	}

	// Move to discard phase
	te.phase = PhaseDiscard
	te.currentPlayerTurnIdx = 0
	return nil
}

// DiscardCard handles a player discarding a community card
func (te *TableEngine) DiscardCard(playerID string, cardIndex int) error {
	if te.phase != PhaseDiscard {
		return errors.New("not in discard phase")
	}

	if te.activePlayers[te.currentPlayerTurnIdx] != playerID {
		return errors.New("not your turn")
	}

	if cardIndex < 0 || cardIndex >= len(te.tableState.CommunityCards) {
		return errors.New("invalid card index")
	}

	player, exists := te.tableState.Players[playerID]
	if !exists {
		return fmt.Errorf("player %s not found at table", playerID)
	}

	// Calculate discard fee based on table configuration
	var discardFee int
	switch te.tableState.DiscardCostType {
	case "fixed":
		discardFee = te.tableState.DiscardCostValue
	case "ante_multiple":
		discardFee = te.tableState.Ante * te.tableState.DiscardCostValue
	case "bet_multiple":
		discardFee = (te.tableState.Ante + (te.tableState.Ante * te.tableState.ContinuationBetMultiplier)) * te.tableState.DiscardCostValue
	case "pot_multiple":
		discardFee = te.tableState.Pot * te.tableState.DiscardCostValue / 100 // Assuming value is in percentage
	default:
		return errors.New("invalid discard cost type")
	}

	if player.Chips < discardFee {
		return errors.New("player doesn't have enough chips for discard fee")
	}

	// Get the card to discard
	discardedCard := te.tableState.CommunityCards[cardIndex]

	// Create CardDiscarded event
	discardEvent := events.CardDiscarded{
		TableID:    te.tableState.ID,
		PlayerID:   playerID,
		Card:       discardedCard,
		DiscardFee: discardFee,
	}

	if err := te.eventStore.Append(discardEvent); err != nil {
		return fmt.Errorf("failed to append CardDiscarded event: %w", err)
	}

	// Remove the discarded card
	// In event sourcing, this would be handled by event handlers
	// For now, we'll update the state directly
	te.tableState.CommunityCards = append(
		te.tableState.CommunityCards[:cardIndex],
		te.tableState.CommunityCards[cardIndex+1:]...,
	)

	// Move to next player
	te.currentPlayerTurnIdx++

	// Check if all active players have had a chance to discard
	if te.currentPlayerTurnIdx >= len(te.activePlayers) {
		te.phase = PhaseCardSelection
		// In a real implementation, we would start timers for the card selection waves
		go te.runCardSelectionPhase()
	}

	return nil
}

// SkipDiscard handles a player choosing not to discard a card
func (te *TableEngine) SkipDiscard(playerID string) error {
	if te.phase != PhaseDiscard {
		return errors.New("not in discard phase")
	}

	if te.activePlayers[te.currentPlayerTurnIdx] != playerID {
		return errors.New("not your turn")
	}

	// Move to next player
	te.currentPlayerTurnIdx++

	// Check if all active players have had a chance to discard
	if te.currentPlayerTurnIdx >= len(te.activePlayers) {
		te.phase = PhaseCardSelection
		// In a real implementation, we would start timers for the card selection waves
		go te.runCardSelectionPhase()
	}

	return nil
}

// runCardSelectionPhase handles the timed waves of card selection
func (te *TableEngine) runCardSelectionPhase() {
	// In a production system, this would be implemented with proper timers
	// and would handle the reveal of cards in waves

	// For now, we'll simulate it with sleep timers

	// Start card selection phase - all players can now select cards
	// Wave 1: First 3 cards are available immediately

	// Wait 5 seconds for Wave 2
	time.Sleep(5 * time.Second)
	// Wave 2: Next 3 cards become available

	// Wait 3 seconds for Wave 3
	time.Sleep(3 * time.Second)
	// Wave 3: Last 2 cards become available

	// Wait 2 seconds for selection to end
	time.Sleep(2 * time.Second)
	// End card selection phase

	// Evaluate hands for remaining players
	te.evaluateHands()
}

// SelectCommunityCard handles a player selecting a community card
func (te *TableEngine) SelectCommunityCard(playerID string, cardIndex int) error {
	if te.phase != PhaseCardSelection {
		return errors.New("not in card selection phase")
	}

	player, exists := te.tableState.Players[playerID]
	if !exists {
		return fmt.Errorf("player %s not found at table", playerID)
	}

	// Check if player is still active
	isActive := false
	for _, id := range te.activePlayers {
		if id == playerID {
			isActive = true
			break
		}
	}
	if !isActive {
		return errors.New("player is not active in this hand")
	}

	if cardIndex < 0 || cardIndex >= len(te.tableState.CommunityCards) {
		return errors.New("invalid card index")
	}

	// Check if player has already selected 3 cards
	if len(player.SelectedCommunityCards) >= 3 {
		return errors.New("player has already selected 3 community cards")
	}

	// Check if player has already selected this card
	selectedCard := te.tableState.CommunityCards[cardIndex]
	for _, card := range player.SelectedCommunityCards {
		if card.Equals(selectedCard) {
			return errors.New("player has already selected this card")
		}
	}

	// Create CommunityCardSelected event
	selectEvent := events.CommunityCardSelected{
		TableID:  te.tableState.ID,
		PlayerID: playerID,
		Card:     selectedCard,
	}

	if err := te.eventStore.Append(selectEvent); err != nil {
		return fmt.Errorf("failed to append CommunityCardSelected event: %w", err)
	}

	// Update player's selected cards
	// In event sourcing, this would be handled by event handlers
	// For now, we'll update the state directly
	player.SelectedCommunityCards = append(player.SelectedCommunityCards, selectedCard)

	return nil
}

// evaluateHands evaluates all player hands and determines winners
func (te *TableEngine) evaluateHands() {
	te.phase = PhaseHandEvaluation

	// In a real implementation, we would evaluate all active player hands
	// and determine 1st and 2nd place
	// For this example, we'll use a simplified approach

	// Assume we have determined winners (in reality this would use poker hand evaluation)
	var firstPlaceID, secondPlaceID string
	var firstPrize, secondPrize int

	if len(te.activePlayers) >= 1 {
		firstPlaceID = te.activePlayers[0]
		firstPrize = te.tableState.Pot * 80 / 100 // 80% of pot
	}

	if len(te.activePlayers) >= 2 {
		secondPlaceID = te.activePlayers[1]
		secondPrize = te.tableState.Pot * 20 / 100 // 20% of pot
	}

	// Create HandCompleted event
	handCompletedEvent := events.HandCompleted{
		TableID:       te.tableState.ID,
		FirstPlaceID:  firstPlaceID,
		FirstPrize:    firstPrize,
		SecondPlaceID: secondPlaceID,
		SecondPrize:   secondPrize,
	}

	// Append the event
	_ = te.eventStore.Append(handCompletedEvent) // Error handling omitted for brevity

	// Move to completed phase
	te.phase = PhaseHandCompleted
}
