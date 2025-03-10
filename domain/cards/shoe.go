package cards

// Shoe represents multiple decks of cards which can be dealt from
type Shoe struct {
	Undealt Stack
	Dealt   Stack
}

func (shoe Shoe) Shuffle() {
	shoe.Undealt.Shuffle()
}

// NewShoe creates a new shoe with a given number of decks
func NewShoe(numDecks int) Shoe {
	var cards Stack
	for i := 0; i < numDecks; i++ {
		cards = append(cards, NewDeck52()...)
	}
	return Shoe{Undealt: cards, Dealt: NewStack()}
}
