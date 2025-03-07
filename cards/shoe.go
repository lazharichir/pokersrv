package cards

// Shoe represents multiple decks of cards
type Shoe struct {
	Cards Stack
}

// NewShoe creates a new shoe with a given number of decks
func NewShoe(numDecks int) Shoe {
	var cards Stack
	for i := 0; i < numDecks; i++ {
		cards = append(cards, NewDeck()...)
	}
	return Shoe{Cards: cards}
}
