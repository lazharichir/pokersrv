package poker

import (
	"errors"
	"time"
)

// Message represents a game event message
type Message interface {
	MessageName() string
}

// Game represents the poker game server
type Game struct {
	tables       map[string]*Table
	messageQueue []Message
	listeners    []func(Message)
}

// Player represents a player in the game
type Player struct {
	ID      string
	Name    string
	Balance int
	Status  string
	Chips   int // chips brought to the table
}

// ActionName represents a type of action a player can take
type ActionName string

// Event represents something that happened during a hand
type Event struct {
	Type      string
	PlayerID  string
	Timestamp time.Time
	Data      interface{}
}

// AddTable adds a new table to the game
func (g *Game) AddTable(table Table) error {
	if g.tables == nil {
		g.tables = make(map[string]*Table)
	}

	if _, exists := g.tables[table.ID]; exists {
		return errors.New("table with this ID already exists")
	}

	table.Status = TableStatusWaiting
	g.tables[table.ID] = &table

	return nil
}

// GetTable retrieves a table by ID
func (g *Game) GetTable(tableID string) (*Table, error) {
	if g.tables == nil {
		return nil, errors.New("no tables exist")
	}

	table, exists := g.tables[tableID]
	if !exists {
		return nil, errors.New("table not found")
	}

	return table, nil
}

// listen registers a message listener
func (g *Game) listen(listener func(Message)) {
	g.listeners = append(g.listeners, listener)
}

// publish sends a message to all listeners
func (g *Game) publish(msg Message) {
	for _, listener := range g.listeners {
		listener(msg)
	}
}
