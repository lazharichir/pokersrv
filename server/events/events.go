package events

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/lazharichir/poker/domain/events"
	"github.com/lazharichir/poker/server/connection"
)

// EventEnvelope wraps an event with its name for client consumption
type EventEnvelope struct {
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload"`
}

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
	// Convert event to JSON for the payload
	eventPayload, err := json.Marshal(event)
	if err != nil {
		log.Println("Failed to marshal event payload:", err)
		return
	}

	// Create the envelope with name and payload
	envelope := EventEnvelope{
		Name:    event.Name(),
		Payload: eventPayload,
	}

	// Marshal the complete envelope
	envelopeData, err := json.Marshal(envelope)
	if err != nil {
		log.Println("Failed to marshal event envelope:", err)
		return
	}

	log.Println("Dispatching event:", event.Name())

	// Route event based on type
	switch e := event.(type) {
	case events.PlayerEnteredLobby:
		fmt.Println("Dispatching PlayerEnteredLobby event for player:", e.PlayerID)
		sent := d.connMgr.SendToPlayer(e.PlayerID, envelopeData)
		fmt.Println("PlayerEnteredLobby event sent:", sent)

	case events.PlayerLeftLobby:
		d.connMgr.SendToPlayer(e.PlayerID, envelopeData)

	case events.PlayerJoinedTable:
		// Send to all players at the table
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.HandStarted:
		// Send to all players at the table
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.HoleCardDealt:
		// Only send to specific player
		d.connMgr.SendToPlayer(e.PlayerID, envelopeData)

	case events.PlayerFolded:
		// Send to all players at the table
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.PlayerLeftTable:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.PlayerChipsChanged:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.PhaseChanged:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.HandEnded:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.AntePlaced:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.ContinuationBetPlaced:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.CommunityCardSelected:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.PlayerTimedOut:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.HoleCardsDealt:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.CardBurned:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.CommunityCardDealt:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.PlayerTurnStarted:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.BettingRoundStarted:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.BettingRoundEnded:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.CommunitySelectionStarted:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.CommunitySelectionEnded:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.HandsEvaluated:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.ShowdownStarted:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.PlayerShowedHand:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.PotChanged:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.PotBrokenDown:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.PotAmountAwarded:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	case events.SingleWinnerDetermined:
		d.connMgr.SendToTable(e.TableID, envelopeData)

	// Add cases for all event types, determining who should receive each event
	default:
		// For events without special handling, send to all players at the table
		// if we can determine the table ID
		if tableID := events.ExtractTableID(event); tableID != "" {
			d.connMgr.SendToTable(tableID, envelopeData)
		}
	}
}
