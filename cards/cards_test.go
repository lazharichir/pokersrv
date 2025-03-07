package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCardFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Card
		wantErr bool
	}{
		// Valid cards with different suit notations
		{"Ace of Spades Unicode", "A♠", Card{Suit: Spades, Value: Ace}, false},
		{"Ace of Spades lowercase", "As", Card{Suit: Spades, Value: Ace}, false},
		{"Ace of Spades uppercase", "AS", Card{Suit: Spades, Value: Ace}, false},
		{"Ten of Hearts Unicode", "10♥", Card{Suit: Hearts, Value: Ten}, false},
		{"Ten of Hearts lowercase", "10h", Card{Suit: Hearts, Value: Ten}, false},
		{"Ten of Hearts uppercase", "10H", Card{Suit: Hearts, Value: Ten}, false},
		{"Queen of Diamonds Unicode", "Q♦", Card{Suit: Diamonds, Value: Queen}, false},
		{"Queen of Diamonds lowercase", "Qd", Card{Suit: Diamonds, Value: Queen}, false},
		{"Queen of Diamonds uppercase", "QD", Card{Suit: Diamonds, Value: Queen}, false},
		{"Two of Clubs Unicode", "2♣", Card{Suit: Clubs, Value: Two}, false},
		{"Two of Clubs lowercase", "2c", Card{Suit: Clubs, Value: Two}, false},
		{"Two of Clubs uppercase", "2C", Card{Suit: Clubs, Value: Two}, false},

		// All values for a single suit
		{"King of Hearts", "Kh", Card{Suit: Hearts, Value: King}, false},
		{"Jack of Hearts", "Jh", Card{Suit: Hearts, Value: Jack}, false},
		{"Nine of Hearts", "9h", Card{Suit: Hearts, Value: Nine}, false},
		{"Eight of Hearts", "8h", Card{Suit: Hearts, Value: Eight}, false},
		{"Seven of Hearts", "7h", Card{Suit: Hearts, Value: Seven}, false},
		{"Six of Hearts", "6h", Card{Suit: Hearts, Value: Six}, false},
		{"Five of Hearts", "5h", Card{Suit: Hearts, Value: Five}, false},
		{"Four of Hearts", "4h", Card{Suit: Hearts, Value: Four}, false},
		{"Three of Hearts", "3h", Card{Suit: Hearts, Value: Three}, false},

		// Unicode handling edge cases
		{"Proper encoding Spades", "A\u2660", Card{Suit: Spades, Value: Ace}, false},
		{"Proper encoding Hearts", "10\u2665", Card{Suit: Hearts, Value: Ten}, false},
		{"Proper encoding Diamonds", "Q\u2666", Card{Suit: Diamonds, Value: Queen}, false},
		{"Proper encoding Clubs", "2\u2663", Card{Suit: Clubs, Value: Two}, false},

		// Handling of spaces and unusual input format
		{"Input with trailing space", "AS ", Card{}, true},
		{"Input with leading space", " AS", Card{}, true},
		{"Input with mixed case", "aS", Card{Suit: Spades, Value: Ace}, false},

		// Wildcard
		{"Wildcard", "W", Wildcard(), false},
		{"Lowercase wildcard", "w", Card{}, true}, // Only uppercase W is valid

		// Invalid inputs
		{"Too short input", "A", Card{}, true},
		{"Empty input", "", Card{}, true},
		{"Invalid suit", "10X", Card{}, true},
		{"Invalid value", "11S", Card{}, true},
		{"Invalid format", "XX", Card{}, true},
		{"Reverse order", "♠A", Card{}, true},
		{"Special characters", "A$", Card{}, true},
		{"Number too large", "100S", Card{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CardFromString(tt.input)
			if tt.wantErr {
				require.Error(t, err, "CardFromString(%q) should return an error", tt.input)
			} else {
				require.NoError(t, err, "CardFromString(%q) should not return an error", tt.input)
				require.Equal(t, tt.want, got, "CardFromString(%q) should return the correct card", tt.input)
			}
		})
	}
}
