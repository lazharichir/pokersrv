package cards

import "strings"

// Stack represents multiple cards
type Stack []Card

func (stack *Stack) DealCard() Card {
	deck := *stack
	card := deck[0]
	*stack = deck[1:]
	return card
}

func (stack *Stack) DealCards(n int) Stack {
	deck := *stack
	cards := deck[:n]
	*stack = deck[n:]
	return Stack(cards)
}

func (stack *Stack) BurnCard() {
	deck := *stack
	*stack = deck[1:]
}

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
	return strings.TrimSpace(s)
}

func (stack Stack) Count() int {
	return len(stack)
}

func (stack Stack) IsEmpty() bool {
	return len(stack) == 0
}

// NewStack creates a new stack with a given number of cards
func NewStack(cards ...Card) Stack {
	return Stack(cards)
}
