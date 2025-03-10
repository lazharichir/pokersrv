package server

import (
	"log"
	"net/http"

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

	// Set up WebSocket handler
	http.HandleFunc("/ws", s.handleWebSocket)

	log.Printf("Starting WebSocket server on port %s", port)
	return http.ListenAndServe(":"+port, nil)
}

// handleWebSocket handles incoming WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}

	// Create a new client
	client := &connection.Client{
		ID:   uuid.NewString(),
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	// Register with connection manager
	s.connMgr.Register <- client

	// Handle reading and writing in separate goroutines
	go s.readPump(client)
	go s.writePump(client)
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
