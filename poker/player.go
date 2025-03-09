package poker

// Player represents a player in the game
type Player struct {
	ID      string
	Name    string
	Status  string
	Balance int
}

// AddToBalance adds amount to player balance
func (p *Player) AddToBalance(amount int) {
	p.Balance += amount
}

// RemoveFromBalance removes amount from player balance
func (p *Player) RemoveFromBalance(amount int) {
	p.Balance -= amount
}
