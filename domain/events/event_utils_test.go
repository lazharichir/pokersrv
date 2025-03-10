package events_test

import (
	"testing"
	"time"

	"github.com/lazharichir/poker/domain/events"
	"github.com/stretchr/testify/assert"
)

type noTableID struct {
	OtherField string
}

func (noTableID) Name() string         { return "noTableID" }
func (noTableID) Timestamp() time.Time { return time.Now() }

func TestExtractTableID(t *testing.T) {
	t.Run("struct with TableID field", func(t *testing.T) {
		e := events.PlayerFolded{TableID: "table123"}
		id := events.ExtractTableID(e)
		assert.Equal(t, "table123", id)
	})

	t.Run("pointer to struct with TableID field", func(t *testing.T) {
		e := &events.PlayerFolded{TableID: "tablePointer"}
		id := events.ExtractTableID(e)
		assert.Equal(t, "tablePointer", id)
	})

	t.Run("struct without TableID field", func(t *testing.T) {

		e := noTableID{OtherField: "noID"}
		id := events.ExtractTableID(e)
		assert.Equal(t, "", id)
	})

	t.Run("pointer to struct without TableID field", func(t *testing.T) {
		e := &noTableID{OtherField: "stillNoID"}
		id := events.ExtractTableID(e)
		assert.Equal(t, "", id)
	})
}
