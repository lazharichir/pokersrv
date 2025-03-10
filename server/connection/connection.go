package connection

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/lazharichir/poker/domain"
)

// Client represents a connected player
type Client struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan []byte
	Player   *domain.Player // Links to domain.Player.ID
	TableIDs []string       // Tables the player is currently on
}

// Manager handles all client connections
type Manager struct {
	clients    map[string]*Client // Map connection IDs to clients
	playerMap  map[string]string  // Map player IDs to connection IDs
	Register   chan *Client
	Unregister chan *Client
	mutex      sync.RWMutex
}

// NewManager creates a new connection manager
func NewManager() *Manager {
	return &Manager{
		clients:    make(map[string]*Client),
		playerMap:  make(map[string]string),
		Register:   make(chan *Client), // Updated to match the capitalized field
		Unregister: make(chan *Client), // Updated to match the capitalized field
	}
}

// Start begins processing connection events
func (m *Manager) Start() {
	for {
		select {
		case client := <-m.Register: // Updated to use capitalized field
			m.mutex.Lock()
			m.clients[client.ID] = client
			if client.Player != nil {
				m.playerMap[client.Player.ID] = client.ID
			}
			m.mutex.Unlock()
		case client := <-m.Unregister:
			m.mutex.Lock()
			if _, ok := m.clients[client.ID]; ok {
				if client.Player != nil {
					delete(m.playerMap, client.Player.ID)
				}
				delete(m.clients, client.ID)
				close(client.Send)
			}
			m.mutex.Unlock()
		}
	}
}

// SendToPlayer sends a message to a specific player
func (m *Manager) SendToPlayer(playerID string, message []byte) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if connID, exists := m.playerMap[playerID]; exists {
		if client, ok := m.clients[connID]; ok {
			client.Send <- message
			return true
		}
	}
	return false
}

// SendToTable sends a message to all players at a table
func (m *Manager) SendToTable(tableID string, message []byte) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, client := range m.clients {
		for _, id := range client.TableIDs {
			if id == tableID {
				client.Send <- message
				break // Send only once even if the client is at the table multiple times
			}
		}
	}
}

// AddTableToClient adds a table ID to a client's tables
func (m *Manager) AddTableToClient(clientID string, tableID string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if client, ok := m.clients[clientID]; ok {
		// Check if table is already in list
		for _, id := range client.TableIDs {
			if id == tableID {
				return true // Already added
			}
		}
		client.TableIDs = append(client.TableIDs, tableID)
		return true
	}
	return false
}

// RemoveTableFromClient removes a table ID from a client's tables
func (m *Manager) RemoveTableFromClient(clientID string, tableID string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if client, ok := m.clients[clientID]; ok {
		for i, id := range client.TableIDs {
			if id == tableID {
				// Remove the table ID by slicing it out
				client.TableIDs = append(client.TableIDs[:i], client.TableIDs[i+1:]...)
				return true
			}
		}
	}
	return false
}

// IsClientAtTable checks if a client is at a specific table
func (m *Manager) IsClientAtTable(clientID string, tableID string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if client, ok := m.clients[clientID]; ok {
		for _, id := range client.TableIDs {
			if id == tableID {
				return true
			}
		}
	}
	return false
}
