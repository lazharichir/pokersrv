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

func (stack Stack) String() string {
	var s string
	for _, c := range stack {
		s += c.String() + " "
	}
	return s
}

// NewStack creates a new stack with a given number of cards
func NewStack(cards ...Card) Stack {
	return Stack(cards)
}
