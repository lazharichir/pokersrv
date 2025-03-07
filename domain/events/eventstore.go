package events

import (
	"fmt"
	"sync"
)

// EventStore is the interface for storing and retrieving events.
type EventStore interface {
	Append(event Event) error
	LoadEvents(tableID string) ([]Event, error)
}

// InMemoryEventStore is an in-memory implementation of the EventStore interface.
type InMemoryEventStore struct {
	events map[string][]Event
	mutex  sync.RWMutex
}

// NewInMemoryEventStore creates a new in-memory event store.
func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events: make(map[string][]Event),
	}
}

// Append adds a new event to the store.
func (s *InMemoryEventStore) Append(event Event) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Get tableID from the event based on its type
	var tableID string
	switch e := event.(type) {
	case HandStarted:
		tableID = e.TableID
	case AntePlacedByPlayer:
		tableID = e.TableID
	case PlayerHoleCardDealt:
		tableID = e.TableID
	case ContinuationBetPlaced:
		tableID = e.TableID
	case PlayerFolded:
		tableID = e.TableID
	case CommunityCardsDealt:
		tableID = e.TableID
	case CardDiscarded:
		tableID = e.TableID
	case CommunityCardSelected:
		tableID = e.TableID
	case HandCompleted:
		tableID = e.TableID
	default:
		return fmt.Errorf("unknown event type: %T", e)
	}

	if tableID == "" {
		return fmt.Errorf("event has no tableID")
	}

	if _, exists := s.events[tableID]; !exists {
		s.events[tableID] = make([]Event, 0)
	}

	s.events[tableID] = append(s.events[tableID], event)
	return nil
}

// LoadEvents retrieves all events for the given tableID.
func (s *InMemoryEventStore) LoadEvents(tableID string) ([]Event, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if events, exists := s.events[tableID]; exists {
		// Make a copy to avoid potential race conditions
		result := make([]Event, len(events))
		copy(result, events)
		return result, nil
	}

	// Return empty slice if no events found
	return []Event{}, nil
}
