package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lazharichir/poker/domain"
	"github.com/lazharichir/poker/server/connection"
	"github.com/lazharichir/poker/server/events"
	"github.com/lazharichir/poker/server/handlers"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, implement proper origin checks
	},
}

// Server represents the WebSocket server
type Server struct {
	lobby      *domain.Lobby
	connMgr    *connection.Manager
	cmdRouter  *handlers.CommandRouter
	dispatcher *events.Dispatcher
}

// TableResponse represents a table in API responses
type TableResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	PlayerCount int      `json:"playerCount"`
	Players     []string `json:"players"`
	Status      string   `json:"status"`
	AnteValue   int      `json:"anteValue"`
	CurrentHand string   `json:"currentHand,omitempty"`
}

// CreateTableRequest represents the request to create a new table
type CreateTableRequest struct {
	Name      string `json:"name"`
	AnteValue int    `json:"anteValue"`
}

// corsMiddleware adds CORS headers to all responses
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next(w, r)
	}
}

// NewServer creates a new poker WebSocket server
func NewServer() *Server {
	lobby := &domain.Lobby{}
	connMgr := connection.NewManager()

	dispatcher := events.NewDispatcher(connMgr)
	cmdRouter := handlers.NewCommandRouter(lobby, connMgr)

	// Register dispatcher as event handler for the lobby
	lobby.AddEventHandler(dispatcher.HandleEvent)

	return &Server{
		lobby:      lobby,
		connMgr:    connMgr,
		cmdRouter:  cmdRouter,
		dispatcher: dispatcher,
	}
}

// Start begins the server on the specified port
func (s *Server) Start(port string) error {
	// Start connection manager in its own goroutine
	go s.connMgr.Start()

	// Set up HTTP handlers with CORS middleware
	http.HandleFunc("/ws", s.handleWebSocket)
	http.HandleFunc("/api/tables", corsMiddleware(s.handleGetTables))
	http.HandleFunc("/api/tables/create", corsMiddleware(s.handleCreateTable))

	log.Printf("Starting server on port %s", port)
	return http.ListenAndServe("0.0.0.0:"+port, nil)
}

// handleWebSocket handles incoming WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}

	// Create a new client with a unique ID
	clientID := uuid.NewString()
	log.Printf("New client connected: %s with ID: %s", r.RemoteAddr, clientID)

	client := &connection.Client{
		ID:   clientID,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	// Register with connection manager
	s.connMgr.Register <- client

	// Handle reading and writing in separate goroutines
	go s.readPump(client)
	go s.writePump(client)
	go s.sendHello(client)
}

// readPump reads messages from the WebSocket connection
func (s *Server) readPump(client *connection.Client) {
	defer func() {
		s.connMgr.Unregister <- client
		client.Conn.Close()
	}()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error: %v", err)
			}
			break
		}

		// Process the message through the command router
		if err := s.cmdRouter.HandleCommand(client, message); err != nil {
			log.Printf("Error handling command: %v", err)
			// You could send an error message back to the client here
		}
	}
}

// writePump sends messages to the WebSocket connection
func (s *Server) writePump(client *connection.Client) {
	defer func() {
		client.Conn.Close()
	}()

	for {
		message, ok := <-client.Send
		if !ok {
			// Channel closed
			client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		err := client.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Error writing message: %v", err)
			return
		}
	}
}

func (s *Server) sendHello(client *connection.Client) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := client.Conn.WriteMessage(websocket.TextMessage, []byte("HELLO")); err != nil {
			log.Printf("Error sending HELLO: %v", err)
			return // Exit if we can't write to the client
		}
	}
}

// handleGetTables returns a list of all tables
func (s *Server) handleGetTables(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tables := s.lobby.GetTables()
	tableResponses := make([]TableResponse, 0, len(tables))

	for _, table := range tables {
		players := table.GetPlayers()
		playerIDs := make([]string, 0, len(players))
		for _, player := range players {
			playerIDs = append(playerIDs, player.ID)
		}

		tableResponses = append(tableResponses, TableResponse{
			ID:          table.ID,
			Name:        table.Name,
			PlayerCount: len(players),
			Players:     playerIDs,
			Status:      string(table.Status),
			AnteValue:   table.Rules.AnteValue,
			CurrentHand: table.GetCurrentHandID(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tableResponses)
}

// handleCreateTable creates a new table
func (s *Server) handleCreateTable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var createReq CreateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&createReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if createReq.Name == "" {
		http.Error(w, "Table name is required", http.StatusBadRequest)
		return
	}

	if createReq.AnteValue <= 0 {
		createReq.AnteValue = 10 // Default ante value
	}

	// Calculate min buy-in (10x ante)
	minBuyIn := createReq.AnteValue * 10

	// Create the table
	table, err := s.lobby.CreateTable(createReq.Name, 6, minBuyIn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the created table
	response := TableResponse{
		ID:          table.ID,
		Name:        table.Name,
		PlayerCount: 0,
		Players:     []string{},
		Status:      string(table.Status),
		AnteValue:   table.Rules.AnteValue,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
