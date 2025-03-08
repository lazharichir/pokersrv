package table

import (
	"sync"
	"testing"
	"time"

	"github.com/lazharichir/poker/cards"
	"github.com/lazharichir/poker/events"
	"github.com/lazharichir/poker/poker"
	"github.com/stretchr/testify/assert"
)

// mockEventStore is a simple implementation of events.EventStore for testing
type mockEventStore struct {
	events []events.Event
	mu     sync.Mutex
}

// TestFullGameFlow tests the entire game flow from start to finish
func TestFullGameFlow(t *testing.T) {
	// Set up test parameters
	tableID := "test-table-123"
	player1 := "player-1"
	player2 := "player-2"
	player3 := "player-3"
	players := []string{player1, player2, player3}

	// Create table rules
	rules := poker.TableRules{
		AnteValue:                 10,
		ContinuationBetMultiplier: 2,
		DiscardPhaseDuration:      5,
		DiscardCostType:           "fixed",
		DiscardCostValue:          5,
	}

	// Create event store and game loop
	eventStore := events.NewInMemoryEventStore()
	gameLoop := NewGameLoop(tableID, rules, eventStore)

	// Start the game loop with our players
	gameLoop.Start(players)
	defer gameLoop.Stop()

	// 1. Wait for transition to ante collection (the game should auto-transition)
	waitForState(t, gameLoop, GameStateAnteCollection)

	// Check that a HandStarted event was created
	hasHandStartedEvent := false
	for _, e := range eventStore.GetEvents() {
		if _, ok := e.(events.HandStarted); ok {
			hasHandStartedEvent = true
			break
		}
	}
	assert.True(t, hasHandStartedEvent, "HandStarted event should be published")

	// 2. Submit ante actions for all players
	for _, player := range players {
		gameLoop.SubmitAction(player, "place_ante", map[string]interface{}{
			"amount": rules.AnteValue,
		})
	}

	// Wait for transition to dealing hole cards
	waitForState(t, gameLoop, GameStateDealingHoleCards)

	// 3. The game should automatically transition to continuation bets
	waitForState(t, gameLoop, GameStateContinuationBets)

	// Check that hole cards were dealt
	holeCardEvents := 0
	for _, e := range eventStore.GetEvents() {
		if _, ok := e.(events.PlayerHoleCardDealt); ok {
			holeCardEvents++
		}
	}
	assert.Equal(t, len(players)*2, holeCardEvents, "Each player should receive 2 hole cards")

	// 4. Submit continuation bet actions for players 1 and 2, fold for player 3
	gameLoop.SubmitAction(player1, "place_continuation_bet", map[string]interface{}{
		"amount": rules.AnteValue * rules.ContinuationBetMultiplier,
	})

	gameLoop.SubmitAction(player2, "place_continuation_bet", map[string]interface{}{
		"amount": rules.AnteValue * rules.ContinuationBetMultiplier,
	})

	gameLoop.SubmitAction(player3, "fold", nil)

	// Wait for dealing community cards
	waitForState(t, gameLoop, GameStateDealingCommunity)

	// Check that continuation bet events were created and player3 folded
	hasContinuationBets := 0
	hasPlayerFolded := false
	for _, e := range eventStore.GetEvents() {
		if _, ok := e.(events.ContinuationBetPlaced); ok {
			hasContinuationBets++
		}
		if fold, ok := e.(events.PlayerFolded); ok && fold.PlayerID == player3 {
			hasPlayerFolded = true
		}
	}
	assert.Equal(t, 2, hasContinuationBets, "Two players should have placed continuation bets")
	assert.True(t, hasPlayerFolded, "Player 3 should have folded")

	// 5. Wait for community cards to be dealt and transition to discard phase
	waitForState(t, gameLoop, GameStateDiscardPhase)

	// Check community cards were dealt
	hasCommunityCards := false
	for _, e := range eventStore.GetEvents() {
		if _, ok := e.(events.CommunityCardsDealt); ok {
			hasCommunityCards = true
			break
		}
	}
	assert.True(t, hasCommunityCards, "Community cards should be dealt")

	// 6. Submit discard actions
	// Player 1 discards a card
	gameLoop.SubmitAction(player1, "discard_card", map[string]interface{}{
		"card": cards.Card{Suit: cards.Spades, Value: cards.Ten},
	})

	// Player 2 skips discard
	gameLoop.SubmitAction(player2, "skip_discard", nil)

	// Wait for wave 1 reveal
	waitForState(t, gameLoop, GameStateWave1Reveal)

	// Check that discard event was created
	hasDiscard := false
	for _, e := range eventStore.GetEvents() {
		if _, ok := e.(events.CardDiscarded); ok {
			hasDiscard = true
			break
		}
	}
	assert.True(t, hasDiscard, "A card should have been discarded")

	// 7. Submit card selection for wave 1
	gameLoop.SubmitAction(player1, "select_card", map[string]interface{}{
		"card": cards.Card{Suit: cards.Hearts, Value: cards.Ace},
	})

	gameLoop.SubmitAction(player2, "select_card", map[string]interface{}{
		"card": cards.Card{Suit: cards.Clubs, Value: cards.King},
	})

	// Wait for wave 2
	waitForState(t, gameLoop, GameStateWave2Reveal)

	// 8. Submit card selection for wave 2
	gameLoop.SubmitAction(player1, "select_card", map[string]interface{}{
		"card": cards.Card{Suit: cards.Diamonds, Value: cards.Queen},
	})

	gameLoop.SubmitAction(player2, "select_card", map[string]interface{}{
		"card": cards.Card{Suit: cards.Spades, Value: cards.Jack},
	})

	// Wait for wave 3
	waitForState(t, gameLoop, GameStateWave3Reveal)

	// 9. Submit card selection for wave 3
	gameLoop.SubmitAction(player1, "select_card", map[string]interface{}{
		"card": cards.Card{Suit: cards.Hearts, Value: cards.Nine},
	})

	gameLoop.SubmitAction(player2, "select_card", map[string]interface{}{
		"card": cards.Card{Suit: cards.Clubs, Value: cards.Eight},
	})

	// 10. Wait for hand evaluation and showdown
	waitForState(t, gameLoop, GameStateHandEvaluation)
	waitForState(t, gameLoop, GameStateShowdown)

	// Check for card selection events
	cardSelections := 0
	for _, e := range eventStore.GetEvents() {
		if _, ok := e.(events.CommunityCardSelected); ok {
			cardSelections++
		}
	}
	assert.Equal(t, 6, cardSelections, "Should have 6 card selections (3 per player)")

	// 11. Wait for hand completion
	waitForState(t, gameLoop, GameStateHandComplete)

	// 12. Wait for the next hand to start (should go back to ante collection)
	waitForState(t, gameLoop, GameStateAnteCollection)

	// Verify we have 2 HandStarted events now
	handStartedCount := 0
	for _, e := range eventStore.GetEvents() {
		if _, ok := e.(events.HandStarted); ok {
			handStartedCount++
		}
	}
	assert.Equal(t, 2, handStartedCount, "Two hands should have been started")
}

// Helper function to wait for a specific game state
func waitForState(t *testing.T, gameLoop *GameLoop, expectedState GameState) {
	t.Helper()

	// Maximum time to wait for a state transition
	maxWaitTime := 5 * time.Second
	checkInterval := 50 * time.Millisecond

	// Calculate number of iterations based on max wait time and check interval
	maxIterations := int(maxWaitTime / checkInterval)

	for i := 0; i < maxIterations; i++ {
		// Check current state
		gameLoop.stateUpdateLock.Lock()
		currentState := gameLoop.currentState
		gameLoop.stateUpdateLock.Unlock()

		if currentState == expectedState {
			return // State reached successfully
		}

		// Wait before checking again
		time.Sleep(checkInterval)
	}

	// If we get here, we timed out waiting for the state
	gameLoop.stateUpdateLock.Lock()
	currentState := gameLoop.currentState
	gameLoop.stateUpdateLock.Unlock()

	t.Fatalf("Timed out waiting for state %s, current state is %s", expectedState, currentState)
}
