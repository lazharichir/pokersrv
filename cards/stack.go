package cards

// Stack represents multiple cards
type Stack []Card

func (stack *Stack) AddCard(card Card) {
	*stack = append(*stack, card)
}

func (stack *Stack) AddCards(cards ...Card) {
	*stack = append(*stack, cards...)
}

func (stack *Stack) Shuffle() {
	deck := *stack
	shuffled := ShuffleCards(deck)
	*stack = Stack(shuffled)
}

// NewStack creates a new stack with a given number of cards
func NewStack(cards ...Card) Stack {
	return Stack(cards)
}
