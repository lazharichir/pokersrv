package cards

// Cards represents a collection of playing cards
type Cards []Card

func (cards Cards) String() string {
	var s string
	for _, c := range cards {
		s += c.String() + " "
	}
	return s
}
