package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPlayerSeats(t *testing.T) {
	// Setup
	table := &Table{
		ID:     uuid.NewString(),
		Name:   "Test Table",
		Status: TableStatusWaiting,
	}
	player := Player{
		ID:   uuid.NewString(),
		Name: "Test Player",
	}

	// Test successful addition
	err := table.PlayerSeats(player)
	assert.NoError(t, err)
	assert.Len(t, table.Players, 1)
	assert.Equal(t, player.ID, table.Players[0].ID)

	// Test error when player already exists
	err = table.PlayerSeats(player)
	assert.Error(t, err)
	assert.Equal(t, "player already at table", err.Error())

	// Test error when table not waiting
	table.Status = TableStatusPlaying
	newPlayer := Player{
		ID:   uuid.NewString(),
		Name: "Another Player",
	}
	err = table.PlayerSeats(newPlayer)
	assert.Error(t, err)
	assert.Equal(t, "can only add players when table is waiting", err.Error())
}

func TestPlayerBuysIn(t *testing.T) {
	// Setup
	playerID := uuid.NewString()
	table := &Table{
		ID:     uuid.NewString(),
		Name:   "Test Table",
		Status: TableStatusWaiting,
		Players: []Player{
			{
				ID:      playerID,
				Name:    "Test Player",
				Balance: 1000,
			},
		},
		BuyIns: make(map[string]int),
	}

	// Test successful buy-in
	err := table.PlayerBuysIn(playerID, 500)
	assert.NoError(t, err)
	assert.Equal(t, 500, table.Players[0].Balance)
	assert.Equal(t, 500, table.BuyIns[playerID])

	// Test error when table not waiting
	table.Status = TableStatusPlaying
	err = table.PlayerBuysIn(playerID, 100)
	assert.Error(t, err)
	assert.Equal(t, "can only add chips when table is waiting", err.Error())

	// Reset status for further tests
	table.Status = TableStatusWaiting

	// Test error when player not found
	err = table.PlayerBuysIn("non-existent", 100)
	assert.Error(t, err)
	assert.Equal(t, "player not found", err.Error())

	// Test error when insufficient balance
	err = table.PlayerBuysIn(playerID, 1001)
	assert.Error(t, err)
	assert.Equal(t, "player does not have enough balance", err.Error())
}

func TestPlayerLeaves(t *testing.T) {
	// Setup
	playerID := uuid.NewString()
	player := Player{
		ID:   playerID,
		Name: "Test Player",
	}
	table := &Table{
		ID:      uuid.NewString(),
		Name:    "Test Table",
		Players: []Player{player},
		BuyIns:  map[string]int{playerID: 500}, // Initialize BuyIns for the player
	}

	// Test successful leave
	err := table.PlayerLeaves(playerID)
	assert.NoError(t, err)
	assert.Empty(t, table.Players)
	assert.Empty(t, table.BuyIns) // Verify buy-ins are cleared

	// Test error when player not found
	err = table.PlayerLeaves(playerID)
	assert.Error(t, err)
	assert.Equal(t, "player not found", err.Error())
}

func TestAllowPlaying(t *testing.T) {
	// Setup
	table := &Table{
		ID:     uuid.NewString(),
		Name:   "Test Table",
		Status: TableStatusWaiting,
	}

	// Test error with not enough players
	err := table.AllowPlaying()
	assert.Error(t, err)
	assert.Equal(t, "need at least 2 players to start", err.Error())

	// Add players
	table.Players = []Player{
		{ID: uuid.NewString(), Name: "Player 1"},
		{ID: uuid.NewString(), Name: "Player 2"},
	}

	// Test successful start
	err = table.AllowPlaying()
	assert.NoError(t, err)
	assert.Equal(t, TableStatusPlaying, table.Status)

	// Test error when table not in waiting status
	table.Status = TableStatusWaiting // Reset to waiting
	table.AllowPlaying()              // Start the game
	table.Status = TableStatusPlaying
	err = table.AllowPlaying()
	assert.Error(t, err)
	assert.Equal(t, "table must be in waiting status to start playing", err.Error())
}

func TestStartNewHand(t *testing.T) {
	// Setup
	table := &Table{
		ID:     uuid.NewString(),
		Name:   "Test Table",
		Status: TableStatusWaiting,
		Rules:  TableRules{},
		Players: []Player{
			{ID: uuid.NewString(), Name: "Player 1"},
			{ID: uuid.NewString(), Name: "Player 2"},
		},
		BuyIns: map[string]int{ // Initialize BuyIns
			"player1": 500,
			"player2": 500,
		},
	}

	// Test error when table not in playing status
	err := table.StartNewHand()
	assert.Error(t, err)
	assert.Equal(t, "table must be in playing status to start a new hand", err.Error())

	// Set table to playing
	table.Status = TableStatusPlaying

	// Test successful start hand
	err = table.StartNewHand()
	assert.NoError(t, err)
	assert.NotNil(t, table.ActiveHand)
	assert.Len(t, table.Hands, 1)
	assert.Equal(t, 0, table.ActiveHand.ButtonPosition)
}

func TestFindButtonPosition(t *testing.T) {
	// Setup
	table := &Table{
		ID:     uuid.NewString(),
		Name:   "Test Table",
		Status: TableStatusPlaying,
		Players: []Player{
			{ID: uuid.NewString(), Name: "Player 1"},
			{ID: uuid.NewString(), Name: "Player 2"},
			{ID: uuid.NewString(), Name: "Player 3"},
		},
	}

	// Test first hand (no active hand)
	position := table.findButtonPosition()
	assert.Equal(t, 0, position)

	// Test subsequent hands
	table.ActiveHand = &Hand{ButtonPosition: 0}
	position = table.findButtonPosition()
	assert.Equal(t, 1, position)

	// Test button position wrapping
	table.ActiveHand.ButtonPosition = 2
	position = table.findButtonPosition()
	assert.Equal(t, 0, position)
}
