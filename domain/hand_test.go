package domain

import (
	"fmt"
	"testing"
	"time"

	"github.com/lazharichir/poker/domain/events"
	"github.com/stretchr/testify/assert"
)

func getDefaultTableRules() TableRules {
	return TableRules{
		AnteValue:                 10,
		ContinuationBetMultiplier: 3,
		PlayerTimeout:             3 * time.Second,
	}
}

func NewTestTable() *Table {
	table := NewTable("Poker Table For Testing", getDefaultTableRules())
	table.ID = "tbl_test"
	return table
}

// setupTestHand creates a hand with specified number of players in the continuation phase
func setupContinuationPhaseHand(numPlayers int) (*Hand, *Table) {
	table := NewTestTable()

	// Create players
	players := make([]Player, numPlayers)
	for i := 0; i < numPlayers; i++ {
		playerID := "player-" + fmt.Sprint(1+i)
		players[i] = Player{
			ID:   playerID,
			Name: "Player " + fmt.Sprint(1+i),
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
		event, found := findEventOfType(hand.Events, events.ContinuationBetPlaced{}.Name())
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

		// Current bettor is the last player to act
		currentBettorID := hand.CurrentBettor
		assert.Equal(t, "player-2", currentBettorID)

		err := hand.PlayerPlacesContinuationBet(hand.CurrentBettor, 100)
		assert.NoError(t, err)

		err = hand.PlayerPlacesContinuationBet(hand.CurrentBettor, 100)
		assert.NoError(t, err)

		// Check phase transition occurred
		assert.Equal(t, HandPhase_CommunityDeal, hand.Phase)

		// Check BettingRoundEnded event was emitted
		event, found := findEventOfType(hand.Events, events.BettingRoundEnded{}.Name())
		assert.True(t, found)
		endEvent, ok := event.(events.BettingRoundEnded)
		assert.True(t, ok)
		assert.Equal(t, string(HandPhase_Continuation), endEvent.Phase)

		// Check PhaseChanged event was emitted
		event, found = findEventOfType(hand.Events, events.PhaseChanged{}.Name())
		assert.True(t, found)
		phaseEvent, ok := event.(events.PhaseChanged)
		assert.True(t, ok)
		assert.Equal(t, string(HandPhase_Continuation), phaseEvent.PreviousPhase)
		assert.Equal(t, string(HandPhase_CommunityDeal), phaseEvent.NewPhase)
	})
}

// setupAntesPhaseHand creates a hand with specified number of players in the antes phase
func setupAntesPhaseHand(numPlayers int) (*Hand, *Table) {
	table := NewTestTable()

	// Create players
	players := make([]Player, numPlayers)
	for i := 0; i < numPlayers; i++ {
		playerID := "player-" + fmt.Sprint(1+i)
		players[i] = Player{
			ID:   playerID,
			Name: "Player " + fmt.Sprint(1+i),
		}
		table.BuyIns[playerID] = 1000 // Start with 1000 chips
	}

	// Create hand
	hand := &Hand{
		ID:             "test-hand-id",
		TableID:        "test-table-id",
		Table:          table,
		Phase:          HandPhase_Antes,
		Players:        players,
		ActivePlayers:  make(map[string]bool),
		AntesPaid:      make(map[string]int),
		ButtonPosition: 0, // First player is the button
		TableRules: TableRules{
			AnteValue:     10,
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

func TestPlayerPlacesAnte(t *testing.T) {
	t.Run("Successful ante placement", func(t *testing.T) {
		// Setup
		hand, table := setupAntesPhaseHand(3)
		currentBettorID := hand.CurrentBettor
		initialChips := table.GetPlayerBuyIn(currentBettorID)
		anteAmount := hand.TableRules.AnteValue
		initialPot := hand.Pot
		initialEventsCount := len(hand.Events)

		// Act
		err := hand.PlayerPlacesAnte(currentBettorID, anteAmount)

		// Assert
		assert.NoError(t, err)

		// Check player's chips decreased
		assert.Equal(t, initialChips-anteAmount, table.GetPlayerBuyIn(currentBettorID))

		// Check pot increased
		assert.Equal(t, initialPot+anteAmount, hand.Pot)

		// Check ante was recorded
		assert.Equal(t, anteAmount, hand.AntesPaid[currentBettorID])

		// Check AntePlaced event was emitted
		assert.Greater(t, len(hand.Events), initialEventsCount)
		event, found := findEventOfType(hand.Events, events.AntePlaced{}.Name())
		assert.True(t, found)
		anteEvent, ok := event.(events.AntePlaced)
		assert.True(t, ok)
		assert.Equal(t, currentBettorID, anteEvent.PlayerID)
		assert.Equal(t, anteAmount, anteEvent.Amount)

		// Check current bettor was updated to next player
		assert.NotEqual(t, currentBettorID, hand.CurrentBettor)
	})

	t.Run("Error when not in antes phase", func(t *testing.T) {
		// Setup
		hand, _ := setupAntesPhaseHand(3)
		hand.Phase = HandPhase_Hole // Change to wrong phase
		currentBettorID := hand.CurrentBettor

		// Act
		err := hand.PlayerPlacesAnte(currentBettorID, hand.TableRules.AnteValue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in antes phase")
	})

	t.Run("Error when not player's turn", func(t *testing.T) {
		// Setup
		hand, _ := setupAntesPhaseHand(3)
		wrongPlayerID := hand.Players[2].ID // Pick a player who is not the current bettor

		// Act
		err := hand.PlayerPlacesAnte(wrongPlayerID, hand.TableRules.AnteValue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not this player's turn to act")
	})

	t.Run("Error when player already paid ante", func(t *testing.T) {
		// Setup
		hand, _ := setupAntesPhaseHand(3)
		currentBettorID := hand.CurrentBettor

		// Make player already have placed an ante
		hand.AntesPaid[currentBettorID] = hand.TableRules.AnteValue

		// Act
		err := hand.PlayerPlacesAnte(currentBettorID, hand.TableRules.AnteValue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "player already paid ante")
	})

	t.Run("Transition to hole phase when all antes are paid", func(t *testing.T) {
		// Setup
		hand, _ := setupAntesPhaseHand(2) // Just 2 players for simplicity

		// Place ante for the first player
		err := hand.PlayerPlacesAnte(hand.CurrentBettor, hand.TableRules.AnteValue)
		assert.NoError(t, err)

		// Get the second player's ID which should now be the current bettor
		secondPlayerID := hand.CurrentBettor

		// Act - place ante for the second player, which should be the last one
		err = hand.PlayerPlacesAnte(secondPlayerID, hand.TableRules.AnteValue)
		assert.NoError(t, err)

		// Assert phase transition occurred
		assert.Equal(t, HandPhase_Hole, hand.Phase)

		// Check BettingRoundEnded event was emitted
		event, found := findEventOfType(hand.Events, events.BettingRoundEnded{}.Name())
		assert.True(t, found)
		endEvent, ok := event.(events.BettingRoundEnded)
		assert.True(t, ok)
		assert.Equal(t, string(HandPhase_Antes), endEvent.Phase)

		// Check PhaseChanged event was emitted
		event, found = findEventOfType(hand.Events, events.PhaseChanged{}.Name())
		assert.True(t, found)
		phaseEvent, ok := event.(events.PhaseChanged)
		assert.True(t, ok)
		assert.Equal(t, string(HandPhase_Antes), phaseEvent.PreviousPhase)
		assert.Equal(t, string(HandPhase_Hole), phaseEvent.NewPhase)
	})
}

func TestHandleAntePhaseTimeout(t *testing.T) {
	t.Run("Some players have paid antes", func(t *testing.T) {
		// Setup
		hand, _ := setupAntesPhaseHand(3)

		// First player has paid ante
		firstPlayerID := hand.CurrentBettor
		hand.PlayerPlacesAnte(firstPlayerID, hand.TableRules.AnteValue)

		// Second and third players have not paid
		secondPlayerID := hand.Players[2].ID
		thirdPlayerID := hand.Players[0].ID // Remember: button is at 0, so this is the button player

		assert.True(t, hand.IsPlayerActive(secondPlayerID))
		assert.True(t, hand.IsPlayerActive(thirdPlayerID))

		// Initial counts
		initialActiveCount := hand.countActivePlayers()

		// Act
		err := hand.HandleAntePhaseTimeout()

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, HandPhase_Hole, hand.Phase) // Should transition to hole phase

		// Second and third players should be folded
		assert.False(t, hand.IsPlayerActive(secondPlayerID))
		assert.False(t, hand.IsPlayerActive(thirdPlayerID))

		// Active players should have decreased
		assert.Equal(t, initialActiveCount-2, hand.countActivePlayers())

		// Check for PlayerTimedOut events
		timedOutEvents := 0
		for _, event := range hand.Events {
			if timeoutEvent, ok := event.(events.PlayerTimedOut); ok {
				timedOutEvents++
				assert.Equal(t, "fold", timeoutEvent.DefaultAction)
				assert.Contains(t, []string{secondPlayerID, thirdPlayerID}, timeoutEvent.PlayerID)
			}
		}
		assert.Equal(t, 2, timedOutEvents)

		// Check BettingRoundEnded event was emitted
		event, found := findEventOfType(hand.Events, events.BettingRoundEnded{}.Name())
		assert.True(t, found)
		assert.Equal(t, string(HandPhase_Antes), event.(events.BettingRoundEnded).Phase)
	})

	t.Run("No players have paid antes, should end hand", func(t *testing.T) {
		// Setup
		hand, _ := setupAntesPhaseHand(3)

		// Act
		err := hand.HandleAntePhaseTimeout()

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, HandPhase_Ended, hand.Phase) // Should end the hand

		// All players should be folded
		for _, player := range hand.Players {
			assert.False(t, hand.IsPlayerActive(player.ID))
		}

		// Check for PlayerTimedOut events
		timedOutEvents := 0
		for _, event := range hand.Events {
			if _, ok := event.(events.PlayerTimedOut); ok {
				timedOutEvents++
			}
		}
		assert.Equal(t, 3, timedOutEvents)

		// Check for PhaseChanged events (should be two: Antes->Hole and Hole->Ended)
		phaseChanges := 0
		for _, event := range hand.Events {
			if _, ok := event.(events.PhaseChanged); ok {
				phaseChanges++
			}
		}
		assert.Equal(t, 1, phaseChanges) // Direct transition from Antes to Ended
	})

	t.Run("Error when not in ante phase", func(t *testing.T) {
		// Setup
		hand, _ := setupAntesPhaseHand(3)
		hand.Phase = HandPhase_Hole

		// Act
		err := hand.HandleAntePhaseTimeout()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in ante phase")
	})
}

func TestTransitionToHolePhase(t *testing.T) {
	t.Run("Successful transition from antes to hole", func(t *testing.T) {
		// Setup
		hand, _ := setupAntesPhaseHand(3)
		initialEventsCount := len(hand.Events)

		// Act
		hand.TransitionToHolePhase()

		// Assert
		assert.Equal(t, HandPhase_Hole, hand.Phase)

		// Check PhaseChanged event was emitted
		assert.Greater(t, len(hand.Events), initialEventsCount)
		event, found := findEventOfType(hand.Events, events.PhaseChanged{}.Name())
		assert.True(t, found)
		phaseEvent, ok := event.(events.PhaseChanged)
		assert.True(t, ok)
		assert.Equal(t, string(HandPhase_Antes), phaseEvent.PreviousPhase)
		assert.Equal(t, string(HandPhase_Hole), phaseEvent.NewPhase)

		// Check that CurrentBettor is reset to player left of button
		expectedBettor := hand.getPlayerLeftOfButton()
		assert.Equal(t, expectedBettor, hand.CurrentBettor)
	})

	t.Run("No transition when not in antes phase", func(t *testing.T) {
		// Setup
		hand, _ := setupAntesPhaseHand(3)
		hand.Phase = HandPhase_Start
		initialEventsCount := len(hand.Events)

		// Act
		hand.TransitionToHolePhase()

		// Assert
		assert.Equal(t, HandPhase_Start, hand.Phase)          // Phase shouldn't change
		assert.Equal(t, initialEventsCount, len(hand.Events)) // No new events
	})
}
