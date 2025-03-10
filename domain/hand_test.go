package domain

import (
	"fmt"
	"testing"
	"time"

	"github.com/lazharichir/poker/domain/events"
	"github.com/stretchr/testify/assert"
)

func NewMockTable() *Table {
	return &Table{
		ID:   "tbl_test",
		Name: "Test Table",
		Rules: TableRules{
			AnteValue:                 10,
			ContinuationBetMultiplier: 3,
			PlayerTimeout:             3 * time.Second,
		},
		BuyIns:  make(map[string]int),
		Players: []Player{},
	}
}

// setupTestHand creates a hand with specified number of players in the continuation phase
func setupContinuationPhaseHand(numPlayers int) (*Hand, *Table) {
	table := NewMockTable()

	// Create players
	players := make([]Player, numPlayers)
	for i := 0; i < numPlayers; i++ {
		playerID := "player-" + fmt.Sprint('1'+i)
		players[i] = Player{
			ID:   playerID,
			Name: "Player " + fmt.Sprint('1'+i),
		}
		table.BuyIns[playerID] = 1000 // Start with 1000 chips
	}

	// Create hand
	hand := &Hand{
		ID:               "test-hand-id",
		TableID:          "test-table-id",
		Table:            table,
		Phase:            HandPhase_Continuation,
		Players:          players,
		ActivePlayers:    make(map[string]bool),
		ContinuationBets: make(map[string]int),
		ButtonPosition:   0, // First player is the button
		TableRules: TableRules{
			PlayerTimeout: 30 * time.Second,
		},
		eventHandlers: []events.EventHandler{},
		Events:        []events.Event{},
	}

	// Set all players as active
	for _, player := range players {
		hand.ActivePlayers[player.ID] = true
	}

	// Set player to the left of the button as current bettor
	hand.CurrentBettor = players[1].ID

	return hand, table
}

// findEventOfType searches for an event of the specified type in the events slice
func findEventOfType(events []events.Event, eventType string) (events.Event, bool) {
	for _, event := range events {
		if event.Name() == eventType {
			return event, true
		}
	}
	return nil, false
}

func TestPlayerPlacesContinuationBet(t *testing.T) {
	t.Run("Successful continuation bet", func(t *testing.T) {
		// Setup
		hand, table := setupContinuationPhaseHand(3)
		currentBettorID := hand.CurrentBettor
		initialChips := table.GetPlayerBuyIn(currentBettorID)
		betAmount := 100
		initialPot := hand.Pot
		initialEventsCount := len(hand.Events)

		// Act
		err := hand.PlayerPlacesContinuationBet(currentBettorID, betAmount)

		// Assert
		assert.NoError(t, err)

		// Check player's chips decreased
		assert.Equal(t, initialChips-betAmount, table.GetPlayerBuyIn(currentBettorID))

		// Check pot increased
		assert.Equal(t, initialPot+betAmount, hand.Pot)

		// Check continuation bet recorded
		assert.Equal(t, betAmount, hand.ContinuationBets[currentBettorID])

		// Check ContinuationBetPlaced event was emitted
		assert.Greater(t, len(hand.Events), initialEventsCount)
		event, found := findEventOfType(hand.Events, "ContinuationBetPlaced")
		assert.True(t, found)
		betEvent, ok := event.(events.ContinuationBetPlaced)
		assert.True(t, ok)
		assert.Equal(t, currentBettorID, betEvent.PlayerID)
		assert.Equal(t, betAmount, betEvent.Amount)

		// Check current bettor was updated to next player
		assert.NotEqual(t, currentBettorID, hand.CurrentBettor)
	})

	t.Run("Error when not in continuation phase", func(t *testing.T) {
		// Setup
		hand, _ := setupContinuationPhaseHand(3)
		hand.Phase = HandPhase_Hole // Change to wrong phase
		currentBettorID := hand.CurrentBettor

		// Act
		err := hand.PlayerPlacesContinuationBet(currentBettorID, 100)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in continuation bet phase")
	})

	t.Run("Error when not player's turn", func(t *testing.T) {
		// Setup
		hand, _ := setupContinuationPhaseHand(3)
		wrongPlayerID := hand.Players[2].ID // Pick a player who is not the current bettor

		// Act
		err := hand.PlayerPlacesContinuationBet(wrongPlayerID, 100)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not this player's turn to act")
	})

	t.Run("Error when player already placed bet", func(t *testing.T) {
		// Setup
		hand, _ := setupContinuationPhaseHand(3)
		currentBettorID := hand.CurrentBettor

		// Make player already have placed a bet
		hand.ContinuationBets[currentBettorID] = 50

		// Act
		err := hand.PlayerPlacesContinuationBet(currentBettorID, 100)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "player already made continuation bet decision")
	})

	t.Run("Transition to community deal when all players have bet", func(t *testing.T) {
		// Setup
		hand, _ := setupContinuationPhaseHand(2) // Just 2 players for simplicity

		// First player already bet
		firstPlayerID := hand.Players[1].ID
		hand.ContinuationBets[firstPlayerID] = 50

		// Current bettor is the last player to act
		currentBettorID := hand.CurrentBettor
		assert.Equal(t, "player-2", currentBettorID)

		// Act - this should complete the betting round
		err := hand.PlayerPlacesContinuationBet(currentBettorID, 100)

		// Assert
		assert.NoError(t, err)

		// Check phase transition occurred
		assert.Equal(t, HandPhase_CommunityDeal, hand.Phase)

		// Check BettingRoundEnded event was emitted
		event, found := findEventOfType(hand.Events, "BettingRoundEnded")
		assert.True(t, found)
		endEvent, ok := event.(events.BettingRoundEnded)
		assert.True(t, ok)
		assert.Equal(t, string(HandPhase_Continuation), endEvent.Phase)

		// Check PhaseChanged event was emitted
		event, found = findEventOfType(hand.Events, "PhaseChanged")
		assert.True(t, found)
		phaseEvent, ok := event.(events.PhaseChanged)
		assert.True(t, ok)
		assert.Equal(t, string(HandPhase_Continuation), phaseEvent.PreviousPhase)
		assert.Equal(t, string(HandPhase_CommunityDeal), phaseEvent.NewPhase)
	})
}
