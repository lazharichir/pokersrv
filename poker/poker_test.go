package poker

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBasicTableFlow(t *testing.T) {

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

	game := Game{}

	// Listen to messages
	game.listen(func(msg Message) {
		fmt.Println("[MESSAGE_RECEIVED]")
		fmt.Println(msg.MessageName(), msg)
	})

	table := Table{
		ID:   "tbl1",
		Name: "Test Table",
		Rules: TableRules{
			AnteValue:                 10,
			ContinuationBetMultiplier: 2,
			DiscardPhaseDuration:      10,
			DiscardCostType:           "fixed",
			DiscardCostValue:          5,
			PlayerTimeout:             3 * time.Second,
		},
		Hands:      []Hand{},
		ActiveHand: nil,
		Players:    []Player{},
	}

	errAddTable := game.AddTable(table)
	assert.NoError(t, errAddTable)

	table1, errGetTable := game.GetTable("tbl1")
	assert.NoError(t, errGetTable)

	// add player 1 to table with 500 chips brought to that table
	addPlayer1Err := table1.PlayerSeats(p1)
	table1.PlayerBuysIn(p1.ID, 500)
	assert.NoError(t, addPlayer1Err)

	// add player 2 to table with 1000 chips brought to that table
	addPlayer2Err := table1.PlayerSeats(p2)
	table1.PlayerBuysIn(p2.ID, 100)
	table1.PlayerBuysIn(p2.ID, 400)
	assert.NoError(t, addPlayer2Err)

	// start the table
	startErr := table1.AllowPlaying()
	assert.NoError(t, startErr)

	// start a new hand
	startHandErr := table1.StartNewHand()
	assert.NoError(t, startHandErr)

	// meaning, there now must be a hand in the table
	assert.NotNil(t, table1.ActiveHand)

	// the hand must have the same table ID as the table
	assert.Equal(t, table1.ID, table1.ActiveHand.TableID)

	// Test table's StartNewHand created the hand correctly
	hand := table1.ActiveHand
	assert.Equal(t, HandPhase_Start, hand.Phase)

	// Test Hand phase transitions (these methods exist in Hand)
	hand.TransitionToAntesPhase()
	assert.Equal(t, HandPhase_Antes, hand.Phase)
	assert.True(t, hand.IsInPhase(HandPhase_Antes))

	hand.TransitionToHolePhase()
	assert.Equal(t, HandPhase_Hole, hand.Phase)

	hand.TransitionToContinuationPhase()
	assert.Equal(t, HandPhase_Continuation, hand.Phase)

	hand.TransitionToCommunityDealPhase()
	assert.Equal(t, HandPhase_CommunityDeal, hand.Phase)

	// Test skipping phases (should not change phase)
	hand.TransitionToPayoutPhase() // Should not work as we need proper phase sequence
	assert.Equal(t, HandPhase_CommunityDeal, hand.Phase)

	// Continue proper phase sequence
	hand.TransitionToCommunityRevealPhase()
	assert.Equal(t, HandPhase_CommunityReveal, hand.Phase)

	hand.TransitionToHandRevealPhase()
	assert.Equal(t, HandPhase_HandReveal, hand.Phase)

	hand.TransitionToDecisionPhase()
	assert.Equal(t, HandPhase_Decision, hand.Phase)

	hand.TransitionToPayoutPhase()
	assert.Equal(t, HandPhase_Payout, hand.Phase)

	hand.TransitionToEndedPhase()
	assert.Equal(t, HandPhase_Ended, hand.Phase)
	assert.True(t, hand.HasEnded())

	// Test player leaving the table
	leaveErr := table1.PlayerLeaves(p2.ID)
	assert.NoError(t, leaveErr)
	assert.Equal(t, 1, len(table1.Players))

	// Try to start a new hand with insufficient players
	secondHandErr := table1.StartNewHand()
	assert.Error(t, secondHandErr)
}

func TestPlayerManagement(t *testing.T) {
	// Create a simple table
	table := Table{
		ID:   "test-table",
		Name: "Player Management Test",
		Rules: TableRules{
			AnteValue: 10,
		},
		Status: TableStatusWaiting,
	}

	// Add players
	player1 := Player{
		ID:      "p1",
		Name:    "Player 1",
		Balance: 1000,
	}

	player2 := Player{
		ID:      "p2",
		Name:    "Player 2",
		Balance: 500,
	}

	// Test PlayerSeats
	err1 := table.PlayerSeats(player1)
	assert.NoError(t, err1)
	assert.Equal(t, 1, len(table.Players))

	// Test PlayerBuysIn
	buyInErr := table.PlayerBuysIn(player1.ID, 300)
	assert.NoError(t, buyInErr)

	// Verify balance changed
	assert.Equal(t, 300, table.Players[0].Chips)
	assert.Equal(t, 700, table.Players[0].Balance) // 1000 - 300

	// Test adding second player
	err2 := table.PlayerSeats(player2)
	assert.NoError(t, err2)
	assert.Equal(t, 2, len(table.Players))

	// Test StartPlaying
	startErr := table.AllowPlaying()
	assert.NoError(t, startErr)
	assert.Equal(t, TableStatusPlaying, table.Status)

	// Test player leaving
	leaveErr := table.PlayerLeaves(player1.ID)
	assert.NoError(t, leaveErr)
	assert.Equal(t, 1, len(table.Players))
}
