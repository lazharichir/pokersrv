package poker

import (
	"github.com/lazharichir/poker/cards"
	"github.com/lazharichir/poker/hands"
)

// HandComparisonResult represents the result of comparing multiple hands
type HandComparisonResult struct {
	PlayerID   string
	HandRank   hands.HandRank
	HandCards  cards.Stack
	IsWinner   bool
	PlaceIndex int // 0 for first place, 1 for second place, etc.
}

// EvaluatePlayerHands uses the hands package to evaluate and compare player hands
func EvaluatePlayerHands(playerCards map[string]cards.Stack) ([]HandComparisonResult, error) {
	// Use the hands package to compare all the hands
	results := hands.CompareHands(playerCards)

	// Convert from hands package results to our format
	handResults := make([]HandComparisonResult, len(results))
	for i, result := range results {
		handResults[i] = HandComparisonResult{
			PlayerID:   result.PlayerID,
			HandRank:   result.HandRank,
			HandCards:  result.HandCards,
			IsWinner:   result.IsWinner,
			PlaceIndex: result.PlaceIndex,
		}
	}

	return handResults, nil
}

// Helper function to be used by Hand.comparePlayerHands
func (h *Hand) comparePlayerHands(playerCards map[string]cards.Stack) ([]HandComparisonResult, error) {
	return EvaluatePlayerHands(playerCards)
}
