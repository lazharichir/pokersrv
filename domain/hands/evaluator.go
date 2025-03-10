package hands

import (
	"sort"

	"github.com/lazharichir/poker/domain/cards"
)

// HandRank represents the strength of a poker hand
type HandRank int

const (
	HighCard HandRank = iota
	OnePair
	TwoPair
	ThreeOfAKind
	Straight
	Flush
	FullHouse
	FourOfAKind
	StraightFlush
	RoyalFlush
)

// HandEvaluation represents the evaluation of a poker hand
type HandEvaluation struct {
	Rank      HandRank    // The hand rank (pair, flush, etc.)
	HandCards cards.Stack // The 5 cards that make up the hand
	Kickers   []int       // Kicker values for breaking ties, highest first
}

// valueToRank converts card values to numerical ranks (2=2, A=14)
func valueToRank(value cards.Value) int {
	valueMap := map[cards.Value]int{
		cards.Two:   2,
		cards.Three: 3,
		cards.Four:  4,
		cards.Five:  5,
		cards.Six:   6,
		cards.Seven: 7,
		cards.Eight: 8,
		cards.Nine:  9,
		cards.Ten:   10,
		cards.Jack:  11,
		cards.Queen: 12,
		cards.King:  13,
		cards.Ace:   14,
	}
	return valueMap[value]
}

// sortCardsByRank sorts cards by rank in descending order
func sortCardsByRank(hand cards.Stack) cards.Stack {
	result := make(cards.Stack, len(hand))
	copy(result, hand)

	sort.Slice(result, func(i, j int) bool {
		return valueToRank(result[i].Value) > valueToRank(result[j].Value)
	})

	return result
}

// evaluateHand evaluates a 5-card poker hand and returns its ranking
func evaluateHand(hand cards.Stack) HandEvaluation {
	if len(hand) != 5 {
		panic("Hand must contain exactly 5 cards")
	}

	// Sort cards by rank (highest first)
	sortedHand := sortCardsByRank(hand)

	// Check for royal flush
	if isRoyalFlush(sortedHand) {
		return HandEvaluation{
			Rank:      RoyalFlush,
			HandCards: sortedHand,
			Kickers:   []int{}, // Royal flush has no kickers for tie breaking
		}
	}

	// Check for straight flush
	if isStraightFlush(sortedHand) {
		// The highest card determines the straight flush strength
		highCard := valueToRank(sortedHand[0].Value)

		// Special case for A-5 straight flush (Ace counts as 1)
		if isA5Straight(sortedHand) {
			highCard = 5 // A-5 straight is ranked by the 5, not the A
		}

		return HandEvaluation{
			Rank:      StraightFlush,
			HandCards: sortedHand,
			Kickers:   []int{highCard},
		}
	}

	// Check for four of a kind
	if fourKind, kicker := isFourOfAKind(sortedHand); fourKind > 0 {
		return HandEvaluation{
			Rank:      FourOfAKind,
			HandCards: sortedHand,
			Kickers:   []int{fourKind, kicker},
		}
	}

	// Check for full house
	if three, pair := isFullHouse(sortedHand); three > 0 {
		return HandEvaluation{
			Rank:      FullHouse,
			HandCards: sortedHand,
			Kickers:   []int{three, pair},
		}
	}

	// Check for flush
	if isFlush(sortedHand) {
		// For a flush, the kickers are all cards in descending order
		kickers := make([]int, 5)
		for i, card := range sortedHand {
			kickers[i] = valueToRank(card.Value)
		}

		return HandEvaluation{
			Rank:      Flush,
			HandCards: sortedHand,
			Kickers:   kickers,
		}
	}

	// Check for straight
	if isStraight(sortedHand) {
		// The highest card determines the straight strength
		highCard := valueToRank(sortedHand[0].Value)

		// Special case for A-5 straight (Ace counts as 1)
		if isA5Straight(sortedHand) {
			highCard = 5 // A-5 straight is ranked by the 5, not the A
		}

		return HandEvaluation{
			Rank:      Straight,
			HandCards: sortedHand,
			Kickers:   []int{highCard},
		}
	}

	// Check for three of a kind
	if threeVal, kickers := isThreeOfAKind(sortedHand); threeVal > 0 {
		return HandEvaluation{
			Rank:      ThreeOfAKind,
			HandCards: sortedHand,
			Kickers:   append([]int{threeVal}, kickers...),
		}
	}

	// Check for two pair
	if pair1, pair2, kicker := isTwoPair(sortedHand); pair1 > 0 {
		return HandEvaluation{
			Rank:      TwoPair,
			HandCards: sortedHand,
			Kickers:   []int{pair1, pair2, kicker},
		}
	}

	// Check for one pair
	if pairVal, kickers := isOnePair(sortedHand); pairVal > 0 {
		return HandEvaluation{
			Rank:      OnePair,
			HandCards: sortedHand,
			Kickers:   append([]int{pairVal}, kickers...),
		}
	}

	// High card
	kickers := make([]int, 5)
	for i, card := range sortedHand {
		kickers[i] = valueToRank(card.Value)
	}

	return HandEvaluation{
		Rank:      HighCard,
		HandCards: sortedHand,
		Kickers:   kickers,
	}
}

// isRoyalFlush checks if a hand is a royal flush (A, K, Q, J, 10 of the same suit)
func isRoyalFlush(hand cards.Stack) bool {
	if !isFlush(hand) {
		return false
	}

	// Check for A, K, Q, J, 10
	values := map[cards.Value]bool{
		cards.Ace:   false,
		cards.King:  false,
		cards.Queen: false,
		cards.Jack:  false,
		cards.Ten:   false,
	}

	for _, card := range hand {
		values[card.Value] = true
	}

	// Make sure all required values are present
	for _, present := range values {
		if !present {
			return false
		}
	}

	return true
}

// isStraightFlush checks if a hand is a straight flush
func isStraightFlush(hand cards.Stack) bool {
	return isFlush(hand) && (isStraight(hand) || isA5Straight(hand))
}

// isFourOfAKind checks for four of a kind and returns the quad value and kicker
func isFourOfAKind(hand cards.Stack) (int, int) {
	// Count the occurrences of each value
	valueCounts := make(map[cards.Value]int)
	for _, card := range hand {
		valueCounts[card.Value]++
	}

	var fourKindValue cards.Value
	var kickerValue cards.Value

	for value, count := range valueCounts {
		if count == 4 {
			fourKindValue = value
		} else {
			kickerValue = value // There can only be one kicker in 5 cards
		}
	}

	if fourKindValue != "" {
		return valueToRank(fourKindValue), valueToRank(kickerValue)
	}

	return 0, 0
}

// isFullHouse checks for a full house and returns the trips value and pair value
func isFullHouse(hand cards.Stack) (int, int) {
	// Count the occurrences of each value
	valueCounts := make(map[cards.Value]int)
	for _, card := range hand {
		valueCounts[card.Value]++
	}

	var threeKindValue cards.Value
	var pairValue cards.Value

	for value, count := range valueCounts {
		if count == 3 {
			threeKindValue = value
		} else if count == 2 {
			pairValue = value
		}
	}

	if threeKindValue != "" && pairValue != "" {
		return valueToRank(threeKindValue), valueToRank(pairValue)
	}

	return 0, 0
}

// isFlush checks if all cards are of the same suit
func isFlush(hand cards.Stack) bool {
	if len(hand) == 0 {
		return false
	}

	suit := hand[0].Suit
	for _, card := range hand[1:] {
		if card.Suit != suit {
			return false
		}
	}

	return true
}

// isStraight checks if the hand is a straight (consecutive values)
func isStraight(hand cards.Stack) bool {
	// For regular straights, sort by rank
	cardCopy := make(cards.Stack, len(hand))
	copy(cardCopy, hand)

	// Sort by rank ascending
	sort.Slice(cardCopy, func(i, j int) bool {
		return valueToRank(cardCopy[i].Value) < valueToRank(cardCopy[j].Value)
	})

	// Check for consecutive values
	for i := 1; i < len(cardCopy); i++ {
		if valueToRank(cardCopy[i].Value) != valueToRank(cardCopy[i-1].Value)+1 {
			// Not consecutive
			return false
		}
	}

	return true
}

// isA5Straight checks for A-5-4-3-2 straight (where Ace is low)
func isA5Straight(hand cards.Stack) bool {
	// Look for A, 5, 4, 3, 2
	hasAce, has2, has3, has4, has5 := false, false, false, false, false

	for _, card := range hand {
		switch card.Value {
		case cards.Ace:
			hasAce = true
		case cards.Two:
			has2 = true
		case cards.Three:
			has3 = true
		case cards.Four:
			has4 = true
		case cards.Five:
			has5 = true
		}
	}

	return hasAce && has2 && has3 && has4 && has5
}

// isThreeOfAKind checks for three of a kind and returns the trips value and kickers
func isThreeOfAKind(hand cards.Stack) (int, []int) {
	// Count the occurrences of each value
	valueCounts := make(map[cards.Value]int)
	for _, card := range hand {
		valueCounts[card.Value]++
	}

	var threeKindValue cards.Value
	var kickers []cards.Value

	for value, count := range valueCounts {
		if count == 3 {
			threeKindValue = value
		} else {
			kickers = append(kickers, value)
		}
	}

	if threeKindValue == "" {
		return 0, nil
	}

	// Sort kickers by rank descending
	sort.Slice(kickers, func(i, j int) bool {
		return valueToRank(kickers[i]) > valueToRank(kickers[j])
	})

	// Convert kicker values to ints
	kickerRanks := make([]int, len(kickers))
	for i, value := range kickers {
		kickerRanks[i] = valueToRank(value)
	}

	return valueToRank(threeKindValue), kickerRanks
}

// isTwoPair checks for two pair and returns the pair values and kicker
func isTwoPair(hand cards.Stack) (int, int, int) {
	// Count the occurrences of each value
	valueCounts := make(map[cards.Value]int)
	for _, card := range hand {
		valueCounts[card.Value]++
	}

	var pairs []cards.Value
	var kicker cards.Value

	for value, count := range valueCounts {
		if count == 2 {
			pairs = append(pairs, value)
		} else if count == 1 {
			kicker = value
		}
	}

	if len(pairs) != 2 {
		return 0, 0, 0
	}

	// Sort pairs by rank descending
	sort.Slice(pairs, func(i, j int) bool {
		return valueToRank(pairs[i]) > valueToRank(pairs[j])
	})

	return valueToRank(pairs[0]), valueToRank(pairs[1]), valueToRank(kicker)
}

// isOnePair checks for one pair and returns the pair value and kickers
func isOnePair(hand cards.Stack) (int, []int) {
	// Count the occurrences of each value
	valueCounts := make(map[cards.Value]int)
	for _, card := range hand {
		valueCounts[card.Value]++
	}

	var pairValue cards.Value
	var kickers []cards.Value

	for value, count := range valueCounts {
		if count == 2 {
			pairValue = value
		} else {
			kickers = append(kickers, value)
		}
	}

	if pairValue == "" {
		return 0, nil
	}

	// Sort kickers by rank descending
	sort.Slice(kickers, func(i, j int) bool {
		return valueToRank(kickers[i]) > valueToRank(kickers[j])
	})

	// Convert kicker values to ints
	kickerRanks := make([]int, len(kickers))
	for i, value := range kickers {
		kickerRanks[i] = valueToRank(value)
	}

	return valueToRank(pairValue), kickerRanks
}

// compareHandsByRank compares two hands of the same rank to determine a winner
func compareHandsByRank(hand1, hand2 HandEvaluation) int {
	// Dispatch to appropriate comparison function based on hand rank
	switch hand1.Rank {
	case RoyalFlush:
		return 0 // Royal flushes are always equal
	case StraightFlush:
		return compareStraightFlushes(hand1, hand2)
	case FourOfAKind:
		return compareFourOfAKinds(hand1, hand2)
	case FullHouse:
		return compareFullHouses(hand1, hand2)
	case Flush:
		return compareFlushes(hand1, hand2)
	case Straight:
		return compareStraights(hand1, hand2)
	case ThreeOfAKind:
		return compareThreeOfAKinds(hand1, hand2)
	case TwoPair:
		return compareTwoPairs(hand1, hand2)
	case OnePair:
		return compareOnePairs(hand1, hand2)
	case HighCard:
		return compareHighCards(hand1, hand2)
	default:
		return 0
	}
}

// compareStraightFlushes compares two straight flush hands
func compareStraightFlushes(hand1, hand2 HandEvaluation) int {
	// For straight flushes, the highest card determines the winner
	return compareInt(hand1.Kickers[0], hand2.Kickers[0])
}

// compareFourOfAKinds compares two four-of-a-kind hands
func compareFourOfAKinds(hand1, hand2 HandEvaluation) int {
	// First compare the four-of-a-kind value
	if comp := compareInt(hand1.Kickers[0], hand2.Kickers[0]); comp != 0 {
		return comp
	}
	// If equal, compare the kicker
	return compareInt(hand1.Kickers[1], hand2.Kickers[1])
}

// compareFullHouses compares two full house hands
func compareFullHouses(hand1, hand2 HandEvaluation) int {
	// First compare the three-of-a-kind value
	if comp := compareInt(hand1.Kickers[0], hand2.Kickers[0]); comp != 0 {
		return comp
	}
	// If equal, compare the pair value
	return compareInt(hand1.Kickers[1], hand2.Kickers[1])
}

// compareFlushes compares two flush hands
func compareFlushes(hand1, hand2 HandEvaluation) int {
	// Compare each card in order from highest to lowest
	for i := 0; i < len(hand1.Kickers) && i < len(hand2.Kickers); i++ {
		if comp := compareInt(hand1.Kickers[i], hand2.Kickers[i]); comp != 0 {
			return comp
		}
	}
	return 0
}

// compareStraights compares two straight hands
func compareStraights(hand1, hand2 HandEvaluation) int {
	// For straights, the highest card determines the winner
	return compareInt(hand1.Kickers[0], hand2.Kickers[0])
}

// compareThreeOfAKinds compares two three-of-a-kind hands
func compareThreeOfAKinds(hand1, hand2 HandEvaluation) int {
	// First compare the three-of-a-kind value
	if comp := compareInt(hand1.Kickers[0], hand2.Kickers[0]); comp != 0 {
		return comp
	}
	// Then compare the kickers in order
	for i := 1; i < len(hand1.Kickers) && i < len(hand2.Kickers); i++ {
		if comp := compareInt(hand1.Kickers[i], hand2.Kickers[i]); comp != 0 {
			return comp
		}
	}
	return 0
}

// compareTwoPairs compares two two-pair hands
func compareTwoPairs(hand1, hand2 HandEvaluation) int {
	// First compare the higher pair
	if comp := compareInt(hand1.Kickers[0], hand2.Kickers[0]); comp != 0 {
		return comp
	}
	// Then compare the lower pair
	if comp := compareInt(hand1.Kickers[1], hand2.Kickers[1]); comp != 0 {
		return comp
	}
	// Finally compare the kicker
	return compareInt(hand1.Kickers[2], hand2.Kickers[2])
}

// compareOnePairs compares two one-pair hands
func compareOnePairs(hand1, hand2 HandEvaluation) int {
	// First compare the pair value
	if comp := compareInt(hand1.Kickers[0], hand2.Kickers[0]); comp != 0 {
		return comp
	}
	// Then compare the kickers in order
	for i := 1; i < len(hand1.Kickers) && i < len(hand2.Kickers); i++ {
		if comp := compareInt(hand1.Kickers[i], hand2.Kickers[i]); comp != 0 {
			return comp
		}
	}
	return 0
}

// compareHighCards compares two high card hands
func compareHighCards(hand1, hand2 HandEvaluation) int {
	// Compare each card in order from highest to lowest
	for i := 0; i < len(hand1.Kickers) && i < len(hand2.Kickers); i++ {
		if comp := compareInt(hand1.Kickers[i], hand2.Kickers[i]); comp != 0 {
			return comp
		}
	}
	return 0
}

// compareInt is a helper function to compare two integers
func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// compareHandEvaluations compares two hand evaluations and returns:
// -1 if hand1 is worse than hand2
// 0 if hands are equal
// 1 if hand1 is better than hand2
func compareHandEvaluations(hand1, hand2 HandEvaluation) int {
	// First compare by rank
	if hand1.Rank < hand2.Rank {
		return -1
	}
	if hand1.Rank > hand2.Rank {
		return 1
	}

	// Same rank, use specialized comparison function for the specific hand type
	return compareHandsByRank(hand1, hand2)
}

// combinations generates all possible combinations of k elements from a set
func combinations(n, k int) [][]int {
	if k > n {
		return nil
	}

	var result [][]int
	var combine func(int, []int)

	combine = func(start int, current []int) {
		if len(current) == k {
			// Make a copy of current combination
			combo := make([]int, k)
			copy(combo, current)
			result = append(result, combo)
			return
		}

		for i := start; i < n; i++ {
			current = append(current, i)
			combine(i+1, current)
			current = current[:len(current)-1]
		}
	}

	combine(0, []int{})
	return result
}

// BestHandEvaluation represents the best possible 5-card hand and its evaluation
type BestHandEvaluation struct {
	Evaluation HandEvaluation
	Cards      cards.Stack // The specific 5 cards used
}

// listAllPossibleHands generates all possible 5-card hands from a given set of cards
// and returns them sorted by hand strength (best first)
func ListAllPossibleHands(cardSet cards.Stack) []BestHandEvaluation {
	n := len(cardSet)
	if n < 5 {
		return nil
	}

	// Generate all combinations of 5 cards from the set
	combos := combinations(n, 5)
	result := make([]BestHandEvaluation, 0, len(combos))

	for _, combo := range combos {
		// Build a 5-card hand from the combination
		hand := make(cards.Stack, 5)
		for i, idx := range combo {
			hand[i] = cardSet[idx]
		}

		// Evaluate the hand
		evaluation := evaluateHand(hand)

		result = append(result, BestHandEvaluation{
			Evaluation: evaluation,
			Cards:      hand,
		})
	}

	// Sort hands by strength (best first)
	sort.Slice(result, func(i, j int) bool {
		return compareHandEvaluations(result[i].Evaluation, result[j].Evaluation) > 0
	})

	return result
}

// HandComparisonResult represents the result of comparing multiple hands
type HandComparisonResult struct {
	PlayerID   string
	HandRank   HandRank
	HandCards  cards.Stack
	IsWinner   bool
	PlaceIndex int // 0 for first place, 1 for second place, etc.
}

// compareHands compares multiple player hands and determines winners
// playerCards is a map of player ID to their available cards
// Returns the comparison results sorted by hand strength (best first)
func CompareHands(playerCards map[string]cards.Stack) []HandComparisonResult {
	if len(playerCards) == 0 {
		return nil
	}

	type playerHandEval struct {
		playerID string
		bestHand BestHandEvaluation
	}

	// Calculate best hand for each player
	playerHands := make([]playerHandEval, 0, len(playerCards))
	for playerID, cards := range playerCards {
		possibleHands := ListAllPossibleHands(cards)
		if len(possibleHands) > 0 {
			playerHands = append(playerHands, playerHandEval{
				playerID: playerID,
				bestHand: possibleHands[0], // First hand is the best one due to sorting
			})
		}
	}

	// Sort players by hand strength
	sort.Slice(playerHands, func(i, j int) bool {
		return compareHandEvaluations(
			playerHands[i].bestHand.Evaluation,
			playerHands[j].bestHand.Evaluation,
		) > 0
	})

	// Create results with place indices
	results := make([]HandComparisonResult, len(playerHands))

	if len(playerHands) > 0 {
		// First place is always index 0
		placeIndex := 0
		results[0] = HandComparisonResult{
			PlayerID:   playerHands[0].playerID,
			HandRank:   playerHands[0].bestHand.Evaluation.Rank,
			HandCards:  playerHands[0].bestHand.Cards,
			IsWinner:   true, // Only the best hand is a winner according to standard poker rules
			PlaceIndex: placeIndex,
		}

		// Process remaining players
		for i := 1; i < len(playerHands); i++ {
			// Check if this player ties with previous player
			if compareHandEvaluations(
				playerHands[i].bestHand.Evaluation,
				playerHands[i-1].bestHand.Evaluation,
			) == 0 {
				// Tie with previous player, same place index and also a winner
				results[i] = HandComparisonResult{
					PlayerID:   playerHands[i].playerID,
					HandRank:   playerHands[i].bestHand.Evaluation.Rank,
					HandCards:  playerHands[i].bestHand.Cards,
					IsWinner:   true, // Players who tie for best hand are also winners
					PlaceIndex: placeIndex,
				}
			} else {
				// Lower hand strength, increment place index
				placeIndex = i
				results[i] = HandComparisonResult{
					PlayerID:   playerHands[i].playerID,
					HandRank:   playerHands[i].bestHand.Evaluation.Rank,
					HandCards:  playerHands[i].bestHand.Cards,
					IsWinner:   false, // Only the best hand(s) can be winners
					PlaceIndex: placeIndex,
				}
			}
		}
	}

	return results
}
