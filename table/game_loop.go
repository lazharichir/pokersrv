package table

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/lazharichir/poker/events"
	"github.com/lazharichir/poker/poker"
)

// GameState represents the current state of the game loop
type GameState string

const (
	GameStateIdle              GameState = "idle"
	GameStateWaitingForPlayers GameState = "waiting_for_players"
	GameStateAnteCollection    GameState = "ante_collection"
	GameStateDealingHoleCards  GameState = "dealing_hole_cards"
	GameStateContinuationBets  GameState = "continuation_bets"
	GameStateDealingCommunity  GameState = "dealing_community"
	GameStateDiscardPhase      GameState = "discard_phase"
	GameStateWave1Reveal       GameState = "wave1_reveal"
	GameStateWave2Reveal       GameState = "wave2_reveal"
	GameStateWave3Reveal       GameState = "wave3_reveal"
	GameStateHandEvaluation    GameState = "hand_evaluation"
	GameStateShowdown          GameState = "showdown"
	GameStateHandComplete      GameState = "hand_complete"
)

// PlayerAction represents an action taken by a player
type PlayerAction struct {
	PlayerID string
	Action   string
	Data     interface{}
}

// GameLoop manages the flow of a poker game table, including timeouts and player actions
type GameLoop struct {
	tableID         string
	currentState    GameState
	rules           poker.TableRules
	players         []string // List of player IDs in the current game
	activePlayers   []string // Players still active in the current hand
	actionChan      chan PlayerAction
	stateChan       chan GameState
	ctx             context.Context
	cancel          context.CancelFunc
	eventStore      events.EventStore
	stateUpdateLock sync.Mutex
	stateHandlers   map[GameState]func()
	wg              sync.WaitGroup
	handID          string
}

// NewGameLoop creates a new game loop for the specified table
func NewGameLoop(tableID string, rules poker.TableRules, eventStore events.EventStore) *GameLoop {
	ctx, cancel := context.WithCancel(context.Background())

	gameLoop := &GameLoop{
		tableID:       tableID,
		currentState:  GameStateIdle,
		rules:         rules,
		actionChan:    make(chan PlayerAction, 100), // Buffer for player actions
		stateChan:     make(chan GameState, 10),     // Buffer for state transitions
		ctx:           ctx,
		cancel:        cancel,
		eventStore:    eventStore,
		stateHandlers: make(map[GameState]func()),
	}

	// Register state handlers
	gameLoop.registerStateHandlers()

	return gameLoop
}

// Start begins the game loop for the table
func (g *GameLoop) Start(initialPlayers []string) {
	g.players = initialPlayers
	g.transitionTo(GameStateWaitingForPlayers)

	// Start the main loop in a goroutine
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		g.runLoop()
	}()
}

// Stop stops the game loop
func (g *GameLoop) Stop() {
	g.cancel()
	g.wg.Wait() // Wait for all goroutines to finish
}

// SubmitAction allows players to submit an action to the game
func (g *GameLoop) SubmitAction(playerID string, action string, data interface{}) {
	select {
	case g.actionChan <- PlayerAction{PlayerID: playerID, Action: action, Data: data}:
		// Action submitted
	case <-g.ctx.Done():
		// Context was canceled, game is shutting down
		return
	}
}

// runLoop is the main game loop that processes state changes and player actions
func (g *GameLoop) runLoop() {
	for {
		select {
		case <-g.ctx.Done():
			// Context canceled, shut down the loop
			return

		case newState := <-g.stateChan:
			// Handle state transition
			g.handleStateTransition(newState)

		case action := <-g.actionChan:
			// Process player action
			g.handlePlayerAction(action)
		}
	}
}

// transitionTo changes the game state to a new state
func (g *GameLoop) transitionTo(newState GameState) {
	g.stateUpdateLock.Lock()
	defer g.stateUpdateLock.Unlock()

	// Only process if state is actually changing
	if g.currentState == newState {
		return
	}

	g.currentState = newState

	// Notify state change listeners
	select {
	case g.stateChan <- newState:
		// State change notification sent
	case <-g.ctx.Done():
		return
	}
}

// handleStateTransition processes a state transition by executing the appropriate handler
func (g *GameLoop) handleStateTransition(newState GameState) {
	g.stateUpdateLock.Lock()
	g.currentState = newState
	g.stateUpdateLock.Unlock()

	// Execute the handler for this state
	if handler, exists := g.stateHandlers[newState]; exists {
		handler()
	}
}

// handlePlayerAction processes an action submitted by a player
func (g *GameLoop) handlePlayerAction(action PlayerAction) {
	// Process the action based on the current state
	switch g.currentState {
	case GameStateAnteCollection:
		g.handleAnteAction(action)

	case GameStateContinuationBets:
		g.handleContinuationBetAction(action)

	case GameStateDiscardPhase:
		g.handleDiscardAction(action)

	case GameStateWave1Reveal, GameStateWave2Reveal, GameStateWave3Reveal:
		g.handleCardSelectionAction(action)

	default:
		// Invalid action for the current state
		// Could log or notify player
	}
}

// startNewHand begins a new hand at the table
func (g *GameLoop) startNewHand() {
	// Generate a new hand ID
	g.handID = uuid.NewString()

	// Reset player states
	g.activePlayers = make([]string, len(g.players))
	copy(g.activePlayers, g.players)

	// Publish hand started event
	event := events.HandStarted{
		TableID:        g.tableID,
		ButtonPlayerID: g.chooseButtonPlayer(),
		AnteAmount:     g.rules.AnteValue,
		PlayerIDs:      g.activePlayers,
	}
	g.eventStore.Append(event)

	// Move to ante collection
	g.transitionTo(GameStateAnteCollection)
}

// chooseButtonPlayer selects a player to be the button (dealer)
func (g *GameLoop) chooseButtonPlayer() string {
	if len(g.players) == 0 {
		return ""
	}
	return g.players[0] // For simplicity, we start with first player
}

// registerStateHandlers sets up handlers for each game state
func (g *GameLoop) registerStateHandlers() {
	// Register handlers for each state
	g.stateHandlers = map[GameState]func(){
		GameStateWaitingForPlayers: g.handleWaitingForPlayersState,
		GameStateAnteCollection:    g.handleAnteCollectionState,
		GameStateDealingHoleCards:  g.handleDealingHoleCardsState,
		GameStateContinuationBets:  g.handleContinuationBetsState,
		GameStateDealingCommunity:  g.handleDealingCommunityState,
		GameStateDiscardPhase:      g.handleDiscardPhaseState,
		GameStateWave1Reveal:       g.handleWave1RevealState,
		GameStateWave2Reveal:       g.handleWave2RevealState,
		GameStateWave3Reveal:       g.handleWave3RevealState,
		GameStateHandEvaluation:    g.handleHandEvaluationState,
		GameStateShowdown:          g.handleShowdownState,
		GameStateHandComplete:      g.handleHandCompleteState,
	}
}
