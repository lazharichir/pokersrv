package cards

import "fmt"

// CardFromString creates a card from a string representation
// e.g., "10♠" or "10s" or "10S" -> Card{Suit: Spades, Value: Ten}
// e.g., "W" -> Card{Suit: "", Value: ""}
func CardFromString(s string) (Card, error) {
	if s == "W" {
		return Wildcard(), nil
	}

	if len(s) < 2 {
		return Card{}, fmt.Errorf("invalid card shorthand: %s", s)
	}

	var suit Suit
	switch s[len(s)-1:] {
	case "♠", "s", "S":
		suit = Spades
	case "♥", "h", "H":
		suit = Hearts
	case "♦", "d", "D":
		suit = Diamonds
	case "♣", "c", "C":
		suit = Clubs
	default:
		return Card{}, fmt.Errorf("invalid card suit: %s", s[len(s)-1:])
	}

	var value Value
	switch s[:len(s)-1] {
	case "A":
		value = Ace
	case "K":
		value = King
	case "Q":
		value = Queen
	case "J":
		value = Jack
	case "10":
		value = Ten
	case "9":
		value = Nine
	case "8":
		value = Eight
	case "7":
		value = Seven
	case "6":
		value = Six
	case "5":
		value = Five
	case "4":
		value = Four
	case "3":
		value = Three
	case "2":
		value = Two
	default:
		return Card{}, fmt.Errorf("invalid card value: %s", s[:len(s)-1])
	}

	return Card{Suit: suit, Value: value}, nil
}

// Suit represents a card suit
type Suit string

const (
	Spades   Suit = "♠"
	Hearts   Suit = "♥"
	Diamonds Suit = "♦"
	Clubs    Suit = "♣"
)

// Value represents a card value
type Value string

const (
	Ace   Value = "A"
	King  Value = "K"
	Queen Value = "Q"
	Jack  Value = "J"
	Ten   Value = "10"
	Nine  Value = "9"
	Eight Value = "8"
	Seven Value = "7"
	Six   Value = "6"
	Five  Value = "5"
	Four  Value = "4"
	Three Value = "3"
	Two   Value = "2"
)

// Card represents a playing card
type Card struct {
	Suit  Suit
	Value Value
}

// String returns the string representation of a card
func (c Card) String() string {
	return fmt.Sprintf("%s%s", c.Value, c.Suit)
}

// IsWildcard checks if the card is a wildcard
func (c Card) IsWildcard() bool {
	return c.Suit == "" && c.Value == ""
}

// Equals checks if two cards are equal
func (c Card) Equals(other Card) bool {
	return c.Suit == other.Suit && c.Value == other.Value
}

// Wildcard creates a wildcard card
func Wildcard() Card {
	return Card{}
}
