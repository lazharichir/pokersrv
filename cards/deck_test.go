package cards

import (
	"testing"
)

func TestNewDeck(t *testing.T) {
	deck := NewDeck52()

	if len(deck) != 52 {
		t.Errorf("Expected deck to have 52 cards, got %d", len(deck))
	}
}

func TestShuffleDeck(t *testing.T) {
	originalDeck := NewDeck52()
	shuffledDeck := ShuffleCards(originalDeck)

	// Check same length
	if len(shuffledDeck) != len(originalDeck) {
		t.Errorf("Shuffled deck length %d does not match original deck length %d",
			len(shuffledDeck), len(originalDeck))
	}

	// Check that cards are shuffled (this is probabilistic but very likely)
	differences := 0
	for i := 0; i < len(originalDeck); i++ {
		if shuffledDeck[i] != originalDeck[i] {
			differences++
		}
	}

	if differences == 0 {
		t.Error("Shuffled deck is identical to original deck")
	}
}

func TestDealCard(t *testing.T) {
	deck := NewDeck52()
	initialLength := len(deck)

	card, remainingDeck := DealCard(deck)

	if len(remainingDeck) != initialLength-1 {
		t.Errorf("Expected remaining deck length to be %d, got %d",
			initialLength-1, len(remainingDeck))
	}

	if card.Equals(remainingDeck[0]) {
		t.Error("Dealt card should not be present in remaining deck")
	}
}

func TestDealCards(t *testing.T) {
	deck := NewDeck52()
	initialLength := len(deck)
	count := 5

	dealtCards, remainingDeck := DealCards(deck, count)

	if len(dealtCards) != count {
		t.Errorf("Expected to deal %d cards, got %d", count, len(dealtCards))
	}

	if len(remainingDeck) != initialLength-count {
		t.Errorf("Expected remaining deck length to be %d, got %d",
			initialLength-count, len(remainingDeck))
	}
}
