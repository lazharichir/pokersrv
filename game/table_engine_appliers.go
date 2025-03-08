package game

import (
	"github.com/lazharichir/poker/domain"
	"github.com/lazharichir/poker/events"
)

// Event handler implementations

func (te *TableEngine) applyHandStartedEvent(event events.HandStarted, table *domain.Table) {
	table.Pot = 0
	table.ButtonPlayerID = event.ButtonPlayerID

	// Reset all players for new hand
	for _, player := range table.Players {
		player.ResetForNewHand()
	}

	// Initialize active players list
	te.activePlayers = event.PlayerIDs
}

func (te *TableEngine) applyAntePlacedByPlayerEvent(event events.AntePlacedByPlayer, table *domain.Table) {
	if player, exists := table.Players[event.PlayerID]; exists {
		player.Chips -= event.Amount
		table.Pot += event.Amount
	}
}

func (te *TableEngine) applyPlayerHoleCardDealtEvent(event events.PlayerHoleCardDealt, table *domain.Table) {
	if player, exists := table.Players[event.PlayerID]; exists {
		player.HoleCards = append(player.HoleCards, event.Card)
	}
}

func (te *TableEngine) applyContinuationBetPlacedEvent(event events.ContinuationBetPlaced, table *domain.Table) {
	if player, exists := table.Players[event.PlayerID]; exists {
		player.Chips -= event.Amount
		player.CurrentBet += event.Amount
		table.Pot += event.Amount
	}
}

func (te *TableEngine) applyPlayerFoldedEvent(event events.PlayerFolded, table *domain.Table) {
	if player, exists := table.Players[event.PlayerID]; exists {
		player.Folded = true
	}

	// Update active players list
	for i, playerID := range te.activePlayers {
		if playerID == event.PlayerID {
			te.activePlayers = append(te.activePlayers[:i], te.activePlayers[i+1:]...)
			break
		}
	}
}

func (te *TableEngine) applyCommunityCardsDealtEvent(event events.CommunityCardsDealt, table *domain.Table) {
	table.CommunityCards = event.Cards
}

func (te *TableEngine) applyCardDiscardedEvent(event events.CardDiscarded, table *domain.Table) {
	if player, exists := table.Players[event.PlayerID]; exists {
		player.Chips -= event.DiscardFee
		table.Pot += event.DiscardFee
	}

	// Remove the discarded card from community cards
	for i, card := range table.CommunityCards {
		if card.Equals(event.Card) {
			table.CommunityCards = append(table.CommunityCards[:i], table.CommunityCards[i+1:]...)
			break
		}
	}
}

func (te *TableEngine) applyCommunityCardSelectedEvent(event events.CommunityCardSelected, table *domain.Table) {
	if player, exists := table.Players[event.PlayerID]; exists {
		player.SelectedCommunityCards = append(player.SelectedCommunityCards, event.Card)
	}
}

func (te *TableEngine) applyHandCompletedEvent(event events.HandCompleted, table *domain.Table) {
	// Update first place player's chips
	if event.FirstPlaceID != "" {
		if player, exists := table.Players[event.FirstPlaceID]; exists {
			player.Chips += event.FirstPrize
		}
	}

	// Update second place player's chips
	if event.SecondPlaceID != "" {
		if player, exists := table.Players[event.SecondPlaceID]; exists {
			player.Chips += event.SecondPrize
		}
	}

	// Reset pot
	table.Pot = 0
}
