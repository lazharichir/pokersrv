package events

import (
	"time"

	"github.com/lazharichir/poker/cards"
)

type EventHandler func(event Event)

type Event interface {
	Name() string
}

// Existing events
type UserSat struct {
	UserID  string
	TableID string
}

func (u UserSat) Name() string { return "USER_SAT" }

type UserStood struct {
	UserID  string
	TableID string
}

func (u UserStood) Name() string { return "USER_STOOD" }

type HandStarted struct {
	TableID string
	HandID  string
	Players []string
}

func (h HandStarted) Name() string { return "HAND_STARTED" }

// Game Structure Events
type PhaseChanged struct {
	HandID        string
	PreviousPhase string
	NewPhase      string
}

func (p PhaseChanged) Name() string { return "PHASE_CHANGED" }

type HandEnded struct {
	HandID   string
	Duration int64 // in milliseconds
	FinalPot int
	Winners  []string
}

func (h HandEnded) Name() string { return "HAND_ENDED" }

// Player Action Events
type AntePlaced struct {
	HandID   string
	PlayerID string
	Amount   int
}

func (a AntePlaced) Name() string { return "ANTE_PLACED" }

type PlayerFolded struct {
	HandID   string
	PlayerID string
	Phase    string
}

func (p PlayerFolded) Name() string { return "PLAYER_FOLDED" }

type ContinuationBetPlaced struct {
	HandID   string
	PlayerID string
	Amount   int
}

func (c ContinuationBetPlaced) Name() string { return "CONTINUATION_BET_PLACED" }

type CommunityCardSelected struct {
	HandID         string
	PlayerID       string
	CardID         string
	SelectionOrder int
}

func (c CommunityCardSelected) Name() string { return "COMMUNITY_CARD_SELECTED" }

type PlayerTimedOut struct {
	HandID        string
	PlayerID      string
	Phase         string
	DefaultAction string
}

func (p PlayerTimedOut) Name() string { return "PLAYER_TIMED_OUT" }

// Dealing Events
type HoleCardsDealt struct {
	HandID    string
	DealOrder map[string]int // PlayerID to dealing position
}

func (h HoleCardsDealt) Name() string { return "HOLE_CARDS_DEALT" }

type CardBurned struct {
	HandID string
}

func (c CardBurned) Name() string { return "CARD_BURNED" }

type CommunityCardDealt struct {
	HandID    string
	CardIndex int
	Card      cards.Card
}

func (c CommunityCardDealt) Name() string { return "COMMUNITY_CARD_DEALT" }

// Turn Management Events
type PlayerTurnStarted struct {
	HandID    string
	PlayerID  string
	Phase     string
	TimeoutAt int64 // Unix timestamp
}

func (p PlayerTurnStarted) Name() string { return "PLAYER_TURN_STARTED" }

type BettingRoundStarted struct {
	HandID     string
	Phase      string
	FirstToAct string // Player ID
}

func (b BettingRoundStarted) Name() string { return "BETTING_ROUND_STARTED" }

type BettingRoundEnded struct {
	HandID    string
	Phase     string
	TotalBets int
}

func (b BettingRoundEnded) Name() string { return "BETTING_ROUND_ENDED" }

type CommunitySelectionStarted struct {
	HandID    string
	TimeLimit time.Duration
}

func (c CommunitySelectionStarted) Name() string { return "COMMUNITY_SELECTION_STARTED" }

type CommunitySelectionEnded struct {
	HandID string
}

func (c CommunitySelectionEnded) Name() string { return "COMMUNITY_SELECTION_ENDED" }

// Evaluation Events
type HandsEvaluated struct {
	HandID  string
	Results map[string]int
}

func (h HandsEvaluated) Name() string { return "HANDS_EVALUATED" }

type ShowdownStarted struct {
	HandID        string
	ActivePlayers []string
}

func (s ShowdownStarted) Name() string { return "SHOWDOWN_STARTED" }

type PlayerShowedHand struct {
	HandID         string
	PlayerID       string
	HoleCards      cards.Stack
	CommunityCards cards.Stack
	Hand           cards.Stack
}

func (p PlayerShowedHand) Name() string { return "PLAYER_SHOWED_HAND" }

// Pot Events
type PotChanged struct {
	HandID         string
	PreviousAmount int
	NewAmount      int
}

func (p PotChanged) Name() string { return "POT_CHANGED" }

type PotBrokenDown struct {
	HandID    string
	Breakdown map[string]int
}

func (p PotBrokenDown) Name() string { return "POT_BROKEN_DOWN" }

type PotAmountAwarded struct {
	HandID   string
	PlayerID string
	Amount   int
	Reason   string
}

func (p PotAmountAwarded) Name() string { return "POT_AMOUNT_AWARDED" }

type SingleWinnerDetermined struct {
	HandID   string
	PlayerID string
	Reason   string
}

func (s SingleWinnerDetermined) Name() string { return "SINGLE_WINNER_DETERMINED" }
