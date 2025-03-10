package events

import (
	"time"

	"github.com/lazharichir/poker/cards"
	"github.com/lazharichir/poker/hands"
)

type EventHandler func(event Event)

type Event interface {
	Name() string
	Timestamp() time.Time
}

// Existing events
type UserSat struct {
	TableID string
	UserID  string
	At      time.Time
}

func (u UserSat) Name() string         { return "USER_SAT" }
func (u UserSat) Timestamp() time.Time { return u.At }

type UserStood struct {
	UserID  string
	TableID string
	At      time.Time
}

func (u UserStood) Name() string         { return "USER_STOOD" }
func (u UserStood) Timestamp() time.Time { return u.At }

// Hand Phase Events
type HandStarted struct {
	TableID string
	HandID  string
	Players []string
	At      time.Time
}

func (h HandStarted) Name() string         { return "HAND_STARTED" }
func (h HandStarted) Timestamp() time.Time { return h.At }

type PhaseChanged struct {
	TableID       string
	HandID        string
	PreviousPhase string
	NewPhase      string
	At            time.Time
}

func (p PhaseChanged) Name() string         { return "PHASE_CHANGED" }
func (p PhaseChanged) Timestamp() time.Time { return p.At }

type HandEnded struct {
	TableID  string
	HandID   string
	Duration int64 // in milliseconds
	FinalPot int
	Winners  []string
	At       time.Time
}

func (h HandEnded) Name() string         { return "HAND_ENDED" }
func (h HandEnded) Timestamp() time.Time { return h.At }

// Player Action Events
type AntePlaced struct {
	TableID  string
	HandID   string
	PlayerID string
	Amount   int
	At       time.Time
}

func (a AntePlaced) Name() string         { return "ANTE_PLACED" }
func (a AntePlaced) Timestamp() time.Time { return a.At }

type PlayerFolded struct {
	TableID  string
	HandID   string
	PlayerID string
	Phase    string
	At       time.Time
}

func (p PlayerFolded) Name() string         { return "PLAYER_FOLDED" }
func (p PlayerFolded) Timestamp() time.Time { return p.At }

type ContinuationBetPlaced struct {
	TableID  string
	HandID   string
	PlayerID string
	Amount   int
	At       time.Time
}

func (c ContinuationBetPlaced) Name() string         { return "CONTINUATION_BET_PLACED" }
func (c ContinuationBetPlaced) Timestamp() time.Time { return c.At }

type CommunityCardSelected struct {
	TableID        string
	HandID         string
	PlayerID       string
	Card           string
	SelectionOrder int
	At             time.Time
}

func (c CommunityCardSelected) Name() string         { return "COMMUNITY_CARD_SELECTED" }
func (c CommunityCardSelected) Timestamp() time.Time { return c.At }

type PlayerTimedOut struct {
	TableID       string
	HandID        string
	PlayerID      string
	Phase         string
	DefaultAction string
	At            time.Time
}

func (p PlayerTimedOut) Name() string         { return "PLAYER_TIMED_OUT" }
func (p PlayerTimedOut) Timestamp() time.Time { return p.At }

// Dealing Events
type HoleCardDealt struct {
	TableID  string
	HandID   string
	PlayerID string
	Card     cards.Card
	At       time.Time
}

func (h HoleCardDealt) Name() string         { return "HOLE_CARD_DEALT" }
func (h HoleCardDealt) Timestamp() time.Time { return h.At }

type HoleCardsDealt struct {
	TableID   string
	HandID    string
	DealOrder map[string]int // PlayerID to dealing position
	At        time.Time
}

func (h HoleCardsDealt) Name() string         { return "HOLE_CARDS_DEALT" }
func (h HoleCardsDealt) Timestamp() time.Time { return h.At }

type CardBurned struct {
	TableID string
	HandID  string
	At      time.Time
}

func (c CardBurned) Name() string         { return "CARD_BURNED" }
func (c CardBurned) Timestamp() time.Time { return c.At }

type CommunityCardDealt struct {
	TableID   string
	HandID    string
	CardIndex int
	Card      cards.Card
	At        time.Time
}

func (c CommunityCardDealt) Name() string         { return "COMMUNITY_CARD_DEALT" }
func (c CommunityCardDealt) Timestamp() time.Time { return c.At }

// Turn Management Events
type PlayerTurnStarted struct {
	TableID   string
	HandID    string
	PlayerID  string
	Phase     string
	TimeoutAt int64 // Unix timestamp
	At        time.Time
}

func (p PlayerTurnStarted) Name() string         { return "PLAYER_TURN_STARTED" }
func (p PlayerTurnStarted) Timestamp() time.Time { return p.At }

type BettingRoundStarted struct {
	TableID    string
	HandID     string
	Phase      string
	FirstToAct string // Player ID
	At         time.Time
}

func (b BettingRoundStarted) Name() string         { return "BETTING_ROUND_STARTED" }
func (b BettingRoundStarted) Timestamp() time.Time { return b.At }

type BettingRoundEnded struct {
	TableID   string
	HandID    string
	Phase     string
	TotalBets int
	At        time.Time
}

func (b BettingRoundEnded) Name() string         { return "BETTING_ROUND_ENDED" }
func (b BettingRoundEnded) Timestamp() time.Time { return b.At }

type CommunitySelectionStarted struct {
	TableID   string
	HandID    string
	TimeLimit time.Duration
	At        time.Time
}

func (c CommunitySelectionStarted) Name() string         { return "COMMUNITY_SELECTION_STARTED" }
func (c CommunitySelectionStarted) Timestamp() time.Time { return c.At }

type CommunitySelectionEnded struct {
	TableID string
	HandID  string
	At      time.Time
}

func (c CommunitySelectionEnded) Name() string         { return "COMMUNITY_SELECTION_ENDED" }
func (c CommunitySelectionEnded) Timestamp() time.Time { return c.At }

// Evaluation Events
type HandsEvaluated struct {
	TableID string
	HandID  string
	Results map[string]hands.HandComparisonResult // playerID => HandComparisonResult
	At      time.Time
}

func (h HandsEvaluated) Name() string         { return "HANDS_EVALUATED" }
func (h HandsEvaluated) Timestamp() time.Time { return h.At }

type ShowdownStarted struct {
	TableID       string
	HandID        string
	ActivePlayers []string
	At            time.Time
}

func (s ShowdownStarted) Name() string         { return "SHOWDOWN_STARTED" }
func (s ShowdownStarted) Timestamp() time.Time { return s.At }

type PlayerShowedHand struct {
	TableID                string
	HandID                 string
	PlayerID               string
	HoleCards              cards.Stack
	SelectedCommunityCards cards.Stack
	At                     time.Time
}

func (p PlayerShowedHand) Name() string         { return "PLAYER_SHOWED_HAND" }
func (p PlayerShowedHand) Timestamp() time.Time { return p.At }

// Pot Events
type PotChanged struct {
	TableID        string
	HandID         string
	PreviousAmount int
	NewAmount      int
	At             time.Time
}

func (p PotChanged) Name() string         { return "POT_CHANGED" }
func (p PotChanged) Timestamp() time.Time { return p.At }

type PotBrokenDown struct {
	TableID   string
	HandID    string
	Breakdown map[string]int
	At        time.Time
}

func (p PotBrokenDown) Name() string         { return "POT_BROKEN_DOWN" }
func (p PotBrokenDown) Timestamp() time.Time { return p.At }

type PotAmountAwarded struct {
	TableID  string
	HandID   string
	PlayerID string
	Amount   int
	Reason   string
	At       time.Time
}

func (p PotAmountAwarded) Name() string         { return "POT_AMOUNT_AWARDED" }
func (p PotAmountAwarded) Timestamp() time.Time { return p.At }

type SingleWinnerDetermined struct {
	TableID  string
	HandID   string
	PlayerID string
	Reason   string
	At       time.Time
}

func (s SingleWinnerDetermined) Name() string         { return "SINGLE_WINNER_DETERMINED" }
func (s SingleWinnerDetermined) Timestamp() time.Time { return s.At }
