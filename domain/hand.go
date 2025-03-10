package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/lazharichir/poker/cards"
	"github.com/lazharichir/poker/domain/events"
	"github.com/lazharichir/poker/hands"
)

type HandPhase string

const (
	HandPhase_Start              HandPhase = "start"
	HandPhase_Antes              HandPhase = "antes"
	HandPhase_Hole               HandPhase = "hole"
	HandPhase_Continuation       HandPhase = "continuation"
	HandPhase_CommunityDeal      HandPhase = "community.deal"
	HandPhase_CommunitySelection HandPhase = "community.selection"
	HandPhase_HandReveal         HandPhase = "hand.reveal"
	HandPhase_Decision           HandPhase = "decision"
	HandPhase_Payout             HandPhase = "payout"
	HandPhase_Ended              HandPhase = "ended"
)

// Hand represents a hand of poker being played
type Hand struct {
	Table          *Table
	ID             string
	Phase          HandPhase
	TableID        string
	Players        []Player
	Deck           cards.Stack
	CommunityCards cards.Stack
	HoleCards      map[string]cards.Stack
	Pot            int
	Events         []events.Event
	TableRules     TableRules
	StartedAt      time.Time

	Results []hands.HandComparisonResult

	// New fields for tracking bets
	AntesPaid           map[string]int  // Maps player IDs to ante amounts
	ContinuationBets    map[string]int  // Maps player IDs to continuation bet amounts
	ActivePlayers       map[string]bool // Maps player IDs to active status (still in the hand)
	CurrentBettor       string          // ID of player who should act next
	ButtonPosition      int             // Index of button player in the Players slice
	CommunitySelections map[string]cards.Stack

	CommunitySelectionStartedAt time.Time

	// events
	eventHandlers []events.EventHandler
}

// RegisterEventHandler registers a callback function that will be called when events occur
func (h *Hand) RegisterEventHandler(handler events.EventHandler) {
	h.eventHandlers = append(h.eventHandlers, handler)
}

// emitEvent notifies all registered handlers of a new event
func (h *Hand) emitEvent(event events.Event) {
	// Add event to hand's event log
	h.Events = append(h.Events, event)

	// Notify all handlers
	for _, handler := range h.eventHandlers {
		handler(event)
	}
}

// InitializeHand initializes a new hand with a fresh deck and activates all players
func (h *Hand) InitializeHand() {
	// Initialize a new shuffled deck
	h.Deck = cards.NewDeck52()
	h.Deck.Shuffle()

	// Initialize the community cards as empty
	h.CommunityCards = []cards.Card{}

	// Initialize hole cards map for each player
	h.HoleCards = make(map[string]cards.Stack)

	// Set all players to active at the start of the hand
	h.ActivePlayers = make(map[string]bool)
	for _, player := range h.Players {
		h.setPlayerAsActive(player.ID)
		h.HoleCards[player.ID] = []cards.Card{}
	}

	// Initialize betting maps
	h.Results = []hands.HandComparisonResult{}
	h.AntesPaid = make(map[string]int)
	h.ContinuationBets = make(map[string]int)
	h.CommunitySelections = make(map[string]cards.Stack)

	// Set the current bettor to the player left of the button
	h.CurrentBettor = h.getPlayerLeftOfButton()

	// Emit HandStarted event
	playerIDs := make([]string, len(h.Players))
	for i, player := range h.Players {
		playerIDs[i] = player.ID
	}

	h.emitEvent(events.HandStarted{
		TableID: h.TableID,
		HandID:  h.ID,
		Players: playerIDs,
		At:      time.Now(),
	})
}

func (h *Hand) TransitionToAntesPhase() {
	if !h.IsInPhase(HandPhase_Start) {
		return
	}

	previousPhase := h.Phase
	h.Phase = HandPhase_Antes

	// Emit phase changed event
	h.emitEvent(events.PhaseChanged{
		TableID:       h.TableID,
		HandID:        h.ID,
		PreviousPhase: string(previousPhase),
		NewPhase:      string(h.Phase),
		At:            time.Now(),
	})

	// The actual ante collection would happen in the game loop,
	// giving each player the specified timeout to respond.
	// Starting from the player left of the dealer (would need dealer position tracking)
	// If a player doesn't respond within the timeout, they would be folded automatically
}

// PlayerPlacesAnte records a player placing an ante
func (h *Hand) PlayerPlacesAnte(playerID string, amount int) error {
	// Check if in the correct phase
	if !h.IsInPhase(HandPhase_Antes) {
		return errors.New("not in antes phase")
	}

	// Check if it's the player's turn to act
	if !h.IsPlayerTheCurrentBettor(playerID) {
		return errors.New("not this player's turn to act")
	}

	// Check if player already paid ante
	if h.hasAlreadyPlacedAnte(playerID) {
		return errors.New("player already paid ante")
	}

	// Record the ante
	h.Table.DecreasePlayerBuyIn(playerID, amount)
	h.addToPlayerAntesPaid(playerID, amount)
	h.increasePot(amount)

	// Emit AntePlaced event
	h.emitEvent(events.AntePlaced{
		TableID:  h.TableID,
		HandID:   h.ID,
		PlayerID: playerID,
		Amount:   amount,
		At:       time.Now(),
	})

	// Find next player to act
	h.CurrentBettor = h.getNextActiveBettor(playerID)

	// Check if all antes have been paid
	if h.areAllAntesPaid() {
		h.TransitionToHolePhase()
	}

	return nil
}

// HandleAntePhaseTimeout handles the case where the ante phase timer expires
func (h *Hand) HandleAntePhaseTimeout() error {
	if !h.IsInPhase(HandPhase_Antes) {
		return errors.New("not in ante phase")
	}

	// Fold all players who haven't placed ante
	for _, player := range h.Players {
		if h.IsPlayerActive(player.ID) && !h.hasAlreadyPlacedAnte(player.ID) {
			h.setPlayerAsInactive(player.ID)

			// Emit PlayerTimedOut event
			h.emitEvent(events.PlayerTimedOut{
				TableID:       h.TableID,
				HandID:        h.ID,
				PlayerID:      player.ID,
				Phase:         string(h.Phase),
				DefaultAction: "fold", // Assuming default action is fold
				At:            time.Now(),
			})
		}
	}

	// If we have at least one active player, proceed
	if h.countActivePlayers() > 0 {
		h.TransitionToHolePhase()
		return nil
	}

	// No active players, end the hand
	h.TransitionToEndedPhase()
	return nil
}

func (h *Hand) TransitionToHolePhase() {
	if !h.IsInPhase(HandPhase_Antes) {
		return
	}

	previousPhase := h.Phase
	h.Phase = HandPhase_Hole

	// Emit phase changed event
	h.emitEvent(events.PhaseChanged{
		TableID:       h.TableID,
		HandID:        h.ID,
		PreviousPhase: string(previousPhase),
		NewPhase:      string(h.Phase),
		At:            time.Now(),
	})

	// Reset CurrentBettor for next phase
	h.CurrentBettor = h.getPlayerLeftOfButton()
}

// DealHoleCards deals two cards to each active player, one card at a time
func (h *Hand) DealHoleCards() error {
	if !h.IsInPhase(HandPhase_Hole) {
		return errors.New("not in hole card phase")
	}

	// Create a map to track the dealing order
	dealOrder := make(map[string]int)
	dealPosition := 0

	dealRound := func() error {
		for i := 0; i < len(h.Players); i++ {
			// Start with player to left of button and go around
			player := h.getPlayerByIndex((h.ButtonPosition + 1 + i) % len(h.Players))

			// Only deal to active players
			if h.IsPlayerActive(player.ID) {
				if len(h.Deck) == 0 {
					return errors.New("no cards left in deck")
				}

				// Deal one card
				card := h.Deck.DealCard()
				h.HoleCards[player.ID] = append(h.HoleCards[player.ID], card)

				// Record deal position for this player (first time only)
				if _, exists := dealOrder[player.ID]; !exists {
					dealOrder[player.ID] = dealPosition
					dealPosition++
				}

				// Emit HoleCardDealt event
				h.emitEvent(events.HoleCardDealt{
					TableID:  h.TableID,
					HandID:   h.ID,
					PlayerID: player.ID,
					Card:     card,
					At:       time.Now(),
				})
			}
		}
		return nil
	}

	// First round of dealing (first card to each player)
	if err := dealRound(); err != nil {
		return err
	}

	// Second round of dealing (second card to each player)
	if err := dealRound(); err != nil {
		return err
	}

	// Emit HoleCardsDealt event with the dealing order
	h.emitEvent(events.HoleCardsDealt{
		TableID:   h.TableID,
		HandID:    h.ID,
		DealOrder: dealOrder,
		At:        time.Now(),
	})

	// Transition to continuation phase
	h.TransitionToContinuationPhase()

	return nil
}

func (h *Hand) TransitionToContinuationPhase() {
	if !h.IsInPhase(HandPhase_Hole) {
		return
	}

	previousPhase := h.Phase
	h.Phase = HandPhase_Continuation

	// Emit phase changed event
	h.emitEvent(events.PhaseChanged{
		TableID:       h.TableID,
		HandID:        h.ID,
		PreviousPhase: string(previousPhase),
		NewPhase:      string(h.Phase),
		At:            time.Now(),
	})

	// Reset CurrentBettor for next phase
	h.CurrentBettor = h.getPlayerLeftOfButton()

	// The actual continuation betting would happen in the game loop,
	// giving each player the specified timeout to respond.
	// Starting from the player left of the dealer (would need dealer position tracking)
	// If a player doesn't respond within the timeout, they would be folded automatically
}

// PlayerPlacesContinuationBet records a player placing a continuation bet
func (h *Hand) PlayerPlacesContinuationBet(playerID string, amount int) error {
	// Check if in the correct phase
	if !h.IsInPhase(HandPhase_Continuation) {
		return errors.New("not in continuation bet phase")
	}

	// Check if it's the player's turn to act
	if !h.IsPlayerTheCurrentBettor(playerID) {
		return errors.New("not this player's turn to act")
	}

	// Check if player already made decision
	if h.hasAlreadyPlacedContinuationBet(playerID) {
		return errors.New("player already made continuation bet decision")
	}

	// Record the bet
	h.Table.DecreasePlayerBuyIn(playerID, amount)
	h.increasePot(amount)

	// Emit ContinuationBetPlaced event
	h.emitEvent(events.ContinuationBetPlaced{
		TableID:  h.TableID,
		HandID:   h.ID,
		PlayerID: playerID,
		Amount:   amount,
		At:       time.Now(),
	})

	// Find next player to act
	h.CurrentBettor = h.getNextActiveBettor(playerID)

	// Check if all continuation bets are in
	if h.haveAllPlayersDecided() {
		h.TransitionToCommunityDealPhase()
	}

	return nil
}

// PlayerFolds handles a player folding
func (h *Hand) PlayerFolds(playerID string) error {
	// Check if player is active
	if !h.IsPlayerActive(playerID) {
		return errors.New("player is not active in this hand")
	}

	// Check if it's appropriate phase for folding (continuation or discard)
	if !h.IsInPhase(HandPhase_Continuation) {
		return errors.New("cannot fold in current phase")
	}

	// Check if it's not the player's turn to act
	if !h.IsPlayerTheCurrentBettor(playerID) {
		return errors.New("not this player's turn to act")
	}

	// Mark player as inactive
	h.setPlayerAsInactive(playerID)

	// Emit PlayerFolded event
	h.emitEvent(events.PlayerFolded{
		TableID:  h.TableID,
		HandID:   h.ID,
		PlayerID: playerID,
		Phase:    string(h.Phase),
		At:       time.Now(),
	})

	// Check if only one player remains
	if h.countActivePlayers() == 1 {
		// Hand is over, last active player wins
		lastActivePlayer, err := h.getLastActivePlayer()
		if err != nil {
			return err
		}
		h.handleSinglePlayerWin(lastActivePlayer.ID)
	}

	// Find next player to act
	h.CurrentBettor = h.getNextActiveBettor(playerID)

	// Check if all continuation bets are in
	if h.haveAllPlayersDecided() {
		h.TransitionToCommunityDealPhase()
	}

	return nil
}

func (h *Hand) hasAlreadyPlacedContinuationBet(playerID string) bool {
	_, decided := h.ContinuationBets[playerID]
	return decided
}

func (h *Hand) haveAllPlayersDecided() bool {
	for playerID := range h.ActivePlayers {
		if _, decided := h.ContinuationBets[playerID]; !decided {
			return false
		}
	}
	return true
}

func (h *Hand) TransitionToCommunityDealPhase() {
	if !h.IsInPhase(HandPhase_Continuation) {
		return
	}

	previousPhase := h.Phase
	h.Phase = HandPhase_CommunityDeal

	// Emit phase changed event
	h.emitEvent(events.PhaseChanged{
		TableID:       h.TableID,
		HandID:        h.ID,
		PreviousPhase: string(previousPhase),
		NewPhase:      string(h.Phase),
		At:            time.Now(),
	})

	// Reset CurrentBettor for next phase
	h.CurrentBettor = h.getPlayerLeftOfButton()

	// The actual community card dealing would happen in the game loop,
	// giving each player the specified timeout to respond.
	// Starting from the player left of the dealer (would need dealer position tracking)
	// If a player doesn't respond within the timeout, they would be folded automatically

	h.StartDealingCommunityCards()
}

func (h *Hand) StartDealingCommunityCards() error {
	// burn one card
	if err := h.BurnCard(); err != nil {
		return err
	}

	// deal 8 cards
	for i := 0; i < 8; i++ {
		if err := h.DealCommunityCard(); err != nil {
			return err
		}
	}

	return nil
}

// DealCommunityCard deals a single community card
func (h *Hand) DealCommunityCard() error {
	if !h.IsInPhase(HandPhase_CommunityDeal) {
		return errors.New("not in community card dealing phase")
	}

	if h.Deck.IsEmpty() {
		return errors.New("no cards left in deck")
	}

	// Deal one card
	card := h.Deck.DealCard()
	h.CommunityCards = append(h.CommunityCards, card)

	// Emit CommunityCardDealt event
	h.emitEvent(events.CommunityCardDealt{
		TableID:   h.TableID,
		HandID:    h.ID,
		CardIndex: len(h.CommunityCards) - 1, // Index of the card just dealt (0-based)
		Card:      card,
		At:        time.Now(),
	})

	// Transition to decision phase if all community cards have been dealt
	if len(h.CommunityCards) == 8 {
		h.TransitionToCommunitySelectionPhase()
	}
	return nil
}

func (h *Hand) TransitionToCommunitySelectionPhase() {
	if !h.IsInPhase(HandPhase_CommunityDeal) {
		return
	}

	previousPhase := h.Phase
	h.Phase = HandPhase_CommunitySelection
	h.CommunitySelectionStartedAt = time.Now()

	// Emit phase changed event
	h.emitEvent(events.PhaseChanged{
		TableID:       h.TableID,
		HandID:        h.ID,
		PreviousPhase: string(previousPhase),
		NewPhase:      string(h.Phase),
		At:            time.Now(),
	})

	// in this phase, players have 5 seconds to select three
	// community cards to form the best hand
	// once a card is selected, they cannot change it
}

func (h *Hand) PlayerSelectsCommunityCard(playerID string, selectedCard cards.Card) error {
	// Check if in the correct phase
	if !h.IsInPhase(HandPhase_CommunitySelection) {
		return errors.New("not in community card selection phase")
	}

	// Check if the player is active in this hand
	if !h.IsPlayerActive(playerID) {
		return errors.New("player is not active")
	}

	// Check if card is in community cards
	if !h.checkIfValidCommunityCard(selectedCard) {
		return errors.New("selected card is not a community card")
	}

	if h.CommunitySelections[playerID] == nil {
		h.CommunitySelections[playerID] = []cards.Card{}
	}

	// Check if player has already selected 3 cards
	if len(h.CommunitySelections[playerID]) >= 3 {
		return errors.New("player has already selected 3 cards")
	}

	// Check if player already selected this card (cannot select same card twice)
	for _, card := range h.CommunitySelections[playerID] {
		if card.Equals(selectedCard) {
			return errors.New("player already selected this card")
		}
	}

	// Check it's within the 5s selection window
	if time.Since(h.CommunitySelectionStartedAt) > 5*time.Second {
		return errors.New("selection window has closed")
	}

	// Add card to player's selections
	h.CommunitySelections[playerID] = append(h.CommunitySelections[playerID], selectedCard)

	// Emit CommunityCardSelected event
	h.emitEvent(events.CommunityCardSelected{
		TableID:        h.TableID,
		HandID:         h.ID,
		PlayerID:       playerID,
		Card:           selectedCard.String(),                // Assuming Card has a String() method
		SelectionOrder: len(h.CommunitySelections[playerID]), // Order in which card was selected
		At:             time.Now(),
	})

	// Transition to the decision phase if all players have selected their cards
	if h.haveAllActivePlayersSelectedTheirCommunityCards() {
		h.TransitionToDecisionPhase()
	}

	return nil
}

func (h *Hand) checkIfValidCommunityCard(card cards.Card) bool {
	for _, c := range h.CommunityCards {
		if c == card {
			return true
		}
	}
	return false
}

func (h *Hand) haveAllActivePlayersSelectedTheirCommunityCards() bool {
	if len(h.CommunitySelections) != len(h.ActivePlayers) {
		return false
	}

	// they all must have selected 3 cards
	for playerID := range h.ActivePlayers {
		if len(h.CommunitySelections[playerID]) != 3 {
			return false
		}
	}

	return true
}

func (h *Hand) TransitionToDecisionPhase() {
	if !h.IsInPhase(HandPhase_HandReveal) {
		return
	}

	previousPhase := h.Phase
	h.Phase = HandPhase_Decision

	// Emit phase changed event
	h.emitEvent(events.PhaseChanged{
		TableID:       h.TableID,
		HandID:        h.ID,
		PreviousPhase: string(previousPhase),
		NewPhase:      string(h.Phase),
		At:            time.Now(),
	})

	// Evaluate hands and determine the winner(s)
	h.EvaluateHands()

	// Transition to payout phase
	if len(h.Results) > 0 {
		h.TransitionToPayoutPhase()
		return
	}

	fmt.Println("No results found, ending hand")
	h.TransitionToEndedPhase()
}

// EvaluateHands evaluates all active players' hands and determines the winner(s)
func (h *Hand) EvaluateHands() ([]hands.HandComparisonResult, error) {
	// Create a map of player ID to their combined hole and community cards
	playerCards := make(map[string]cards.Stack)
	for playerID, holeCards := range h.HoleCards {
		if !h.IsPlayerActive(playerID) {
			continue
		}

		// Combine hole cards and community cards
		combinedCards := append(cards.Stack{}, holeCards...)
		combinedCards = append(combinedCards, h.CommunityCards...)
		playerCards[playerID] = combinedCards
	}

	// Use the hand evaluator to determine the best hand for each player
	// (This assumes we have access to the hands package)
	results := h.comparePlayerHands(playerCards)

	// Record event with the results
	resultData := make([]map[string]interface{}, len(results))
	for i, result := range results {
		resultData[i] = map[string]interface{}{
			"player_id":   result.PlayerID,
			"hand_rank":   result.HandRank,
			"is_winner":   result.IsWinner,
			"place_index": result.PlaceIndex,
		}
	}

	h.Results = results

	return results, nil
}

func (h *Hand) comparePlayerHands(playerCards map[string]cards.Stack) []hands.HandComparisonResult {
	return hands.CompareHands(playerCards)
}

func (h *Hand) TransitionToPayoutPhase() {
	if !h.IsInPhase(HandPhase_Decision) {
		return
	}

	previousPhase := h.Phase
	h.Phase = HandPhase_Payout

	// Emit phase changed event
	h.emitEvent(events.PhaseChanged{
		TableID:       h.TableID,
		HandID:        h.ID,
		PreviousPhase: string(previousPhase),
		NewPhase:      string(h.Phase),
		At:            time.Now(),
	})

	// Payout the pot to the winner(s)
	h.Payout()
}

// Payout distributes the pot to the winner(s)
func (h *Hand) Payout() error {
	// Check if in the correct phase
	if !h.IsInPhase(HandPhase_Payout) {
		return errors.New("not in payout phase")
	}

	// Find winners
	var winners []string
	for _, result := range h.Results {
		if result.IsWinner {
			winners = append(winners, result.PlayerID)
		}
	}

	// If no winners found (shouldn't happen), return error
	if len(winners) == 0 {
		return errors.New("no winners found")
	}

	// If one winner found
	if len(winners) == 0 {
		return h.awardPayout(winners[0], h.Pot, "winner takes all")
	}

	// If more than one winner, calculate the amount each winner gets (split pot)
	winAmount := h.Pot / len(winners)
	remainder := h.Pot % len(winners)

	// Prepare breakdown for event
	breakdown := make(map[string]int)
	for _, winnerID := range winners {
		breakdown[winnerID] = winAmount
	}

	// Distribute the pot
	for _, winnerID := range winners {
		// Find player index
		if err := h.awardPayout(winnerID, winAmount, "pot split"); err != nil {
			return err
		}
	}

	// If there's a remainder due to uneven split, give it to first winner
	// (usually the player closest to the left of the dealer)
	if remainder > 0 && len(winners) > 0 {
		if err := h.awardPayout(winners[0], remainder, "remainder payout after pot split"); err != nil {
			return err
		}
		breakdown[winners[0]] += remainder
	}

	// Emit PotBrokenDown event
	h.emitEvent(events.PotBrokenDown{
		TableID:   h.TableID,
		HandID:    h.ID,
		Breakdown: breakdown,
		At:        time.Now(),
	})

	// Empty the pot
	h.Pot = 0

	// Transition to ended state
	h.TransitionToEndedPhase()

	return nil
}

func (h *Hand) awardPayout(winnerID string, amount int, reason string) error {
	h.Table.IncreasePlayerBuyIn(winnerID, amount)

	// Emit PotAmountAwarded event
	h.emitEvent(events.PotAmountAwarded{
		TableID:  h.TableID,
		HandID:   h.ID,
		PlayerID: winnerID,
		Amount:   amount,
		Reason:   reason,
		At:       time.Now(),
	})

	return nil
}

// payoutToLastPlayerStanding distributes the pot to the last player standing
func (h *Hand) payoutToLastPlayerStanding(winnerID string) error {
	if err := h.awardPayout(winnerID, h.Pot, "last player standing"); err != nil {
		return err
	}

	// Empty the pot
	h.Pot = 0

	// Transition to ended state
	h.TransitionToEndedPhase()

	return nil
}

func (h *Hand) TransitionToEndedPhase() {
	if !h.IsInPhase(HandPhase_Payout) {
		return
	}

	previousPhase := h.Phase
	h.Phase = HandPhase_Ended

	// Emit phase changed event
	h.emitEvent(events.PhaseChanged{
		TableID:       h.TableID,
		HandID:        h.ID,
		PreviousPhase: string(previousPhase),
		NewPhase:      string(h.Phase),
		At:            time.Now(),
	})

	// Find winners
	var winners []string
	for _, result := range h.Results {
		if result.IsWinner {
			winners = append(winners, result.PlayerID)
		}
	}

	// Calculate hand duration
	duration := time.Since(h.StartedAt).Milliseconds()

	// Emit HandEnded event
	h.emitEvent(events.HandEnded{
		TableID:  h.TableID,
		HandID:   h.ID,
		Duration: duration,
		FinalPot: h.Pot,
		Winners:  winners,
		At:       time.Now(),
	})
}

func (h *Hand) IsPlayerActive(playerID string) bool {
	return h.ActivePlayers[playerID]
}

func (h *Hand) setPlayerAsActive(playerID string) {
	h.ActivePlayers[playerID] = true
}

func (h *Hand) setPlayerAsInactive(playerID string) {
	h.ActivePlayers[playerID] = false
}

// handleSinglePlayerWin handles case where only one player remains
func (h *Hand) handleSinglePlayerWin(playerID string) {
	// Skip to the payout phase directly
	h.Phase = HandPhase_Payout

	// Emit SingleWinnerDetermined event
	h.emitEvent(events.SingleWinnerDetermined{
		TableID:  h.TableID,
		HandID:   h.ID,
		PlayerID: playerID,
		Reason:   "last player standing",
		At:       time.Now(),
	})

	// Handle payout logic
	h.Payout()

	// End the hand
	h.TransitionToEndedPhase()
}

// getPlayerLeftOfButton returns the player ID to the left of the button
func (h *Hand) getPlayerLeftOfButton() string {
	if len(h.Players) == 0 {
		return ""
	}

	pos := (h.ButtonPosition + 1) % len(h.Players)
	return h.Players[pos].ID
}

// IsWaitingForBet checks if the hand is waiting for a player to bet
func (h *Hand) IsWaitingForBet() bool {
	// Check the current phase
	switch h.Phase {
	case HandPhase_Antes:
		return len(h.AntesPaid) < len(h.ActivePlayers)
	case HandPhase_Continuation:
		for playerID, active := range h.ActivePlayers {
			if active {
				if _, decided := h.ContinuationBets[playerID]; !decided {
					return true
				}
			}
		}
		return false
	default:
		return false
	}
}

// BurnCard removes the top card from the deck without revealing it
func (h *Hand) BurnCard() error {
	if len(h.Deck) == 0 {
		return errors.New("no cards left in deck to burn")
	}

	// Remove top card without using it
	h.Deck.BurnCard()

	// Emit CardBurned event
	h.emitEvent(events.CardBurned{
		TableID: h.TableID,
		HandID:  h.ID,
		At:      time.Now(),
	})

	return nil
}

func (h *Hand) IsPlayerTheCurrentBettor(playerID string) bool {
	return h.CurrentBettor == playerID
}

func (h *Hand) IsInPhase(phase HandPhase) bool {
	return h.Phase == phase
}

func (h *Hand) HasEnded() bool {
	return h.IsInPhase(HandPhase_Ended)
}

// PrintState is a debugging function to print the current state of the hand in a string, over multiple lines and in a human-readable structured format
func (h *Hand) PrintState() string {
	output := "Hand State:\n"
	output += "--------------------------------------------------\n"
	output += "ID: " + h.ID + "\n"
	output += "Phase: " + string(h.Phase) + "\n"
	output += "TableID: " + h.TableID + "\n"
	output += "Pot: " + fmt.Sprint(h.Pot) + "\n"
	output += "CurrentBettor: " + h.CurrentBettor + "\n"
	output += "ButtonPosition: " + fmt.Sprint(h.ButtonPosition) + "\n"
	output += "\n"

	output += "Players:\n"
	for _, player := range h.Players {
		output += "  - ID: " + player.ID + ", Name: " + player.Name + ", Chips: " + fmt.Sprint(h.Table.GetlayerBuyIn(player.ID)) + "\n"
	}
	output += "\n"

	output += "Active Players:\n"
	for playerID, active := range h.ActivePlayers {
		output += "  - ID: " + playerID + ", Active: " + fmt.Sprint(active) + "\n"
	}
	output += "\n"

	output += "Hole Cards:\n"
	for playerID, cards := range h.HoleCards {
		output += "  - Player: " + playerID + ", Cards: " + cards.String() + "\n"
	}
	output += "\n"

	output += "Community Cards: " + h.CommunityCards.String() + "\n"
	output += "\n"

	output += "Antes Paid:\n"
	for playerID, amount := range h.AntesPaid {
		output += "  - Player: " + playerID + ", Amount: " + fmt.Sprint(amount) + "\n"
	}
	output += "\n"

	output += "Continuation Bets:\n"
	for playerID, amount := range h.ContinuationBets {
		output += "  - Player: " + playerID + ", Amount: " + fmt.Sprint(amount) + "\n"
	}
	output += "\n"

	output += "Events:\n"
	for _, event := range h.Events {
		output += "  - Type: " + event.Name() + "\n"
	}
	output += "\n"

	output += "--------------------------------------------------\n"

	return output
}

func (h *Hand) countActivePlayers() int {
	count := 0
	for _, active := range h.ActivePlayers {
		if active {
			count++
		}
	}
	return count
}

func (h *Hand) getLastActivePlayer() (Player, error) {
	if h.countActivePlayers() == 0 {
		return Player{}, errors.New("no active players found")
	}
	if h.countActivePlayers() > 1 {
		return Player{}, errors.New("more than one active player found")
	}

	for _, player := range h.Players {
		if h.ActivePlayers[player.ID] {
			return player, nil
		}
	}

	return Player{}, errors.New("no active players found")
}

func (h *Hand) increasePot(amount int) {
	previousAmount := h.Pot
	h.Pot += amount

	// Emit PotChanged event
	h.emitEvent(events.PotChanged{
		TableID:        h.TableID,
		HandID:         h.ID,
		PreviousAmount: previousAmount,
		NewAmount:      h.Pot,
		At:             time.Now(),
	})
}

func (h *Hand) decreasePot(amount int) {
	previousAmount := h.Pot
	h.Pot -= amount

	// Emit PotChanged event
	h.emitEvent(events.PotChanged{
		TableID:        h.TableID,
		HandID:         h.ID,
		PreviousAmount: previousAmount,
		NewAmount:      h.Pot,
		At:             time.Now(),
	})
}

func (h *Hand) resetPot() {
	previousAmount := h.Pot
	h.Pot = 0

	// Emit PotChanged event
	h.emitEvent(events.PotChanged{
		TableID:        h.TableID,
		HandID:         h.ID,
		PreviousAmount: previousAmount,
		NewAmount:      h.Pot,
		At:             time.Now(),
	})
}

func (h *Hand) setPot(value int) {
	previousAmount := h.Pot
	h.Pot = value

	// Emit PotChanged event
	h.emitEvent(events.PotChanged{
		TableID:        h.TableID,
		HandID:         h.ID,
		PreviousAmount: previousAmount,
		NewAmount:      h.Pot,
		At:             time.Now(),
	})
}

func (h *Hand) areAllAntesPaid() bool {
	return len(h.AntesPaid) == len(h.ActivePlayers)
}

func (h *Hand) addToPlayerAntesPaid(playerID string, amount int) {
	if _, ok := h.AntesPaid[playerID]; !ok {
		h.AntesPaid[playerID] = 0
	}
	h.AntesPaid[playerID] += amount
}

func (h *Hand) hasAlreadyPlacedAnte(playerID string) bool {
	_, paid := h.AntesPaid[playerID]
	return paid
}

func (h *Hand) getPlayerByIndex(index int) Player {
	if index < 0 || index >= len(h.Players) {
		return Player{}
	}
	return h.Players[index]
}

// getNextActiveBettor returns the next active player who should bet
func (h *Hand) getNextActiveBettor(currentBettorID string) string {
	if len(h.Players) == 0 {
		return ""
	}

	// Find current bettor's position
	currentPos := -1
	for i, player := range h.Players {
		if player.ID == currentBettorID {
			currentPos = i
			break
		}
	}

	if currentPos == -1 {
		return ""
	}

	// Find next active player
	pos := (currentPos + 1) % len(h.Players)
	for i := 0; i < len(h.Players); i++ {
		if pos == currentPos {
			break // We've come full circle
		}

		playerID := h.Players[pos].ID
		if h.ActivePlayers[playerID] {
			return playerID
		}
		pos = (pos + 1) % len(h.Players)
	}

	return ""
}
