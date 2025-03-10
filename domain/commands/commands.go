package commands

import "github.com/lazharichir/poker/cards"

type Command interface {
	Name() string
}

type PlayerSeats struct {
	PlayerID string
	TableID  string
}

func (p PlayerSeats) Name() string { return "PLAYER_SEATS" }

type PlayerLeavesTable struct {
	PlayerID string
	TableID  string
}

func (p PlayerLeavesTable) Name() string { return "PLAYER_LEAVES_TABLE" }

type PlayerBuysIn struct {
	PlayerID string
	TableID  string
	Amount   int
}

func (p PlayerBuysIn) Name() string { return "PLAYER_BUYS_IN" }

type PlayerFolds struct {
	PlayerID string
	TableID  string
	HandID   string
}

func (p PlayerFolds) Name() string { return "PLAYER_FOLDS" }

type PlayerPlacesAnte struct {
	PlayerID string
	TableID  string
	HandID   string
	Amount   int
}

func (p PlayerPlacesAnte) Name() string { return "PLAYER_PLACES_ANTE" }

type PlayerPlacesContinuationBet struct {
	PlayerID string
	TableID  string
	HandID   string
	Amount   int
}

func (p PlayerPlacesContinuationBet) Name() string { return "PLAYER_PLACES_CONTINUATION_BET" }

type PlayerSelectsCommunityCard struct {
	PlayerID string
	TableID  string
	HandID   string
	Card     cards.Card
}

func (p PlayerSelectsCommunityCard) Name() string { return "PLAYER_SELECTS_COMMUNITY_CARD" }
