package game

// Command represents a game action that can be performed
type Command interface {
	CommandName() string
}

// StartHandCommand starts a new hand at a table
type StartHandCommand struct {
	TableID string
}

func (c StartHandCommand) CommandName() string { return "start-hand" }

// PlaceAnteCommand handles a player placing an ante
type PlaceAnteCommand struct {
	TableID  string
	PlayerID string
}

func (c PlaceAnteCommand) CommandName() string { return "place-ante" }

// PlaceContinuationBetCommand handles a player placing a continuation bet
type PlaceContinuationBetCommand struct {
	TableID  string
	PlayerID string
}

func (c PlaceContinuationBetCommand) CommandName() string { return "place-continuation-bet" }

// FoldCommand handles a player folding their hand
type FoldCommand struct {
	TableID  string
	PlayerID string
}

func (c FoldCommand) CommandName() string { return "fold" }

// DiscardCardCommand handles a player discarding a community card
type DiscardCardCommand struct {
	TableID       string
	PlayerID      string
	CardShorthand string
}

func (c DiscardCardCommand) CommandName() string { return "discard-card" }

// SelectCommunityCardCommand handles a player selecting a community card
type SelectCommunityCardCommand struct {
	TableID       string
	PlayerID      string
	CardShorthand string
}

func (c SelectCommunityCardCommand) CommandName() string { return "select-community-card" }
