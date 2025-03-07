package cards

// Stack represents multiple cards
type Stack []Card

// NewStack creates a new stack with a given number of cards
func NewStack(cards ...Card) []Card {
	return cards
}
