package domain

import (
	"errors"
)

// Message represents a game event message
type Message interface {
	Name() string
}

// Game represents the poker game server
type Game struct {
	tables map[string]*Table
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
		g.tables = make(map[string]*Table)
	}

	table, exists := g.tables[tableID]
	if !exists {
		return nil, errors.New("table not found")
	}

	return table, nil
}
