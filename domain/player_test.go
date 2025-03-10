package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddToBalance(t *testing.T) {
	player := &Player{
		ID:      "1",
		Name:    "Test Player",
		Status:  "active",
		Balance: 100,
	}

	player.AddToBalance(50)

	assert.Equal(t, 150, player.Balance)
}

func TestRemoveFromBalance(t *testing.T) {
	player := &Player{
		ID:      "1",
		Name:    "Test Player",
		Status:  "active",
		Balance: 100,
	}

	player.RemoveFromBalance(50)

	assert.Equal(t, 50, player.Balance)
}
