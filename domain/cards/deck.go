package cards

import (
	"math/rand"
	"time"
)

// NewDeck52 creates a standard deck of 52 cards
func NewDeck52() Stack {
	var deck Stack
	suits := []Suit{Spades, Hearts, Diamonds, Clubs}
	values := []Value{Ace, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King}

	for _, suit := range suits {
		for _, value := range values {
			deck.AddCard(Card{Suit: suit, Value: value})
		}
	}

	return deck
}

// ShuffleCards shuffles a deck of cards randomly
func ShuffleCards(cards []Card) []Card {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	shuffled := make([]Card, len(cards))
	copy(shuffled, cards)

	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled
}

// DealCard deals the top card from the deck and returns the card and the remaining deck
func DealCard(deck []Card) (Card, []Card) {
	if len(deck) == 0 {
		return Card{}, nil
	}

	card := deck[0]
	return card, deck[1:]
}

// DealCards deals count cards and returns them with the remaining deck
func DealCards(deck []Card, count int) ([]Card, []Card) {
	if count > len(deck) {
		count = len(deck)
	}

	dealtCards := make([]Card, count)
	copy(dealtCards, deck[:count])

	return dealtCards, deck[count:]
}
