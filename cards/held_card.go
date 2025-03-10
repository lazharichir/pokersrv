package cards

type CardVisibility string

const (
	FaceDown      CardVisibility = "down"  // Nobody can see
	FaceUpToOwner CardVisibility = "owner" // Only the owner can see
	FaceUpToAll   CardVisibility = "all"   // Everyone can see
)

// HeldCard represents a card that's in play with visibility information
type HeldCard struct {
	Card
	Visibility CardVisibility
}

// SetVisibility sets the visibility of the card
func (c *HeldCard) SetVisibility(visibility CardVisibility) {
	c.Visibility = visibility
}

// Hide sets the card as face down
func (c *HeldCard) Hide() {
	c.SetVisibility(FaceDown)
}

// VisibleToOwner sets the card as visible to the owner
func (c *HeldCard) VisibleToOwner() {
	c.SetVisibility(FaceUpToOwner)
}

// VisibleToAll sets the card as face up to all
func (c *HeldCard) VisibleToAll() {
	c.SetVisibility(FaceUpToAll)
}

// NewHeldCard creates a new held card with the specified visibility
func NewHeldCard(card Card, visibility CardVisibility) HeldCard {
	return HeldCard{
		Card:       card,
		Visibility: visibility,
	}
}

type HeldStack []HeldCard

// NewHeldStack creates a new held stack
func NewHeldStack(cards ...HeldCard) HeldStack {
	return HeldStack(cards)
}

// Add adds a card to the stack
func (s *HeldStack) Add(card HeldCard) {
	*s = append(*s, card)
}

// Remove removes a card from the stack
func (s *HeldStack) Remove(card HeldCard) {
	for i, c := range *s {
		if c.Card.Equals(card.Card) {
			*s = append((*s)[:i], (*s)[i+1:]...)
			return
		}
	}
}
