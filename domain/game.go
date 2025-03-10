package domain

import (
	"errors"
	"fmt"

	"github.com/lazharichir/poker/domain/events"
)

// Message represents a game event message
type Message interface {
	Name() string
}

// Game represents the poker game server
type Game struct {
	tables map[string]*Table

	// Events
	Events        []events.Event
	eventHandlers []events.EventHandler
}

// NewTable creates a new table with the given name and rules
func (g *Game) NewTable(name string, rules TableRules) (*Table, error) {
	if g.tables == nil {
		g.tables = make(map[string]*Table)
	}

	// Create a new table
	table := NewTable(name, rules)
	if table == nil {
		return nil, errors.New("failed to create table")
	}

	table.RegisterEventHandler(g.handleTableEvent)

	// Add to tables map
	g.tables[table.ID] = table

	return table, nil
}

func (g *Game) handleTableEvent(event events.Event) {
	fmt.Println("---")
	fmt.Println("Game received event from table:", event.Name())

	g.emitEvent(event)

	switch ev := event.(type) {
	default:
		_ = ev
	}
}

// GetTable retrieves a table by ID
func (g *Game) GetTable(tableID string) (*Table, error) {
	if g.tables == nil {
		g.tables = make(map[string]*Table)
	}

	table, exists := g.tables[tableID]
	if !exists {
		return nil, errors.New("table not found")
	}

	return table, nil
}

// AddEventHandler adds an event handler to the game
func (g *Game) AddEventHandler(handler events.EventHandler) {
	g.eventHandlers = append(g.eventHandlers, handler)
}

// emitEvent notifies all registered handlers of a new event
func (g *Game) emitEvent(event events.Event) {
	// Add event to game's event log
	g.Events = append(g.Events, event)

	// Notify all handlers
	for _, handler := range g.eventHandlers {
		handler(event)
	}
}
