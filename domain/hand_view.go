package domain

import (
	"time"

	"github.com/lazharichir/poker/cards"
)

// HandView represents a player's view of a hand
type HandView struct {
	ID             string
	Phase          HandPhase
	TableID        string
	PlayerID       string
	MyTurn         bool
	MyRole         string // "button", "active", "waiting", etc.
	ButtonPosition int
	MyPosition     int

	MyHoleCards    cards.Stack
	OtherPlayers   []PlayerView
	CommunityCards cards.Stack

	Pot       int
	MyChips   int
	AnteValue int

	ActionTimeout    time.Time     // When the current player's turn will timeout
	AvailableActions []string      // Actions the player can take now
	Events           []PublicEvent // Recent events visible to this player
}

type PlayerView struct {
	ID                    string
	Name                  string
	Position              int
	Chips                 int
	HasFolded             bool
	IsActive              bool
	IsCurrent             bool
	IsButton              bool
	HasCards              bool
	HoleCards             cards.Stack // Will be hidden unless it's the viewing player or showdown
	AnteStatus            string      // "paid", "not_paid", "folded"
	ContinuationBetStatus string      // "bet", "not_bet", "folded"
}

type PublicEvent struct {
	Type      string
	PlayerID  string
	Timestamp time.Time
	// Only include event data safe to share with all players
}

// BuildPlayerView constructs a view of the hand specific to a player
func (h *Hand) BuildPlayerView(playerID string) HandView {
	view := HandView{
		ID:             h.ID,
		Phase:          h.Phase,
		TableID:        h.TableID,
		PlayerID:       playerID,
		MyTurn:         h.IsPlayerTheCurrentBettor(playerID),
		ButtonPosition: h.ButtonPosition,
		CommunityCards: h.CommunityCards,
		Pot:            h.Pot,
		AnteValue:      h.TableRules.AnteValue,
	}

	// Set player's hole cards if they exist
	if cards, exists := h.HoleCards[playerID]; exists {
		view.MyHoleCards = cards
	}

	// Find player position
	for i, player := range h.Players {
		if player.ID == playerID {
			view.MyPosition = i
			break
		}
	}

	// Set player's role
	if view.MyPosition == h.ButtonPosition {
		view.MyRole = "button"
	} else if h.IsPlayerActive(playerID) {
		view.MyRole = "active"
	} else {
		view.MyRole = "spectator"
	}

	// Set player's chips
	view.MyChips = h.Table.GetlayerBuyIn(playerID)

	// Determine available actions based on game state and player's turn
	view.AvailableActions = h.getAvailableActions(playerID)

	// Build other player views
	view.OtherPlayers = make([]PlayerView, 0, len(h.Players))
	for i, player := range h.Players {
		isCurrentPlayer := player.ID == playerID
		if !isCurrentPlayer {
			pView := PlayerView{
				ID:        player.ID,
				Name:      player.Name,
				Position:  i,
				Chips:     h.Table.GetlayerBuyIn(player.ID),
				HasFolded: !h.IsPlayerActive(player.ID),
				IsActive:  h.IsPlayerActive(player.ID),
				IsCurrent: h.IsPlayerTheCurrentBettor(player.ID),
				IsButton:  i == h.ButtonPosition,
				HasCards:  len(h.HoleCards[player.ID]) > 0,
			}

			// Only show other players' cards during showdown
			if h.Phase == HandPhase_HandReveal {
				pView.HoleCards = h.HoleCards[player.ID]
			}

			// Set ante status
			if _, paid := h.AntesPaid[player.ID]; paid {
				pView.AnteStatus = "paid"
			} else if h.IsPlayerActive(player.ID) {
				pView.AnteStatus = "not_paid"
			} else {
				pView.AnteStatus = "folded"
			}

			// Set continuation bet status
			if _, bet := h.ContinuationBets[player.ID]; bet {
				pView.ContinuationBetStatus = "bet"
			} else if h.IsPlayerActive(player.ID) {
				pView.ContinuationBetStatus = "not_bet"
			} else {
				pView.ContinuationBetStatus = "folded"
			}

			view.OtherPlayers = append(view.OtherPlayers, pView)
		}
	}

	// Filter events for this player's view
	view.Events = h.filterEventsForPlayer(playerID)

	return view
}

// getAvailableActions determines what actions a player can take in the current state
func (h *Hand) getAvailableActions(playerID string) []string {
	actions := []string{}

	if !h.IsPlayerActive(playerID) {
		return actions // No actions for inactive players
	}

	if !h.IsPlayerTheCurrentBettor(playerID) {
		return actions // No actions when it's not the player's turn
	}

	switch h.Phase {
	case HandPhase_Antes:
		if !h.hasAlreadyPlacedAnte(playerID) {
			actions = append(actions, "place_ante")
		}

	case HandPhase_Continuation:
		if !h.hasAlreadyPlacedContinuationBet(playerID) {
			actions = append(actions, "place_continuation_bet", "fold")
		}

	case HandPhase_CommunitySelection:
		// Player can select up to 3 cards
		if h.CommunitySelections[playerID] == nil || len(h.CommunitySelections[playerID]) < 3 {
			actions = append(actions, "select_card")
		}
	}

	return actions
}

// filterEventsForPlayer returns events relevant to this player
func (h *Hand) filterEventsForPlayer(playerID string) []PublicEvent {
	// Implementation would filter out private information from events
	// and only return recent relevant events
	// ...
	return []PublicEvent{}
}
