package events

import "reflect"

// Helper function to extract table ID from events
func ExtractTableID(event Event) string {
	val := reflect.ValueOf(event)

	// If it's a pointer, get the underlying element
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Check if the value is a struct
	if val.Kind() == reflect.Struct {
		// Try to get the TableID field
		tableID := val.FieldByName("TableID")

		// Check if the field exists and is a string
		if tableID.IsValid() && tableID.Kind() == reflect.String {
			return tableID.String()
		}
	}

	return ""
}
