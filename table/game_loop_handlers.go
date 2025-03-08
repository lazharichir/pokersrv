package table

import (
	"time"

	"github.com/lazharichir/poker/cards"
	"github.com/lazharichir/poker/events"
)

// State handler implementations

// handleWaitingForPlayersState waits for enough players to start a hand
func (g *GameLoop) handleWaitingForPlayersState() {
	// Check if we have enough players
	if len(g.players) >= 2 {
		// We have enough players, start a new hand
		g.startNewHand()
	}
	// Otherwise stay in this state until more players join
}

// handleAnteCollectionState handles the ante collection phase
func (g *GameLoop) handleAnteCollectionState() {
	// Set a timeout for ante collection (e.g., 10 seconds)
	timeout := time.NewTimer(10 * time.Second)

	// Create a map to track which players have placed antes
	antePlaced := make(map[string]bool)

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		defer timeout.Stop()

		for {
			select {
			case <-g.ctx.Done():
				// Game loop is shutting down
				return

			case <-timeout.C:
				// Timeout reached, remove players who didn't place ante
				g.stateUpdateLock.Lock()

				// Identify players who didn't place ante
				var stillActive []string
				for _, playerID := range g.activePlayers {
					if placed := antePlaced[playerID]; placed {
						stillActive = append(stillActive, playerID)
					}
				}
				g.activePlayers = stillActive

				// If we still have enough players, proceed to next phase
				if len(g.activePlayers) >= 2 {
					g.stateUpdateLock.Unlock()
					g.transitionTo(GameStateDealingHoleCards)
				} else {
					// Not enough players remaining
					g.stateUpdateLock.Unlock()
					g.transitionTo(GameStateHandComplete)
				}
				return

			case action := <-g.actionChan:
				if action.Action == "place_ante" {
					// Mark player as having placed ante
					antePlaced[action.PlayerID] = true

					// Create and store ante placed event
					event := events.AntePlacedByPlayer{
						TableID:  g.tableID,
						PlayerID: action.PlayerID,
						Amount:   g.rules.AnteValue,
					}
					g.eventStore.Append(event)

					// Check if all active players have placed ante
					allPlaced := true
					for _, playerID := range g.activePlayers {
						if !antePlaced[playerID] {
							allPlaced = false
							break
						}
					}

					if allPlaced {
						// All players placed ante, move to next phase
						g.transitionTo(GameStateDealingHoleCards)
						return
					}
				}
			}
		}
	}()
}

// handleDealingHoleCardsState deals hole cards to all active players
func (g *GameLoop) handleDealingHoleCardsState() {
	// Create and shuffle a deck
	deck := cards.ShuffleCards(cards.NewDeck52())

	// Deal 2 cards to each active player
	for _, playerID := range g.activePlayers {
		// Deal first card
		card1, remainingDeck := cards.DealCard(deck)
		deck = remainingDeck

		event1 := events.PlayerHoleCardDealt{
			TableID:  g.tableID,
			PlayerID: playerID,
			Card:     card1,
		}
		g.eventStore.Append(event1)

		// Deal second card
		card2, remainingDeck := cards.DealCard(deck)
		deck = remainingDeck

		event2 := events.PlayerHoleCardDealt{
			TableID:  g.tableID,
			PlayerID: playerID,
			Card:     card2,
		}
		g.eventStore.Append(event2)
	}

	// Move to continuation bet phase
	g.transitionTo(GameStateContinuationBets)
}

// handleContinuationBetsState handles the continuation bet phase
func (g *GameLoop) handleContinuationBetsState() {
	// Set a timeout for continuation bets (e.g., 15 seconds)
	timeout := time.NewTimer(15 * time.Second)

	// Track players who've acted
	continuationBets := make(map[string]bool)
	folded := make(map[string]bool)

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		defer timeout.Stop()

		for {
			select {
			case <-g.ctx.Done():
				// Game loop is shutting down
				return

			case <-timeout.C:
				// Timeout reached, fold players who didn't act
				g.stateUpdateLock.Lock()

				// Identify players who are still active
				var stillActive []string
				for _, playerID := range g.activePlayers {
					if folded[playerID] {
						continue // Player folded
					}
					if !continuationBets[playerID] {
						// Player didn't act, auto-fold them
						folded[playerID] = true
						event := events.PlayerFolded{
							TableID:  g.tableID,
							PlayerID: playerID,
						}
						g.eventStore.Append(event)
					} else {
						stillActive = append(stillActive, playerID)
					}
				}
				g.activePlayers = stillActive

				// Check if game can continue
				if len(g.activePlayers) >= 1 {
					g.stateUpdateLock.Unlock()
					g.transitionTo(GameStateDealingCommunity)
				} else {
					// No players remaining
					g.stateUpdateLock.Unlock()
					g.transitionTo(GameStateHandComplete)
				}
				return

			case action := <-g.actionChan:
				if action.Action == "place_continuation_bet" {
					// Mark player as having placed continuation bet
					continuationBets[action.PlayerID] = true

					// Create and store continuation bet event
					event := events.ContinuationBetPlaced{
						TableID:  g.tableID,
						PlayerID: action.PlayerID,
						Amount:   g.rules.AnteValue * g.rules.ContinuationBetMultiplier,
					}
					g.eventStore.Append(event)

					// Check if all active players have acted (bet or folded)
					allActed := true
					var stillActive []string

					for _, playerID := range g.activePlayers {
						if !continuationBets[playerID] && !folded[playerID] {
							allActed = false
							break
						}
						if !folded[playerID] {
							stillActive = append(stillActive, playerID)
						}
					}

					// Update active players list
					g.stateUpdateLock.Lock()
					g.activePlayers = stillActive
					g.stateUpdateLock.Unlock()

					if allActed {
						// All players have acted, check if we can continue
						if len(stillActive) >= 1 {
							g.transitionTo(GameStateDealingCommunity)
						} else {
							g.transitionTo(GameStateHandComplete)
						}
						return
					}

				} else if action.Action == "fold" {
					// Mark player as folded
					folded[action.PlayerID] = true

					// Create and store player folded event
					event := events.PlayerFolded{
						TableID:  g.tableID,
						PlayerID: action.PlayerID,
					}
					g.eventStore.Append(event)

					// Check if all active players have acted (bet or folded)
					allActed := true
					var stillActive []string

					for _, playerID := range g.activePlayers {
						if !continuationBets[playerID] && !folded[playerID] {
							allActed = false
							break
						}
						if !folded[playerID] {
							stillActive = append(stillActive, playerID)
						}
					}

					// Update active players list
					g.stateUpdateLock.Lock()
					g.activePlayers = stillActive
					g.stateUpdateLock.Unlock()

					// Check if only one player remains
					if len(stillActive) == 1 {
						// Only one player left, they win by default
						g.transitionTo(GameStateHandEvaluation)
						return
					}

					if allActed {
						// All players have acted, check if we can continue
						if len(stillActive) >= 1 {
							g.transitionTo(GameStateDealingCommunity)
						} else {
							g.transitionTo(GameStateHandComplete)
						}
						return
					}
				}
			}
		}
	}()
}

// handleDealingCommunityState deals community cards
func (g *GameLoop) handleDealingCommunityState() {
	// Create and shuffle a deck
	deck := cards.ShuffleCards(cards.NewDeck52())

	// Deal 8 community cards
	communityCards, _ := cards.DealCards(deck, 8)

	// Store the community cards event
	event := events.CommunityCardsDealt{
		TableID: g.tableID,
		Cards:   communityCards,
	}
	g.eventStore.Append(event)

	// Move to discard phase
	g.transitionTo(GameStateDiscardPhase)
}

// handleDiscardPhaseState manages the discard phase
func (g *GameLoop) handleDiscardPhaseState() {
	// Set a timeout for the discard phase based on table rules
	discardTimeout := time.Duration(g.rules.DiscardPhaseDuration) * time.Second
	timeout := time.NewTimer(discardTimeout)

	// Track which players have completed their discard action
	discardedOrSkipped := make(map[string]bool)

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		defer timeout.Stop()

		for {
			select {
			case <-g.ctx.Done():
				// Game loop is shutting down
				return

			case <-timeout.C:
				// Discard phase has timed out, move to card selection wave 1
				g.transitionTo(GameStateWave1Reveal)
				return

			case action := <-g.actionChan:
				if action.Action == "discard_card" {
					cardData, ok := action.Data.(map[string]interface{})
					if !ok {
						continue // Invalid data
					}

					// Mark player as having acted in discard phase
					discardedOrSkipped[action.PlayerID] = true

					// Extract card information
					card := cardData["card"].(cards.Card)
					discardFee := g.calculateDiscardFee()

					// Create and store discard event
					event := events.CardDiscarded{
						TableID:    g.tableID,
						PlayerID:   action.PlayerID,
						Card:       card,
						DiscardFee: discardFee,
					}
					g.eventStore.Append(event)

				} else if action.Action == "skip_discard" {
					// Player chose to skip discard
					discardedOrSkipped[action.PlayerID] = true
				}

				// Check if all active players have acted
				allActed := true
				for _, playerID := range g.activePlayers {
					if !discardedOrSkipped[playerID] {
						allActed = false
						break
					}
				}

				if allActed {
					// All players have acted, move to card selection wave 1
					g.transitionTo(GameStateWave1Reveal)
					return
				}
			}
		}
	}()
}

// calculateDiscardFee determines the cost to discard a card based on table rules
func (g *GameLoop) calculateDiscardFee() int {
	switch g.rules.DiscardCostType {
	case "fixed":
		return g.rules.DiscardCostValue
	case "ante_multiple":
		return g.rules.AnteValue * g.rules.DiscardCostValue
	case "bet_multiple":
		totalBet := g.rules.AnteValue + (g.rules.AnteValue * g.rules.ContinuationBetMultiplier)
		return totalBet * g.rules.DiscardCostValue
	default:
		return g.rules.DiscardCostValue // Default to fixed value
	}
}

// handleWave1RevealState handles the first wave of community card reveals
func (g *GameLoop) handleWave1RevealState() {
	// In a real implementation, we would publish an event that Wave 1 has started
	// and which cards are revealed (first 3)

	// Wait 5 seconds before transitioning to Wave 2
	timer := time.NewTimer(5 * time.Second)

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		select {
		case <-g.ctx.Done():
			// Game loop is shutting down
			timer.Stop()
			return
		case <-timer.C:
			// Time to reveal Wave 2
			g.transitionTo(GameStateWave2Reveal)
		}
	}()

	// While waiting, g.actionChan will capture any card selection actions
}

// handleWave2RevealState handles the second wave of community card reveals
func (g *GameLoop) handleWave2RevealState() {
	// In a real implementation, we would publish an event that Wave 2 has started
	// and which additional cards are revealed (next 3)

	// Wait 3 seconds before transitioning to Wave 3
	timer := time.NewTimer(3 * time.Second)

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		select {
		case <-g.ctx.Done():
			// Game loop is shutting down
			timer.Stop()
			return
		case <-timer.C:
			// Time to reveal Wave 3
			g.transitionTo(GameStateWave3Reveal)
		}
	}()

	// While waiting, g.actionChan will capture any card selection actions
}

// handleWave3RevealState handles the third wave of community card reveals
func (g *GameLoop) handleWave3RevealState() {
	// In a real implementation, we would publish an event that Wave 3 has started
	// and which additional cards are revealed (final 2)

	// Wait 2 seconds before transitioning to hand evaluation
	timer := time.NewTimer(2 * time.Second)

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		select {
		case <-g.ctx.Done():
			// Game loop is shutting down
			timer.Stop()
			return
		case <-timer.C:
			// Time for hand evaluation
			g.transitionTo(GameStateHandEvaluation)
		}
	}()

	// While waiting, g.actionChan will capture any card selection actions
}

// handleCardSelectionAction processes a player's card selection during the reveal phases
func (g *GameLoop) handleCardSelectionAction(action PlayerAction) {
	if action.Action != "select_card" {
		return
	}

	cardData, ok := action.Data.(map[string]interface{})
	if !ok {
		return // Invalid data
	}

	card := cardData["card"].(cards.Card)

	// Create and store card selection event
	event := events.CommunityCardSelected{
		TableID:  g.tableID,
		PlayerID: action.PlayerID,
		Card:     card,
	}
	g.eventStore.Append(event)
}

// handleHandEvaluationState evaluates all player hands and determines winners
func (g *GameLoop) handleHandEvaluationState() {
	// In a real implementation, we would:
	// 1. Gather all player hands (hole cards + selected community cards)
	// 2. Evaluate each hand's strength
	// 3. Determine winners
	// 4. Create appropriate events

	// For now, we'll just transition to showdown
	g.transitionTo(GameStateShowdown)
}

// handleShowdownState reveals all hands and announces winners
func (g *GameLoop) handleShowdownState() {
	// Simulate a brief pause for dramatic effect before completing the hand
	timer := time.NewTimer(2 * time.Second)

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		select {
		case <-g.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			// Hand complete
			g.transitionTo(GameStateHandComplete)
		}
	}()
}

// handleHandCompleteState finalizes the hand and prepares for the next hand
func (g *GameLoop) handleHandCompleteState() {
	// Reset for next hand
	g.handID = ""

	// Check if we should start a new hand
	if len(g.players) >= 2 {
		// Wait a short period before starting the next hand
		timer := time.NewTimer(5 * time.Second)

		g.wg.Add(1)
		go func() {
			defer g.wg.Done()
			select {
			case <-g.ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				// Start a new hand
				g.startNewHand()
			}
		}()
	} else {
		// Not enough players, go back to waiting
		g.transitionTo(GameStateWaitingForPlayers)
	}
}

// handleAnteAction processes a player's ante action
func (g *GameLoop) handleAnteAction(action PlayerAction) {
	// Only process "place_ante" actions
	if action.Action != "place_ante" {
		return
	}

	// Verify the player is in the active players list
	isActive := false
	for _, playerID := range g.activePlayers {
		if playerID == action.PlayerID {
			isActive = true
			break
		}
	}

	if !isActive {
		return // Player not active in this hand
	}

	// Extract ante amount from data if provided, otherwise use table rules
	anteAmount := g.rules.AnteValue
	if data, ok := action.Data.(map[string]interface{}); ok {
		if amount, ok := data["amount"].(int); ok {
			anteAmount = amount
		}
	}

	// Create and store ante placed event
	event := events.AntePlacedByPlayer{
		TableID:  g.tableID,
		PlayerID: action.PlayerID,
		Amount:   anteAmount,
	}
	g.eventStore.Append(event)
}

// handleContinuationBetAction processes a player's continuation bet or fold action
func (g *GameLoop) handleContinuationBetAction(action PlayerAction) {
	// Verify the player is in the active players list
	isActive := false
	for _, playerID := range g.activePlayers {
		if playerID == action.PlayerID {
			isActive = true
			break
		}
	}

	if !isActive {
		return // Player not active in this hand
	}

	switch action.Action {
	case "place_continuation_bet":
		// Extract bet amount from data if provided, otherwise calculate from table rules
		betAmount := g.rules.AnteValue * g.rules.ContinuationBetMultiplier
		if data, ok := action.Data.(map[string]interface{}); ok {
			if amount, ok := data["amount"].(int); ok {
				betAmount = amount
			}
		}

		// Create and store continuation bet event
		event := events.ContinuationBetPlaced{
			TableID:  g.tableID,
			PlayerID: action.PlayerID,
			Amount:   betAmount,
		}
		g.eventStore.Append(event)

	case "fold":
		// Create and store player folded event
		event := events.PlayerFolded{
			TableID:  g.tableID,
			PlayerID: action.PlayerID,
		}
		g.eventStore.Append(event)

		// Remove player from active players list
		g.stateUpdateLock.Lock()
		var stillActive []string
		for _, id := range g.activePlayers {
			if id != action.PlayerID {
				stillActive = append(stillActive, id)
			}
		}
		g.activePlayers = stillActive
		g.stateUpdateLock.Unlock()

		// Check if only one player remains
		if len(g.activePlayers) == 1 {
			// Only one player left, they win by default
			g.transitionTo(GameStateHandEvaluation)
		} else if len(g.activePlayers) == 0 {
			// No players left
			g.transitionTo(GameStateHandComplete)
		}
	}
}

// handleDiscardAction processes a player's discard or skip discard action
func (g *GameLoop) handleDiscardAction(action PlayerAction) {
	// Verify the player is in the active players list
	isActive := false
	for _, playerID := range g.activePlayers {
		if playerID == action.PlayerID {
			isActive = true
			break
		}
	}

	if !isActive {
		return // Player not active in this hand
	}

	switch action.Action {
	case "discard_card":
		// Extract card information from the action data
		var card cards.Card
		var cardIndex int = -1

		if data, ok := action.Data.(map[string]interface{}); ok {
			// Try to extract card object directly
			if cardObj, ok := data["card"].(cards.Card); ok {
				card = cardObj
			} else if cardIdxVal, ok := data["cardIndex"].(int); ok {
				// If card object not provided, try to use index
				cardIndex = cardIdxVal
			}
		}

		// If we got a card index but not a card object, try to get the card from community cards
		// Note: This would require having access to the community cards state
		// For now, we'll just log a warning that the card wasn't found

		if (card == cards.Card{}) && cardIndex == -1 {
			// Invalid card data
			return
		}

		// Calculate discard fee
		discardFee := g.calculateDiscardFee()

		// Create and store discard event
		event := events.CardDiscarded{
			TableID:    g.tableID,
			PlayerID:   action.PlayerID,
			Card:       card,
			DiscardFee: discardFee,
		}
		g.eventStore.Append(event)

	case "skip_discard":
		// Player chose to skip discard - no specific event needed
		// The handler in handleDiscardPhaseState will track this using the discardedOrSkipped map
	}
}
