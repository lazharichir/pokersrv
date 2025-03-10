package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/lazharichir/poker/domain/events"
)

// Lobby represents the poker game lobby
type Lobby struct {
	tables  map[string]*Table
	players map[string]*Player

	// Events
	Events        []events.Event
	eventHandlers []events.EventHandler
}

// IsInLobby checks if a player is in the lobby
func (l *Lobby) IsInLobby(playerID string) bool {
	if l.players == nil {
		return false
	}

	_, exists := l.players[playerID]
	return exists
}

// EntersLobby adds a player to the lobby
func (l *Lobby) EntersLobby(player *Player) error {
	if player == nil {
		return errors.New("player is nil")
	}

	if l.players == nil {
		l.players = make(map[string]*Player)
	}

	if _, exists := l.players[player.ID]; exists {
		return errors.New("player is already in the lobby")
	}

	l.players[player.ID] = player

	l.emitEvent(events.PlayerEnteredLobby{
		PlayerID: player.ID,
		At:       time.Now(),
	})

	return nil
}

func (l *Lobby) LeavesLobby(playerID string) error {
	if l.players == nil {
		l.players = make(map[string]*Player)
	}

	_, exists := l.players[playerID]
	if !exists {
		return errors.New("player not found")
	}

	delete(l.players, playerID)

	l.emitEvent(events.PlayerLeftLobby{
		PlayerID: playerID,
		At:       time.Now(),
	})

	return nil
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

// GetTables returns all tables in the lobby
func (l *Lobby) GetTables() []*Table {
	tables := make([]*Table, 0, len(l.tables))
	for _, table := range l.tables {
		tables = append(tables, table)
	}
	return tables
}

// CreateTable creates a new table in the lobby
func (l *Lobby) CreateTable(name string, maxPlayers int, minBuyIn int) (*Table, error) {
	if l.tables == nil {
		l.tables = make(map[string]*Table)
	}

	// Create table rules
	rules := TableRules{
		AnteValue:                 minBuyIn / 10,   // 10% of min buy-in
		ContinuationBetMultiplier: 2,               // Double ante for continuation bet
		PlayerTimeout:             time.Second * 5, // 5s timeout
		MaxPlayers:                maxPlayers,
	}

	// Create the table
	table := NewTable(name, rules)

	l.tables[table.ID] = table

	return table, nil
}
