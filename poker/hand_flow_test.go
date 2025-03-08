package poker

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lazharichir/poker/cards"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a test game with table and players
func setupTestGameWithPlayers(t *testing.T) (*Game, *Table, []Player) {
	game := Game{}

	// Create players
	p1 := Player{
		ID:      "p1",
		Name:    "Player 1",
		Balance: 1000,
		Status:  "active",
	}

	p2 := Player{
		ID:      "p2",
		Name:    "Player 2",
		Balance: 1500,
		Status:  "active",
	}

	p3 := Player{
		ID:      "p3",
		Name:    "Player 3",
		Balance: 2000,
		Status:  "active",
	}

	// Create table
	tableID := "test-table-" + uuid.NewString()[:8]
	table := Table{
		ID:   tableID,
		Name: "Test Hand Flow Table",
		Rules: TableRules{
			AnteValue:                 10,
			ContinuationBetMultiplier: 2,
			DiscardPhaseDuration:      10,
			DiscardCostType:           "fixed",
			DiscardCostValue:          5,
			PlayerTimeout:             3 * time.Second,
		},
		Status: TableStatusWaiting,
	}

	// Add table to game
	err := game.AddTable(table)
	assert.NoError(t, err)

	// Get the table
	tbl, err := game.GetTable(tableID)
	assert.NoError(t, err)

	// Add players to table
	err = tbl.PlayerSeats(p1)
	assert.NoError(t, err)
	err = tbl.PlayerBuysIn(p1.ID, 500)
	assert.NoError(t, err)

	err = tbl.PlayerSeats(p2)
	assert.NoError(t, err)
	err = tbl.PlayerBuysIn(p2.ID, 500)
	assert.NoError(t, err)

	err = tbl.PlayerSeats(p3)
	assert.NoError(t, err)
	err = tbl.PlayerBuysIn(p3.ID, 500)
	assert.NoError(t, err)

	// Start the table
	err = tbl.StartPlaying()
	assert.NoError(t, err)

	return &game, tbl, []Player{p1, p2, p3}
}

// Helper function to initialize a hand for testing
func initializeTestHand(t *testing.T, table *Table) *Hand {
	// Start a new hand
	err := table.StartNewHand()
	assert.NoError(t, err)
	assert.NotNil(t, table.ActiveHand)

	hand := table.ActiveHand
	assert.Equal(t, HandPhase_Start, hand.Phase)

	// Initialize the hand
	hand.InitializeHand()
	assert.NotEmpty(t, hand.Deck)

	// Make sure all players are active
	for _, player := range table.Players {
		assert.True(t, hand.ActivePlayers[player.ID])
	}

	return hand
}

// TestBasicHandFlow tests a full hand from start to finish with all players staying in
func TestBasicHandFlow(t *testing.T) {
	_, table, players := setupTestGameWithPlayers(t)
	hand := initializeTestHand(t, table)

	// Start the hand with antes phase
	hand.TransitionToAntesPhase()
	assert.Equal(t, HandPhase_Antes, hand.Phase)

	// Players post antes
	for _, player := range players {
		// Set current bettor to this player to simulate turn order
		hand.CurrentBettor = player.ID
		err := hand.PlayerPlacesAnte(player.ID, table.Rules.AnteValue)
		assert.NoError(t, err)
	}

	// After all antes are posted, it should move to hole card phase
	assert.Equal(t, HandPhase_Hole, hand.Phase)

	// Deal hole cards
	err := hand.DealHoleCards()
	assert.NoError(t, err)

	// Check each player has 2 hole cards
	for _, player := range players {
		assert.Equal(t, 2, len(hand.HoleCards[player.ID]))
	}

	// Move to continuation bet phase
	hand.TransitionToContinuationPhase()
	assert.Equal(t, HandPhase_Continuation, hand.Phase)

	// Players place continuation bets
	for _, player := range players {
		// Set current bettor to this player to simulate turn order
		hand.CurrentBettor = player.ID
		err := hand.PlayerPlacesContinuationBet(player.ID, table.Rules.AnteValue*table.Rules.ContinuationBetMultiplier)
		assert.NoError(t, err)
	}

	// After all continuation bets, should move to community deal phase
	assert.Equal(t, HandPhase_CommunityDeal, hand.Phase)

	// Deal community cards
	err = hand.BurnCard()
	assert.NoError(t, err)

	for i := 0; i < 5; i++ {
		err = hand.DealCommunityCard()
		assert.NoError(t, err)
	}

	assert.Equal(t, 5, len(hand.CommunityCards))

	fmt.Print(hand.PrintState())

	// Move through remaining phases
	hand.TransitionToCommunityRevealPhase()
	assert.Equal(t, HandPhase_CommunityReveal, hand.Phase)

	hand.TransitionToHandRevealPhase()
	assert.Equal(t, HandPhase_HandReveal, hand.Phase)

	// Add stub for hand comparison since we're just testing the flow
	// hand.comparePlayerHands = func(playerCards map[string]interface{}) ([]HandComparisonResult, error) {
	// 	results := []HandComparisonResult{
	// 		{PlayerID: players[0].ID, HandRank: 1, IsWinner: true, PlaceIndex: 0},
	// 		{PlayerID: players[1].ID, HandRank: 2, IsWinner: false, PlaceIndex: 1},
	// 		{PlayerID: players[2].ID, HandRank: 3, IsWinner: false, PlaceIndex: 2},
	// 	}
	// 	return results, nil
	// }

	hand.TransitionToDecisionPhase()
	assert.Equal(t, HandPhase_Decision, hand.Phase)

	hand.TransitionToPayoutPhase()
	assert.Equal(t, HandPhase_Payout, hand.Phase)

	// Check pot before payout
	initialPot := hand.Pot
	assert.Greater(t, initialPot, 0)

	// Calculate expected pot from antes and continuation bets
	expectedPot := len(players)*table.Rules.AnteValue +
		len(players)*(table.Rules.AnteValue*table.Rules.ContinuationBetMultiplier)
	assert.Equal(t, expectedPot, initialPot)

	// Execute payout
	err = hand.Payout()
	assert.NoError(t, err)

	// After payout, pot should be empty
	assert.Equal(t, 0, hand.Pot)

	// Hand should be marked as ended
	assert.Equal(t, HandPhase_Ended, hand.Phase)
	assert.True(t, hand.HasEnded())

	// Winner should have more chips than they started with
	winnerFound := false
	for _, player := range table.Players {
		if player.ID == players[0].ID {
			assert.Greater(t, player.Chips, 500) // Started with 500 chips
			winnerFound = true
		}
	}
	assert.True(t, winnerFound, "Winner should be found among table players")
}

// TestPlayerFoldingFlow tests a hand where one player folds during the continuation phase
func TestPlayerFoldingFlow(t *testing.T) {
	_, table, players := setupTestGameWithPlayers(t)
	hand := initializeTestHand(t, table)

	// Start the hand with antes phase
	hand.TransitionToAntesPhase()

	// Players post antes
	for _, player := range players {
		hand.CurrentBettor = player.ID
		err := hand.PlayerPlacesAnte(player.ID, table.Rules.AnteValue)
		assert.NoError(t, err)
	}

	// Deal hole cards
	err := hand.DealHoleCards()
	assert.NoError(t, err)

	// Move to continuation bet phase
	hand.TransitionToContinuationPhase()

	// First player places continuation bet
	hand.CurrentBettor = players[0].ID
	err = hand.PlayerPlacesContinuationBet(players[0].ID, table.Rules.AnteValue*table.Rules.ContinuationBetMultiplier)
	assert.NoError(t, err)

	// Second player folds
	hand.CurrentBettor = players[1].ID
	err = hand.PlayerFolds(players[1].ID)
	assert.NoError(t, err)

	// Verify player is no longer active
	assert.False(t, hand.IsPlayerActive(players[1].ID))

	// Third player places continuation bet
	hand.CurrentBettor = players[2].ID
	err = hand.PlayerPlacesContinuationBet(players[2].ID, table.Rules.AnteValue*table.Rules.ContinuationBetMultiplier)
	assert.NoError(t, err)

	// After all continuation bets/folds, should move to community deal phase
	assert.Equal(t, HandPhase_CommunityDeal, hand.Phase)

	// Count active players
	activeCount := 0
	for _, active := range hand.ActivePlayers {
		if active {
			activeCount++
		}
	}
	assert.Equal(t, 2, activeCount)

	// Continue with the hand as normal
	// Deal community cards
	err = hand.BurnCard()
	assert.NoError(t, err)

	for i := 0; i < 5; i++ {
		err = hand.DealCommunityCard()
		assert.NoError(t, err)
	}

	// Move through remaining phases
	hand.TransitionToCommunityRevealPhase()
	hand.TransitionToHandRevealPhase()

	// Add stub for hand comparison
	// hand.comparePlayerHands = func(playerCards map[string]interface{}) ([]HandComparisonResult, error) {
	// 	// Only active players should be compared
	// 	results := []HandComparisonResult{
	// 		{PlayerID: players[0].ID, HandRank: 1, IsWinner: true, PlaceIndex: 0},
	// 		{PlayerID: players[2].ID, HandRank: 2, IsWinner: false, PlaceIndex: 1},
	// 	}
	// 	return results, nil
	// }

	hand.TransitionToDecisionPhase()
	hand.TransitionToPayoutPhase()

	// Execute payout
	err = hand.Payout()
	assert.NoError(t, err)

	// Hand should be ended
	assert.True(t, hand.HasEnded())
}

// TestDiscardFlow tests a hand including the discard phase
func TestDiscardFlow(t *testing.T) {
	_, table, players := setupTestGameWithPlayers(t)

	// Update table rules to include discard phase
	table.Rules.DiscardPhaseDuration = 30
	table.Rules.DiscardCostValue = 15

	hand := initializeTestHand(t, table)

	// Start the hand and go through phases up to community deal
	hand.TransitionToAntesPhase()

	// Players post antes
	for _, player := range players {
		hand.CurrentBettor = player.ID
		err := hand.PlayerPlacesAnte(player.ID, table.Rules.AnteValue)
		assert.NoError(t, err)
	}

	// Deal hole cards
	err := hand.DealHoleCards()
	assert.NoError(t, err)

	// Store initial hole cards to verify they change after discard
	initialHoleCards := make(map[string][]cards.Card)
	for _, player := range players {
		initialHoleCards[player.ID] = make([]cards.Card, len(hand.HoleCards[player.ID]))
		copy(initialHoleCards[player.ID], hand.HoleCards[player.ID])
	}

	hand.TransitionToContinuationPhase()

	// Players place continuation bets
	for _, player := range players {
		hand.CurrentBettor = player.ID
		err := hand.PlayerPlacesContinuationBet(player.ID, table.Rules.AnteValue*table.Rules.ContinuationBetMultiplier)
		assert.NoError(t, err)
	}

	// After continuation bets, go to community deal
	assert.Equal(t, HandPhase_CommunityDeal, hand.Phase)

	// Deal community cards
	err = hand.BurnCard()
	assert.NoError(t, err)

	for i := 0; i < 5; i++ {
		err = hand.DealCommunityCard()
		assert.NoError(t, err)
	}

	// Now transition to discard phase instead of community reveal
	hand.TransitionToDiscardPhase()
	assert.Equal(t, HandPhase_Discard, hand.Phase)

	// First player pays discard cost and discards first card
	hand.CurrentBettor = players[0].ID
	err = hand.PlayerPaysDiscardCost(players[0].ID)
	assert.NoError(t, err)

	err = hand.PlayerDiscardsCard(players[0].ID, 0)
	assert.NoError(t, err)

	// Verify card changed
	assert.NotEqual(t, initialHoleCards[players[0].ID][0], hand.HoleCards[players[0].ID][0])
	assert.Equal(t, 2, len(hand.HoleCards[players[0].ID])) // Still has 2 cards

	// Second player doesn't discard (skip)

	// Third player pays discard cost and discards second card
	hand.CurrentBettor = players[2].ID
	err = hand.PlayerPaysDiscardCost(players[2].ID)
	assert.NoError(t, err)

	err = hand.PlayerDiscardsCard(players[2].ID, 1)
	assert.NoError(t, err)

	// Verify card changed
	assert.Equal(t, initialHoleCards[players[2].ID][0], hand.HoleCards[players[2].ID][0])    // First card unchanged
	assert.NotEqual(t, initialHoleCards[players[2].ID][1], hand.HoleCards[players[2].ID][1]) // Second card changed
	assert.Equal(t, 2, len(hand.HoleCards[players[2].ID]))                                   // Still has 2 cards

	// Continue with the rest of the hand
	hand.TransitionToCommunityRevealPhase()
	hand.TransitionToHandRevealPhase()

	// Add stub for hand comparison
	// hand.comparePlayerHands = func(playerCards map[string]interface{}) ([]HandComparisonResult, error) {
	// 	results := []HandComparisonResult{
	// 		{PlayerID: players[0].ID, HandRank: 2, IsWinner: false, PlaceIndex: 1},
	// 		{PlayerID: players[1].ID, HandRank: 3, IsWinner: false, PlaceIndex: 2},
	// 		{PlayerID: players[2].ID, HandRank: 1, IsWinner: true, PlaceIndex: 0}, // Player who discarded second card wins
	// 	}
	// 	return results, nil
	// }

	hand.TransitionToDecisionPhase()
	hand.TransitionToPayoutPhase()

	// Check pot includes discard costs
	potBeforePayout := hand.Pot
	assert.Equal(t,
		(len(players)*table.Rules.AnteValue)+
			(len(players)*(table.Rules.AnteValue*table.Rules.ContinuationBetMultiplier))+
			(2*table.Rules.DiscardCostValue), // Two players paid discard costs
		potBeforePayout)

	// Execute payout
	err = hand.Payout()
	assert.NoError(t, err)

	// After payout, winner should have gained chips
	for _, player := range table.Players {
		if player.ID == players[2].ID { // Winner
			assert.Greater(t, player.Chips, 500) // Started with 500
		}
	}
}

// TestSinglePlayerRemainingFlow tests a hand where all but one player folds
func TestSinglePlayerRemainingFlow(t *testing.T) {
	_, table, players := setupTestGameWithPlayers(t)
	hand := initializeTestHand(t, table)

	// Start the hand with antes phase
	hand.TransitionToAntesPhase()

	// Players post antes
	for _, player := range players {
		hand.CurrentBettor = player.ID
		err := hand.PlayerPlacesAnte(player.ID, table.Rules.AnteValue)
		assert.NoError(t, err)
	}

	// Deal hole cards
	err := hand.DealHoleCards()
	assert.NoError(t, err)

	// Move to continuation bet phase
	hand.TransitionToContinuationPhase()

	// First player places continuation bet
	hand.CurrentBettor = players[0].ID
	err = hand.PlayerPlacesContinuationBet(players[0].ID, table.Rules.AnteValue*table.Rules.ContinuationBetMultiplier)
	assert.NoError(t, err)

	// Record pot size before folds
	potBeforeFolds := hand.Pot
	assert.Greater(t, potBeforeFolds, 0)

	// Record player 0's chips before winning by default
	var player0ChipsBeforeWin int
	for _, p := range table.Players {
		if p.ID == players[0].ID {
			player0ChipsBeforeWin = p.Chips
		}
	}

	// Second player folds
	hand.CurrentBettor = players[1].ID
	err = hand.PlayerFolds(players[1].ID)
	assert.NoError(t, err)

	// Third player folds - should trigger automatic win for player 0
	hand.CurrentBettor = players[2].ID
	err = hand.PlayerFolds(players[2].ID)
	assert.NoError(t, err)

	// Should automatically jump to payout phase when only one player remains
	assert.Equal(t, HandPhase_Payout, hand.Phase)

	// Check if player 0 won the pot
	var player0ChipsAfterWin int
	for _, p := range table.Players {
		if p.ID == players[0].ID {
			player0ChipsAfterWin = p.Chips
		}
	}

	// Player 0 should have won the pot amount
	assert.Equal(t, player0ChipsBeforeWin+potBeforeFolds, player0ChipsAfterWin)

	// Pot should be empty
	assert.Equal(t, 0, hand.Pot)

	// Hand should be ended
	assert.Equal(t, HandPhase_Ended, hand.Phase)
	assert.True(t, hand.HasEnded())
}
