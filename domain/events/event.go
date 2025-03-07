package events

// Event is the interface that all domain events must implement.
type Event interface {
	EventName() string // Returns a unique name for the event type
}
