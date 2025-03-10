package events

import (
	"encoding/json"

	"github.com/lazharichir/poker/domain/events"
	"github.com/lazharichir/poker/server/connection"
)

// Dispatcher handles routing events to clients
type Dispatcher struct {
	connMgr *connection.Manager
}

// NewDispatcher creates a new event dispatcher
func NewDispatcher(connMgr *connection.Manager) *Dispatcher {
	return &Dispatcher{
		connMgr: connMgr,
	}
}

// HandleEvent processes domain events and sends them to clients
func (d *Dispatcher) HandleEvent(event events.Event) {
	// Convert event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		// Log error
		return
	}

	// Route event based on type
	switch e := event.(type) {
	case events.PlayerJoinedTable:
		// Send to all players at the table
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.HandStarted:
		// Send to all players at the table
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.HoleCardDealt:
		// Only send to specific player
		d.connMgr.SendToPlayer(e.PlayerID, eventData)

	case events.PlayerFolded:
		// Send to all players at the table
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.PlayerLeftTable:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.PlayerChipsChanged:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.PhaseChanged:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.HandEnded:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.AntePlaced:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.ContinuationBetPlaced:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.CommunityCardSelected:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.PlayerTimedOut:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.HoleCardsDealt:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.CardBurned:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.CommunityCardDealt:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.PlayerTurnStarted:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.BettingRoundStarted:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.BettingRoundEnded:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.CommunitySelectionStarted:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.CommunitySelectionEnded:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.HandsEvaluated:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.ShowdownStarted:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.PlayerShowedHand:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.PotChanged:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.PotBrokenDown:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.PotAmountAwarded:
		d.connMgr.SendToTable(e.TableID, eventData)

	case events.SingleWinnerDetermined:
		d.connMgr.SendToTable(e.TableID, eventData)

	// Add cases for all event types, determining who should receive each event
	default:
		// For events without special handling, send to all players at the table
		// if we can determine the table ID
		if tableID := events.ExtractTableID(event); tableID != "" {
			d.connMgr.SendToTable(tableID, eventData)
		}
	}
}
