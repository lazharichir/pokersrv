package domain

import (
	"errors"
	"fmt"

	"github.com/lazharichir/poker/domain/events"
)

// Lobby represents the poker game lobby
type Lobby struct {
	tables map[string]*Table

	// Events
	Events        []events.Event
	eventHandlers []events.EventHandler
}

// NewTable creates a new table with the given name and rules
func (l *Lobby) NewTable(name string, rules TableRules) (*Table, error) {
	if l.tables == nil {
		l.tables = make(map[string]*Table)
	}

	// Create a new table
	table := NewTable(name, rules)
	if table == nil {
		return nil, errors.New("failed to create table")
	}

	table.RegisterEventHandler(l.handleTableEvent)

	// Add to tables map
	l.tables[table.ID] = table

	return table, nil
}

func (l *Lobby) handleTableEvent(event events.Event) {
	fmt.Println("---")
	fmt.Println("Game received event from table:", event.Name())

	l.emitEvent(event)

	switch ev := event.(type) {
	default:
		_ = ev
	}
}

// GetTable retrieves a table by ID
func (l *Lobby) GetTable(tableID string) (*Table, error) {
	if l.tables == nil {
		l.tables = make(map[string]*Table)
	}

	table, exists := l.tables[tableID]
	if !exists {
		return nil, errors.New("table not found")
	}

	return table, nil
}

// AddEventHandler adds an event handler to the lobby
func (l *Lobby) AddEventHandler(handler events.EventHandler) {
	l.eventHandlers = append(l.eventHandlers, handler)
}

// emitEvent notifies all registered handlers of a new event
func (l *Lobby) emitEvent(event events.Event) {
	// Add event to game's event log
	l.Events = append(l.Events, event)

	// Notify all handlers
	for _, handler := range l.eventHandlers {
		handler(event)
	}
}
