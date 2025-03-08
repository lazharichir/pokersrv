package events

import "reflect"

// Event is the interface that all domain events must implement.
type Event interface {
	EventName() string // Returns a unique name for the event type
}

func GetTableID(event Event) string {
	val := reflect.ValueOf(event)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	field := val.FieldByName("TableID")
	if field.IsValid() && field.Kind() == reflect.String {
		return field.String()
	}
	return ""
}
