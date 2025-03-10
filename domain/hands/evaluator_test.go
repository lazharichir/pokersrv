package hands

import (
	"testing"

	"github.com/lazharichir/poker/domain/cards"
	"github.com/stretchr/testify/assert"
)

func TestCompareHands_EmptyInput(t *testing.T) {
	result := CompareHands(map[string]cards.Stack{})
	assert.Nil(t, result, "Expected nil result for empty input")
}

func TestCompareHands_SinglePlayer(t *testing.T) {
	playerCards := map[string]cards.Stack{
		"player1": {
			{Suit: cards.Hearts, Value: cards.Ace},
			{Suit: cards.Hearts, Value: cards.King},
			{Suit: cards.Hearts, Value: cards.Queen},
			{Suit: cards.Hearts, Value: cards.Jack},
			{Suit: cards.Hearts, Value: cards.Ten},
		},
	}

	result := CompareHands(playerCards)

	assert.Equal(t, 1, len(result), "Expected 1 result for single player")
	assert.Equal(t, "player1", result[0].PlayerID)
	assert.Equal(t, RoyalFlush, result[0].HandRank)
	assert.True(t, result[0].IsWinner)
	assert.Equal(t, 0, result[0].PlaceIndex)
}

func TestCompareHands_MultiplePlayersWithClearWinner(t *testing.T) {
	playerCards := map[string]cards.Stack{
		"player1": { // Royal Flush
			{Suit: cards.Hearts, Value: cards.Ace},
			{Suit: cards.Hearts, Value: cards.King},
			{Suit: cards.Hearts, Value: cards.Queen},
			{Suit: cards.Hearts, Value: cards.Jack},
			{Suit: cards.Hearts, Value: cards.Ten},
		},
		"player2": { // Straight Flush
			{Suit: cards.Spades, Value: cards.Nine},
			{Suit: cards.Spades, Value: cards.Eight},
			{Suit: cards.Spades, Value: cards.Seven},
			{Suit: cards.Spades, Value: cards.Six},
			{Suit: cards.Spades, Value: cards.Five},
		},
		"player3": { // Four of a Kind
			{Suit: cards.Hearts, Value: cards.Seven},
			{Suit: cards.Diamonds, Value: cards.Seven},
			{Suit: cards.Clubs, Value: cards.Seven},
			{Suit: cards.Spades, Value: cards.Seven},
			{Suit: cards.Hearts, Value: cards.King},
		},
	}

	result := CompareHands(playerCards)

	assert.Equal(t, 3, len(result), "Expected 3 results")

	// Check first place
	assert.Equal(t, "player1", result[0].PlayerID)
	assert.Equal(t, RoyalFlush, result[0].HandRank)
	assert.True(t, result[0].IsWinner)
	assert.Equal(t, 0, result[0].PlaceIndex)

	// Check second place
	assert.Equal(t, "player2", result[1].PlayerID)
	assert.Equal(t, StraightFlush, result[1].HandRank)
	assert.False(t, result[1].IsWinner)
	assert.Equal(t, 1, result[1].PlaceIndex)

	// Check third place
	assert.Equal(t, "player3", result[2].PlayerID)
	assert.Equal(t, FourOfAKind, result[2].HandRank)
	assert.False(t, result[2].IsWinner)
	assert.Equal(t, 2, result[2].PlaceIndex)
}

func TestCompareHands_TiedPlayers(t *testing.T) {
	playerCards := map[string]cards.Stack{
		"player1": { // Flush with A-K-Q-J-9
			{Suit: cards.Hearts, Value: cards.Ace},
			{Suit: cards.Hearts, Value: cards.King},
			{Suit: cards.Hearts, Value: cards.Queen},
			{Suit: cards.Hearts, Value: cards.Jack},
			{Suit: cards.Hearts, Value: cards.Nine},
		},
		"player2": { // Same flush with A-K-Q-J-9
			{Suit: cards.Spades, Value: cards.Ace},
			{Suit: cards.Spades, Value: cards.King},
			{Suit: cards.Spades, Value: cards.Queen},
			{Suit: cards.Spades, Value: cards.Jack},
			{Suit: cards.Spades, Value: cards.Nine},
		},
		"player3": { // Lower flush A-K-Q-J-8
			{Suit: cards.Diamonds, Value: cards.Ace},
			{Suit: cards.Diamonds, Value: cards.King},
			{Suit: cards.Diamonds, Value: cards.Queen},
			{Suit: cards.Diamonds, Value: cards.Jack},
			{Suit: cards.Diamonds, Value: cards.Eight},
		},
	}

	result := CompareHands(playerCards)

	assert.Equal(t, 3, len(result), "Expected 3 results")

	// Check player1 and player2 are tied for first place
	assert.Equal(t, 0, result[0].PlaceIndex)
	assert.Equal(t, 0, result[1].PlaceIndex) // Same place index indicates tie
	assert.True(t, result[0].IsWinner)
	assert.True(t, result[1].IsWinner)

	// Check player3 is in third place
	assert.Equal(t, "player3", result[2].PlayerID)
	assert.Equal(t, 2, result[2].PlaceIndex)
	assert.False(t, result[2].IsWinner)
}

func TestCompareHands_MoreThanFiveCards(t *testing.T) {
	playerCards := map[string]cards.Stack{
		"player1": { // 7 cards, best 5 form a flush
			{Suit: cards.Hearts, Value: cards.Ace},
			{Suit: cards.Hearts, Value: cards.King},
			{Suit: cards.Hearts, Value: cards.Queen},
			{Suit: cards.Hearts, Value: cards.Jack},
			{Suit: cards.Hearts, Value: cards.Nine},
			{Suit: cards.Spades, Value: cards.Seven},
			{Suit: cards.Diamonds, Value: cards.Two},
		},
		"player2": { // 7 cards, best 5 form a straight
			{Suit: cards.Spades, Value: cards.Six},
			{Suit: cards.Hearts, Value: cards.Five},
			{Suit: cards.Diamonds, Value: cards.Four},
			{Suit: cards.Clubs, Value: cards.Three},
			{Suit: cards.Hearts, Value: cards.Two},
			{Suit: cards.Spades, Value: cards.King},
			{Suit: cards.Diamonds, Value: cards.Queen},
		},
	}

	result := CompareHands(playerCards)

	assert.Equal(t, 2, len(result), "Expected 2 results")
	assert.Equal(t, "player1", result[0].PlayerID)
	assert.Equal(t, Flush, result[0].HandRank)
	assert.Equal(t, "player2", result[1].PlayerID)
	assert.Equal(t, Straight, result[1].HandRank)
}

func TestCompareHands_InsufficientCards(t *testing.T) {
	playerCards := map[string]cards.Stack{
		"player1": { // Only 4 cards, not enough for a hand
			{Suit: cards.Hearts, Value: cards.Ace},
			{Suit: cards.Hearts, Value: cards.King},
			{Suit: cards.Hearts, Value: cards.Queen},
			{Suit: cards.Hearts, Value: cards.Jack},
		},
		"player2": { // 5 cards, enough for a hand
			{Suit: cards.Spades, Value: cards.Six},
			{Suit: cards.Hearts, Value: cards.Five},
			{Suit: cards.Diamonds, Value: cards.Four},
			{Suit: cards.Clubs, Value: cards.Three},
			{Suit: cards.Hearts, Value: cards.Two},
		},
	}

	result := CompareHands(playerCards)

	assert.Equal(t, 1, len(result), "Expected 1 result")
	assert.Equal(t, "player2", result[0].PlayerID)
	assert.Equal(t, Straight, result[0].HandRank) // This is a 6-5-4-3-2 straight
	assert.True(t, result[0].IsWinner)
}

func TestCompareHands_HighestPairWins(t *testing.T) {
	playerCards := map[string]cards.Stack{
		"player1": { // Pair of 7s
			{Suit: cards.Hearts, Value: cards.Seven},
			{Suit: cards.Hearts, Value: cards.Queen},
			{Suit: cards.Hearts, Value: cards.Seven},
			{Suit: cards.Hearts, Value: cards.Jack},
			{Suit: cards.Diamonds, Value: cards.Nine},
		},
		"player2": { // Pair of 8s
			{Suit: cards.Spades, Value: cards.Eight},
			{Suit: cards.Diamonds, Value: cards.Queen},
			{Suit: cards.Spades, Value: cards.Jack},
			{Suit: cards.Spades, Value: cards.Nine},
			{Suit: cards.Spades, Value: cards.Eight},
		},
	}

	result := CompareHands(playerCards)

	assert.Equal(t, 2, len(result), "Expected 2 results")
	assert.Equal(t, "player2", result[0].PlayerID)
	assert.Equal(t, OnePair, result[0].HandRank)
	assert.True(t, result[0].IsWinner)
	assert.Equal(t, "player1", result[1].PlayerID)
	assert.Equal(t, OnePair, result[1].HandRank)
	assert.False(t, result[1].IsWinner)
}

func TestCompareHands_HighestThreeOfAKindWins(t *testing.T) {
	playerCards := map[string]cards.Stack{
		"player1": {
			{Suit: cards.Hearts, Value: cards.Ten},
			{Suit: cards.Hearts, Value: cards.Ten},
			{Suit: cards.Hearts, Value: cards.Jack},
			{Suit: cards.Hearts, Value: cards.Ten},
			{Suit: cards.Diamonds, Value: cards.Nine},
		},
		"player2": {
			{Suit: cards.Spades, Value: cards.Jack},
			{Suit: cards.Spades, Value: cards.Ace},
			{Suit: cards.Diamonds, Value: cards.King},
			{Suit: cards.Spades, Value: cards.Jack},
			{Suit: cards.Spades, Value: cards.Jack},
		},
	}

	result := CompareHands(playerCards)

	assert.Equal(t, 2, len(result), "Expected 2 results")
	assert.Equal(t, "player2", result[0].PlayerID)
	assert.Equal(t, ThreeOfAKind, result[0].HandRank)
	assert.True(t, result[0].IsWinner)
	assert.Equal(t, "player1", result[1].PlayerID)
	assert.Equal(t, ThreeOfAKind, result[1].HandRank)
	assert.False(t, result[1].IsWinner)
}
