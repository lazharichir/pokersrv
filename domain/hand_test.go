package domain

import (
	"fmt"
	"testing"
	"time"

	"github.com/lazharichir/poker/cards"
	"github.com/lazharichir/poker/domain/events"
	"github.com/lazharichir/poker/domain/hands"
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
		eventHandlers:       []events.EventHandler{},
		Events:              []events.Event{},
		Deck:                cards.NewDeck52(),
		CommunitySelections: make(map[string]cards.Stack),
		CommunityCards:      cards.Stack{},
		HoleCards:           make(map[string]cards.Stack),
		AntesPaid:           make(map[string]int),
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
		eventHandlers:       []events.EventHandler{},
		Events:              []events.Event{},
		Deck:                cards.NewDeck52(),
		HoleCards:           make(map[string]cards.Stack),
		CommunityCards:      cards.Stack{},
		Pot:                 0,
		Results:             []hands.HandComparisonResult{},
		ContinuationBets:    make(map[string]int),
		CommunitySelections: make(map[string]cards.Stack),
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

func TestDealHoleCards(t *testing.T) {
	t.Run("Successfully deal hole cards", func(t *testing.T) {
		// Setup
		hand, _ := setupAntesPhaseHand(3)
		hand.Phase = HandPhase_Hole

		// Act
		err := hand.DealHoleCards()
		assert.NoError(t, err)

		// Each active player should have 2 cards
		for playerID, active := range hand.ActivePlayers {
			if active {
				assert.Len(t, hand.HoleCards[playerID], 2)
			}
		}

		// Check events are emitted
		_, found := findEventOfType(hand.Events, events.HoleCardsDealt{}.Name())
		assert.True(t, found)

		// Should transition to continuation phase
		assert.Equal(t, HandPhase_Continuation, hand.Phase)
	})

	t.Run("Error when not in hole card phase", func(t *testing.T) {
		hand, _ := setupAntesPhaseHand(3)
		err := hand.DealHoleCards() // Don't change to hole phase
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in hole card phase")
	})
}

func TestDealCommunityCard(t *testing.T) {
	t.Run("Successfully deal community card", func(t *testing.T) {
		// Setup
		hand, _ := setupContinuationPhaseHand(3)
		hand.Phase = HandPhase_CommunityDeal
		initialCommunityCardCount := len(hand.CommunityCards)

		// Act
		err := hand.DealCommunityCard()

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, initialCommunityCardCount+1, len(hand.CommunityCards))

		// Check event is emitted
		_, found := findEventOfType(hand.Events, events.CommunityCardDealt{}.Name())
		assert.True(t, found)
	})

	t.Run("Transition to community selection after 8 cards", func(t *testing.T) {
		// Setup
		hand, _ := setupContinuationPhaseHand(3)
		hand.Phase = HandPhase_CommunityDeal

		// Add 7 cards
		for i := 0; i < 7; i++ {
			hand.CommunityCards = append(hand.CommunityCards, cards.Card{})
		}

		// Act - deal the 8th card
		err := hand.DealCommunityCard()

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, HandPhase_CommunitySelection, hand.Phase)
	})
}

func TestPlayerFolds(t *testing.T) {
	t.Run("Successfully fold", func(t *testing.T) {
		// Setup
		hand, _ := setupContinuationPhaseHand(3)
		currentBettorID := hand.CurrentBettor

		// Act
		err := hand.PlayerFolds(currentBettorID)

		// Assert
		assert.NoError(t, err)
		assert.False(t, hand.IsPlayerActive(currentBettorID))

		// Check event is emitted
		event, found := findEventOfType(hand.Events, events.PlayerFolded{}.Name())
		assert.True(t, found)
		foldedEvent, ok := event.(events.PlayerFolded)
		assert.True(t, ok)
		assert.Equal(t, currentBettorID, foldedEvent.PlayerID)
	})

	t.Run("Last player standing wins immediately", func(t *testing.T) {
		// Setup
		hand, _ := setupContinuationPhaseHand(2)
		currentBettorID := hand.CurrentBettor

		// Act
		err := hand.PlayerFolds(currentBettorID)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, HandPhase_Ended, hand.Phase)

		// Check SingleWinnerDetermined event emitted
		_, found := findEventOfType(hand.Events, events.SingleWinnerDetermined{}.Name())
		assert.True(t, found)
	})
}

func TestPlayerSelectsCommunityCard(t *testing.T) {
	t.Run("Successfully select card", func(t *testing.T) {
		// Setup
		hand, _ := setupContinuationPhaseHand(2)
		hand.Phase = HandPhase_CommunitySelection
		hand.CommunitySelectionStartedAt = time.Now()
		playerID := hand.Players[0].ID

		// Add community card
		testCard := cards.Card{Suit: cards.Hearts, Value: cards.Ace}
		hand.CommunityCards = append(hand.CommunityCards, testCard)

		// Act
		err := hand.PlayerSelectsCommunityCard(playerID, testCard)

		// Assert
		assert.NoError(t, err)
		assert.Contains(t, hand.CommunitySelections[playerID], testCard)

		// Check event emitted
		_, found := findEventOfType(hand.Events, events.CommunityCardSelected{}.Name())
		assert.True(t, found)
	})

	t.Run("Error when selecting more than 3 cards", func(t *testing.T) {
		// Setup
		hand, _ := setupContinuationPhaseHand(2)
		hand.Phase = HandPhase_CommunitySelection
		hand.CommunitySelectionStartedAt = time.Now()
		playerID := hand.Players[0].ID

		// Add community cards
		hand.CommunityCards = cards.NewDeck52()[:8]

		// Player already selected 3 cards
		hand.CommunitySelections[playerID] = hand.CommunityCards[:3]

		// Act
		err := hand.PlayerSelectsCommunityCard(playerID, hand.CommunityCards[3])

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already selected 3 cards")
	})
}

func TestEvaluateHands(t *testing.T) {
	t.Run("Correctly determine winner with three-of-a-kind vs two pair", func(t *testing.T) {
		// Setup: create a hand with 2 players
		hand, _ := setupContinuationPhaseHand(2)
		hand.Phase = HandPhase_Decision

		player1ID := hand.Players[0].ID
		player2ID := hand.Players[1].ID

		// Set up hole cards
		// Player 1: Ace of Hearts, Ace of Spades (pair of aces)
		hand.HoleCards[player1ID] = cards.Stack{
			{Suit: cards.Hearts, Value: cards.Ace},
			{Suit: cards.Spades, Value: cards.Ace},
		}

		// Player 2: King of Spades, Queen of Spades
		hand.HoleCards[player2ID] = cards.Stack{
			{Suit: cards.Spades, Value: cards.King},
			{Suit: cards.Spades, Value: cards.Queen},
		}

		// Set up community cards (available to both players)
		hand.CommunityCards = cards.Stack{
			{Suit: cards.Clubs, Value: cards.Ace},      // Third ace for player 1
			{Suit: cards.Clubs, Value: cards.King},     // King for player 2's pair
			{Suit: cards.Diamonds, Value: cards.Queen}, // Queen for player 2's pair
			{Suit: cards.Hearts, Value: cards.King},    // Extra King (not used)
			{Suit: cards.Hearts, Value: cards.Queen},   // Extra Queen (not used)
			{Suit: cards.Hearts, Value: cards.Ten},     // Filler card
			{Suit: cards.Diamonds, Value: cards.Five},  // Filler card
			{Suit: cards.Clubs, Value: cards.Two},      // Filler card
		}

		// Player 1 selects: Ace of Clubs, King of Hearts, Queen of Hearts
		// Will make three-of-a-kind with Aces
		hand.CommunitySelections[player1ID] = cards.Stack{
			{Suit: cards.Clubs, Value: cards.Ace},
			{Suit: cards.Hearts, Value: cards.King},
			{Suit: cards.Hearts, Value: cards.Queen},
		}

		// Player 2 selects: King of Clubs, Queen of Diamonds, Ten of Hearts
		// Will make two pairs: Kings and Queens
		hand.CommunitySelections[player2ID] = cards.Stack{
			{Suit: cards.Clubs, Value: cards.King},
			{Suit: cards.Diamonds, Value: cards.Queen},
			{Suit: cards.Hearts, Value: cards.Ten},
		}

		// Mark both players as active
		hand.ActivePlayers[player1ID] = true
		hand.ActivePlayers[player2ID] = true

		// Act
		results, err := hand.EvaluateHands()

		// Assert
		assert.NoError(t, err)
		assert.Len(t, results, 2, "Should return results for both players")

		// Find player 1's result
		var player1Result, player2Result hands.HandComparisonResult
		for _, result := range results {
			if result.PlayerID == player1ID {
				player1Result = result
			} else if result.PlayerID == player2ID {
				player2Result = result
			}
		}

		// Player 1 should be the winner with three of a kind
		assert.True(t, player1Result.IsWinner, "Player 1 with three-of-a-kind should be the winner")
		assert.False(t, player2Result.IsWinner, "Player 2 with two pair should not be the winner")

		// Verify hand ranks
		assert.Equal(t, hands.ThreeOfAKind, player1Result.HandRank, "Player 1 should have three of a kind")
		assert.Equal(t, hands.TwoPair, player2Result.HandRank, "Player 2 should have two pair")

		// Verify player 1 has higher rank
		assert.Greater(t, player1Result.HandRank, player2Result.HandRank, "Three of a kind should rank higher than two pair")
	})
}

func TestPayout(t *testing.T) {
	t.Run("Single winner payout", func(t *testing.T) {
		// Setup
		hand, table := setupContinuationPhaseHand(3)
		hand.Phase = HandPhase_Payout
		hand.Pot = 300

		// Set up a single winner
		winnerID := hand.Players[0].ID
		initialChips := table.GetPlayerBuyIn(winnerID)
		hand.Results = []hands.HandComparisonResult{
			{PlayerID: winnerID, IsWinner: true, HandRank: 1},
		}

		// Act
		err := hand.Payout()

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, initialChips+300, table.GetPlayerBuyIn(winnerID))
		assert.Equal(t, 0, hand.Pot)
		assert.Equal(t, HandPhase_Ended, hand.Phase)
	})

	t.Run("Split pot between multiple winners", func(t *testing.T) {
		t.Skip("Not implemented yet")
		// Setup for split pot scenario
	})
}

func TestBurnCard(t *testing.T) {
	t.Run("Successfully burn card", func(t *testing.T) {
		hand, _ := setupContinuationPhaseHand(2)
		initialDeckSize := len(hand.Deck)

		err := hand.BurnCard()

		assert.NoError(t, err)
		assert.Equal(t, initialDeckSize-1, len(hand.Deck))
	})
}

func TestCountActivePlayers(t *testing.T) {
	t.Skip("Not implemented yet")
	// Test with different active player counts
}

func TestHandleView(t *testing.T) {
	t.Run("BuildPlayerView returns correct view", func(t *testing.T) {
		t.Skip("Not implemented yet")
		// Test player view construction
	})

	t.Run("getAvailableActions returns correct actions", func(t *testing.T) {
		t.Skip("Not implemented yet")
		// Test available actions in different phases
	})
}
