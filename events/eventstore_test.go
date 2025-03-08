package events

import (
	"testing"

	"github.com/lazharichir/poker/cards"
)

func TestInMemoryEventStore(t *testing.T) {
	store := NewInMemoryEventStore()

	// Test data
	tableID := "table-123"
	playerID := "player-456"

	// Test appending an event
	t.Run("Append and load events", func(t *testing.T) {
		// Create test events
		handStarted := HandStarted{
			TableID:        tableID,
			ButtonPlayerID: playerID,
			AnteAmount:     10,
			PlayerIDs:      []string{playerID, "player-789"},
		}

		antePlaced := AntePlacedByPlayer{
			TableID:  tableID,
			PlayerID: playerID,
			Amount:   10,
		}

		holeCard := PlayerHoleCardDealt{
			TableID:  tableID,
			PlayerID: playerID,
			Card:     cards.Card{Suit: cards.Spades, Value: cards.Ace},
		}

		// Append events to the store
		if err := store.Append(handStarted); err != nil {
			t.Errorf("Failed to append HandStarted event: %v", err)
		}
		if err := store.Append(antePlaced); err != nil {
			t.Errorf("Failed to append AntePlaced event: %v", err)
		}
		if err := store.Append(holeCard); err != nil {
			t.Errorf("Failed to append PlayerHoleCardDealt event: %v", err)
		}

		// Load events back
		events, err := store.LoadEvents(tableID)
		if err != nil {
			t.Errorf("Failed to load events: %v", err)
		}

		// Check events count
		if len(events) != 3 {
			t.Errorf("Expected 3 events, got %d", len(events))
		}

		// Check event types and ordering
		if events[0].EventName() != "hand-started" {
			t.Errorf("Expected first event to be hand-started, got %s", events[0].EventName())
		}
		if events[1].EventName() != "ante-placed" {
			t.Errorf("Expected second event to be ante-placed, got %s", events[1].EventName())
		}
		if events[2].EventName() != "player-hole-card-dealt" {
			t.Errorf("Expected third event to be player-hole-card-dealt, got %s", events[2].EventName())
		}
	})

	t.Run("Load events for non-existent table", func(t *testing.T) {
		events, err := store.LoadEvents("non-existent-table")
		if err != nil {
			t.Errorf("Expected no error for non-existent table, got %v", err)
		}
		if len(events) != 0 {
			t.Errorf("Expected 0 events for non-existent table, got %d", len(events))
		}
	})
}
