package game

import (
	"github.com/lazharichir/poker/cards"
)

// HandStarted represents the event when a new hand begins.
type HandStarted struct {
	TableID        string
	ButtonPlayerID string
	AnteAmount     int
	PlayerIDs      []string
}

func (e HandStarted) EventName() string { return "hand-started" }

// AntePlacedByPlayer represents the event when a player places an ante bet.
type AntePlacedByPlayer struct {
	TableID  string
	PlayerID string
	Amount   int
}

func (e AntePlacedByPlayer) EventName() string { return "ante-placed-by-player" }

// PlayerHoleCardDealt represents the event when a hole card has been dealt to a player.
type PlayerHoleCardDealt struct {
	TableID  string
	PlayerID string
	Card     cards.Card
}

func (e PlayerHoleCardDealt) EventName() string { return "player-hole-card-dealt" }

// ContinuationBetPlaced represents the event when a player places a continuation bet.
type ContinuationBetPlaced struct {
	TableID  string
	PlayerID string
	Amount   int
}

func (e ContinuationBetPlaced) EventName() string { return "continuation-bet-placed" }

// PlayerFolded represents the event when a player folds.
type PlayerFolded struct {
	TableID  string
	PlayerID string
}

func (e PlayerFolded) EventName() string { return "player-folded" }

// CommunityCardsDealt represents the event when community cards are dealt.
type CommunityCardsDealt struct {
	TableID string
	Cards   []cards.Card
}

func (e CommunityCardsDealt) EventName() string { return "community-cards-dealt" }

// CardDiscarded represents the event when a player discards a community card.
type CardDiscarded struct {
	TableID    string
	PlayerID   string
	Card       cards.Card
	DiscardFee int
}

func (e CardDiscarded) EventName() string { return "card-discarded" }

// CommunityCardSelected represents the event when a player selects a community card.
type CommunityCardSelected struct {
	TableID  string
	PlayerID string
	Card     cards.Card
}

func (e CommunityCardSelected) EventName() string { return "community-card-selected" }

// HandCompleted represents the event when a hand is completed and winners determined.
type HandCompleted struct {
	TableID       string
	FirstPlaceID  string
	FirstPrize    int
	SecondPlaceID string
	SecondPrize   int
}

func (e HandCompleted) EventName() string { return "hand-completed" }
