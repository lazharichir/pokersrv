package cards

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack_DealCards(t *testing.T) {
	card1 := Card{Suit: Clubs, Value: Ace}
	card2 := Card{Suit: Diamonds, Value: Two}
	card3 := Card{Suit: Hearts, Value: King}
	stack := NewStack(card1, card2, card3)

	dealtCards := stack.DealCards(2)

	assert.Len(t, dealtCards, 2, "Expected 2 cards to be dealt")
	assert.Equal(t, card1, dealtCards[0], "Expected first dealt card to be card1")
	assert.Equal(t, card2, dealtCards[1], "Expected second dealt card to be card2")
	assert.Len(t, stack, 1, "Expected stack to have 1 card remaining")
	assert.Equal(t, card3, stack[0], "Expected remaining card to be card3")
}

func TestStack_DealCard(t *testing.T) {
	card1 := Card{Suit: Clubs, Value: Ace}
	card2 := Card{Suit: Diamonds, Value: Two}
	stack := NewStack(card1, card2)

	dealtCard := stack.DealCard()

	assert.Equal(t, card1, dealtCard, "Expected dealt card to be card1")
	assert.Len(t, stack, 1, "Expected stack to have 1 card remaining")
	assert.Equal(t, card2, stack[0], "Expected remaining card to be card2")
}

func TestStack_BurnCard(t *testing.T) {
	card1 := Card{Suit: Clubs, Value: Ace}
	card2 := Card{Suit: Diamonds, Value: Two}
	stack := NewStack(card1, card2)

	stack.BurnCard()

	assert.Len(t, stack, 1, "Expected stack to have 1 card remaining")
	assert.Equal(t, card2, stack[0], "Expected remaining card to be card2")
}

func TestStack_AddCard(t *testing.T) {
	stack := NewStack()
	card := Card{Suit: Clubs, Value: Ace}

	stack.AddCard(card)

	assert.Len(t, stack, 1, "Expected stack to have 1 card")
	assert.Equal(t, card, stack[0], "Expected card to be card")
}

func TestStack_AddCards(t *testing.T) {
	stack := NewStack()
	card1 := Card{Suit: Clubs, Value: Ace}
	card2 := Card{Suit: Diamonds, Value: Two}

	stack.AddCards(card1, card2)

	assert.Len(t, stack, 2, "Expected stack to have 2 cards")
	assert.Equal(t, card1, stack[0], "Expected first card to be card1")
	assert.Equal(t, card2, stack[1], "Expected second card to be card2")
}

func TestStack_String(t *testing.T) {
	card1 := Card{Suit: Clubs, Value: Ace}
	card2 := Card{Suit: Diamonds, Value: Two}
	stack := NewStack(card1, card2)

	expectedString := "A♣ 2♦"
	assert.Equal(t, expectedString, stack.String(), "Expected string representation to be equal to expectedString")
}

func TestNewStack(t *testing.T) {
	card1 := Card{Suit: Clubs, Value: Ace}
	card2 := Card{Suit: Diamonds, Value: Two}

	stack := NewStack(card1, card2)

	assert.Len(t, stack, 2, "Expected stack to have 2 cards")
	assert.Equal(t, card1, stack[0], "Expected first card to be card1")
	assert.Equal(t, card2, stack[1], "Expected second card to be card2")
}
