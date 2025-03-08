package game

import (
	"testing"

	"github.com/lazharichir/poker/events"
)

// MockEventStore implements the EventStore interface for testing
type MockEventStore struct {
	events map[string][]events.Event
}

func NewMockEventStore() *MockEventStore {
	return &MockEventStore{
		events: make(map[string][]events.Event),
	}
}

func (m *MockEventStore) Append(event events.Event) error {
	tableID := events.GetTableID(event)
	if _, exists := m.events[tableID]; !exists {
		m.events[tableID] = []events.Event{}
	}
	m.events[tableID] = append(m.events[tableID], event)
	return nil
}

func (m *MockEventStore) LoadEvents(tableID string) ([]events.Event, error) {
	if events, exists := m.events[tableID]; exists {
		return events, nil
	}
	return []events.Event{}, nil
}

// TestHandFlowSuccess tests a successful flow of a hand from start to completion
func TestHandFlowSuccess(t *testing.T) {
	// Setup
	mockEventStore := NewMockEventStore()
	tableID := "test-table-123"

	// Create a table in the event store
	tableCreatedEvent := events.TableCreated{
		TableID:                   tableID,
		Name:                      "Test Table",
		Ante:                      10,
		ContinuationBetMultiplier: 2,
		DiscardPhaseDuration:      10,
		DiscardCostType:           "fixed",
		DiscardCostValue:          5,
	}
	mockEventStore.Append(tableCreatedEvent)

	// Add players to the table
	player1ID := "player-1"
	player2ID := "player-2"
	player3ID := "player-3"

	mockEventStore.Append(events.PlayerJoinedTable{
		TableID:      tableID,
		PlayerID:     player1ID,
		ChipsBrought: 1000,
	})

	mockEventStore.Append(events.PlayerJoinedTable{
		TableID:      tableID,
		PlayerID:     player2ID,
		ChipsBrought: 1000,
	})

	mockEventStore.Append(events.PlayerJoinedTable{
		TableID:      tableID,
		PlayerID:     player3ID,
		ChipsBrought: 1000,
	})

	// Create table engine
	engine, err := NewTableEngine(mockEventStore, tableID)
	if err != nil {
		t.Fatalf("Failed to create table engine: %v", err)
	}

	// Test phase at start
	if engine.phase != PhaseNotStarted {
		t.Errorf("Expected initial phase to be PhaseNotStarted, got %v", engine.phase)
	}

	// Start a hand
	err = engine.StartHand()
	if err != nil {
		t.Fatalf("Failed to start hand: %v", err)
	}

	// Verify phase changed to ante collection
	if engine.phase != PhaseAnteCollection {
		t.Errorf("Expected phase to be PhaseAnteCollection after starting hand, got %v", engine.phase)
	}

	// Place antes for all players
	for _, playerID := range engine.activePlayers {
		err = engine.PlaceAnte(playerID)
		if err != nil {
			t.Fatalf("Failed to place ante for player %s: %v", playerID, err)
		}
	}

	// Verify phase changed to continuation bet after dealing hole cards
	if engine.phase != PhaseContinuationBet {
		t.Errorf("Expected phase to be PhaseContinuationBet after antes placed, got %v", engine.phase)
	}

	// Check that players were dealt 2 hole cards each
	for playerID, player := range engine.tableState.Players {
		if len(player.HoleCards) != 2 {
			t.Errorf("Expected player %s to have 2 hole cards, got %d", playerID, len(player.HoleCards))
		}
	}

	// First player places continuation bet, others fold
	err = engine.PlaceContinuationBet(engine.activePlayers[0])
	if err != nil {
		t.Fatalf("Failed to place continuation bet: %v", err)
	}

	// Second player folds
	err = engine.Fold(engine.activePlayers[1])
	if err != nil {
		t.Fatalf("Failed to fold: %v", err)
	}

	// If there are 3 players, check if we can place a bet for the third
	if len(engine.activePlayers) > 1 {
		err = engine.PlaceContinuationBet(engine.activePlayers[1])
		if err != nil {
			t.Fatalf("Failed to place continuation bet for third player: %v", err)
		}
	}

	// Verify community cards were dealt
	if len(engine.tableState.CommunityCards) != 8 {
		t.Errorf("Expected 8 community cards to be dealt, got %d", len(engine.tableState.CommunityCards))
	}

	// Check phase changed to discard
	if engine.phase != PhaseDiscard {
		t.Errorf("Expected phase to be PhaseDiscard, got %v", engine.phase)
	}

	// Players discard or skip discard
	for i := 0; i < len(engine.activePlayers); i++ {
		playerID := engine.activePlayers[engine.currentPlayerTurnIdx]
		if i%2 == 0 { // Have some players discard, others skip
			err = engine.DiscardCard(playerID, 0) // Discard first card
		} else {
			err = engine.SkipDiscard(playerID)
		}
		if err != nil {
			t.Fatalf("Failed discard/skip for player %s: %v", playerID, err)
		}
	}

	// Wait for card selection phase timing
	// In a real test, we'd mock time or use dependency injection
	// For now we'll just verify the engine moved to card selection phase
	if engine.phase != PhaseCardSelection {
		t.Fatalf("Expected phase to be PhaseCardSelection, got %v", engine.phase)
	}

	// Simulate card selection for active players
	// In a real test we'd need to handle the timing aspects better
	for _, playerID := range engine.activePlayers {
		// Select 3 different cards
		for i := 0; i < 3 && i < len(engine.tableState.CommunityCards); i++ {
			err = engine.SelectCommunityCard(playerID, i)
			if err != nil {
				t.Fatalf("Failed to select community card for player %s: %v", playerID, err)
			}

			// Verify card was added to player's selected cards
			player := engine.tableState.Players[playerID]
			if len(player.SelectedCommunityCards) != i+1 {
				t.Errorf("Expected player %s to have %d selected cards, got %d",
					playerID, i+1, len(player.SelectedCommunityCards))
			}
		}
	}

	// Call evaluateHands directly for testing
	// In a real game this would be triggered by the timer
	engine.evaluateHands()

	// Verify phase changed to hand completed
	if engine.phase != PhaseHandCompleted {
		t.Errorf("Expected phase to be PhaseHandCompleted, got %v", engine.phase)
	}

	// Verify a winner was determined
	if len(engine.activePlayers) > 0 {
		// First place should be one of the active players
		firstPlaceFound := false
		for _, playerID := range engine.activePlayers {
			if playerID == engine.tableState.LastHandFirstPlacePlayerID {
				firstPlaceFound = true
				break
			}
		}
		if !firstPlaceFound && engine.tableState.LastHandFirstPlacePlayerID != "" {
			t.Errorf("First place winner %s is not among active players",
				engine.tableState.LastHandFirstPlacePlayerID)
		}
	}
}

// TestHandFlowEdgeCases tests edge cases in the hand flow
func TestHandFlowEdgeCases(t *testing.T) {
	// Setup
	mockEventStore := NewMockEventStore()
	tableID := "test-table-456"

	// Create a table in the event store
	tableCreatedEvent := events.TableCreated{
		TableID:                   tableID,
		Name:                      "Edge Case Table",
		Ante:                      10,
		ContinuationBetMultiplier: 2,
		DiscardPhaseDuration:      10,
		DiscardCostType:           "fixed",
		DiscardCostValue:          5,
	}
	mockEventStore.Append(tableCreatedEvent)

	// Add minimum players to the table (just 2)
	player1ID := "player-1"
	player2ID := "player-2"

	mockEventStore.Append(events.PlayerJoinedTable{
		TableID:      tableID,
		PlayerID:     player1ID,
		ChipsBrought: 1000,
	})

	mockEventStore.Append(events.PlayerJoinedTable{
		TableID:      tableID,
		PlayerID:     player2ID,
		ChipsBrought: 1000,
	})

	// Create table engine
	engine, err := NewTableEngine(mockEventStore, tableID)
	if err != nil {
		t.Fatalf("Failed to create table engine: %v", err)
	}

	// Start a hand
	err = engine.StartHand()
	if err != nil {
		t.Fatalf("Failed to start hand: %v", err)
	}

	// Test that we can't start another hand while one is in progress
	err = engine.StartHand()
	if err == nil {
		t.Errorf("Expected error when starting a hand while one is in progress")
	}

	// Place antes for all players
	for _, playerID := range engine.activePlayers {
		err = engine.PlaceAnte(playerID)
		if err != nil {
			t.Fatalf("Failed to place ante for player %s: %v", playerID, err)
		}
	}

	// Test that we can't place an ante in the wrong phase
	err = engine.PlaceAnte(player1ID)
	if err == nil {
		t.Errorf("Expected error when placing ante in wrong phase")
	}

	// Second player folds, leaving only one player
	err = engine.Fold(engine.activePlayers[1])
	if err != nil {
		t.Fatalf("Failed to fold: %v", err)
	}

	// Verify hand completed since only one player remains
	if engine.phase != PhaseHandCompleted {
		t.Errorf("Expected phase to be PhaseHandCompleted after all but one player folded, got %v", engine.phase)
	}

	// Verify the remaining player is the winner
	if engine.tableState.LastHandFirstPlacePlayerID != player1ID {
		t.Errorf("Expected player %s to be the winner, got %s",
			player1ID, engine.tableState.LastHandFirstPlacePlayerID)
	}
}
