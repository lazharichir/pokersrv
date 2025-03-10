package domain

import (
	"testing"
	"time"

	"github.com/lazharichir/poker/domain/events"
	"github.com/stretchr/testify/assert"
)

// MockEvent implements events.Event interface for testing purposes
type MockEvent struct {
	name      string
	timestamp time.Time
}

func (e *MockEvent) Name() string {
	return e.name
}

func (e *MockEvent) Timestamp() time.Time {
	return e.timestamp
}

func TestNewTable(t *testing.T) {
	// Setup
	game := &Game{}
	tableName := "Test Table"
	rules := TableRules{}

	// Test successful table creation
	table, err := game.NewTable(tableName, rules)
	assert.NoError(t, err)
	assert.NotNil(t, table)
	assert.Equal(t, tableName, table.Name)
	assert.Equal(t, 1, len(game.tables))

	// Verify table is stored in game's tables map
	retrievedTable, exists := game.tables[table.ID]
	assert.True(t, exists)
	assert.Equal(t, table, retrievedTable)
}

func TestGetTable(t *testing.T) {
	// Setup
	game := &Game{
		tables: make(map[string]*Table),
	}
	tableName := "Test Table"
	rules := TableRules{}

	// Create a table first
	table, _ := game.NewTable(tableName, rules)

	// Test successful retrieval
	retrievedTable, err := game.GetTable(table.ID)
	assert.NoError(t, err)
	assert.Equal(t, table, retrievedTable)

	// Test error when table not found
	_, err = game.GetTable("non-existent-id")
	assert.Error(t, err)
	assert.Equal(t, "table not found", err.Error())
}

func TestAddEventHandler(t *testing.T) {
	// Setup
	game := &Game{}
	handlerCalled := false

	// Create a test handler
	handler := func(event events.Event) {
		handlerCalled = true
	}

	// Add the handler
	game.AddEventHandler(handler)

	// Verify handler was added
	assert.Equal(t, 1, len(game.eventHandlers))

	// Create a mock event
	mockEvent := &MockEvent{
		name:      "test_event",
		timestamp: time.Now(),
	}

	// Emit the event
	game.emitEvent(mockEvent)

	// Verify handler was called
	assert.True(t, handlerCalled)

	// Verify event was logged
	assert.Equal(t, 1, len(game.Events))
	assert.Equal(t, mockEvent, game.Events[0])
}

func TestHandleTableEvent(t *testing.T) {
	// Setup
	game := &Game{}
	eventReceived := false

	// Add a game event handler to verify event propagation
	game.AddEventHandler(func(event events.Event) {
		if event.Name() == "table_event" {
			eventReceived = true
		}
	})

	// Create a mock event
	mockEvent := &MockEvent{
		name:      "table_event",
		timestamp: time.Now(),
	}

	// Call handleTableEvent
	game.handleTableEvent(mockEvent)

	// Verify event was propagated to game handlers
	assert.True(t, eventReceived)

	// Verify event was logged
	assert.Equal(t, 1, len(game.Events))
	assert.Equal(t, mockEvent, game.Events[0])
}

func TestGame_MultipleEventHandlers(t *testing.T) {
	// Setup
	game := &Game{}
	handler1Called := false
	handler2Called := false

	// Create test handlers
	handler1 := func(event events.Event) {
		handler1Called = true
	}
	handler2 := func(event events.Event) {
		handler2Called = true
	}

	// Add the handlers
	game.AddEventHandler(handler1)
	game.AddEventHandler(handler2)

	// Create a mock event
	mockEvent := &MockEvent{
		name:      "test_event",
		timestamp: time.Now(),
	}

	// Emit the event
	game.emitEvent(mockEvent)

	// Verify both handlers were called
	assert.True(t, handler1Called)
	assert.True(t, handler2Called)
}
