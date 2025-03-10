package handlers

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lazharichir/poker/domain"
	"github.com/lazharichir/poker/domain/commands"
	"github.com/lazharichir/poker/server/connection"
)

// CommandRouter routes incoming commands to the appropriate handler
type CommandRouter struct {
	lobby   *domain.Lobby
	connMgr *connection.Manager
}

// NewCommandRouter creates a new command router
func NewCommandRouter(lobby *domain.Lobby, connMgr *connection.Manager) *CommandRouter {
	return &CommandRouter{
		lobby:   lobby,
		connMgr: connMgr,
	}
}

// HandleCommand processes an incoming command message
func (r *CommandRouter) HandleCommand(client *connection.Client, message []byte) error {
	// First determine command type
	var baseCmd struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(message, &baseCmd); err != nil {
		return err
	}

	// Route to appropriate handler based on command type
	switch baseCmd.Name {
	case commands.EnterLobby{}.Name():
		var cmd commands.EnterLobby
		if err := json.Unmarshal(message, &cmd); err != nil {
			return err
		}
		return r.handleEnterLobby(client, cmd)

	case commands.LeaveLobby{}.Name():
		var cmd commands.LeaveLobby
		if err := json.Unmarshal(message, &cmd); err != nil {
			return err
		}
		return r.handleLeaveLobby(client, cmd)

	case commands.PlayerSeats{}.Name():
		var cmd commands.PlayerSeats
		if err := json.Unmarshal(message, &cmd); err != nil {
			return err
		}
		return r.handlePlayerSeats(client, cmd)

	case commands.PlayerLeavesTable{}.Name():
		var cmd commands.PlayerLeavesTable
		if err := json.Unmarshal(message, &cmd); err != nil {
			return err
		}
		return r.handlePlayerLeavesTable(client, cmd)

	case commands.PlayerBuysIn{}.Name():
		var cmd commands.PlayerBuysIn
		if err := json.Unmarshal(message, &cmd); err != nil {
			return err
		}
		return r.handlePlayerBuysIn(client, cmd)

	case commands.PlayerFolds{}.Name():
		var cmd commands.PlayerFolds
		if err := json.Unmarshal(message, &cmd); err != nil {
			return err
		}
		return r.handlePlayerFolds(client, cmd)

	case commands.PlayerPlacesAnte{}.Name():
		var cmd commands.PlayerPlacesAnte
		if err := json.Unmarshal(message, &cmd); err != nil {
			return err
		}
		return r.handlePlayerPlacesAnte(client, cmd)

	case commands.PlayerPlacesContinuationBet{}.Name():
		var cmd commands.PlayerPlacesContinuationBet
		if err := json.Unmarshal(message, &cmd); err != nil {
			return err
		}
		return r.handlePlayerPlacesContinuationBet(client, cmd)

	case commands.PlayerSelectsCommunityCard{}.Name():
		var cmd commands.PlayerSelectsCommunityCard
		if err := json.Unmarshal(message, &cmd); err != nil {
			return err
		}
		return r.handlePlayerSelectsCommunityCard(client, cmd)

	default:
		fmt.Println("unknown command type", baseCmd.Name)
		return errors.New("unknown command type")
	}
}

func (r *CommandRouter) handleEnterLobby(client *connection.Client, cmd commands.EnterLobby) error {
	// Initialize Player if not already set
	if client.Player == nil {
		// Create a new player - in future we'd fetch this from a database
		client.Player = &domain.Player{
			ID:      cmd.PlayerID,
			Name:    cmd.PlayerName,
			Status:  "active",
			Balance: 1_000, // Default starting balance
		}

		// Register the player ID with the client ID in the connection manager
		r.connMgr.AddPlayerToClient(client.ID, cmd.PlayerID)
	}

	if err := r.lobby.EntersLobby(client.Player); err != nil {
		return err
	}
	return nil
}

func (r *CommandRouter) handleLeaveLobby(client *connection.Client, cmd commands.LeaveLobby) error {
	if err := r.lobby.LeavesLobby(cmd.PlayerID); err != nil {
		return err
	}
	return nil
}

// Command handler implementations
func (r *CommandRouter) handlePlayerSeats(client *connection.Client, cmd commands.PlayerSeats) error {
	if !r.lobby.IsInLobby(client.Player.ID) {
		return errors.New("client is not in the lobby")
	}

	table, err := r.lobby.GetTable(cmd.TableID)
	if err != nil {
		return err
	}

	player := client.Player

	if err := table.SeatPlayer(player); err != nil {
		return err
	}

	client.TableIDs = append(client.TableIDs, cmd.TableID)

	return nil
}

func (r *CommandRouter) handlePlayerLeavesTable(client *connection.Client, cmd commands.PlayerLeavesTable) error {
	table, err := r.lobby.GetTable(cmd.TableID)
	if err != nil {
		return err
	}

	if err := table.PlayerLeaves(client.Player.ID); err != nil {
		return err
	}

	for i, tableID := range client.TableIDs {
		if tableID == cmd.TableID {
			client.TableIDs = append(client.TableIDs[:i], client.TableIDs[i+1:]...)
			break
		}
	}

	return nil
}

func (r *CommandRouter) handlePlayerBuysIn(client *connection.Client, cmd commands.PlayerBuysIn) error {
	if !r.lobby.IsInLobby(client.Player.ID) {
		return errors.New("client is not in the lobby")
	}

	table, err := r.lobby.GetTable(cmd.TableID)
	if err != nil {
		return err
	}

	if err := table.PlayerBuysIn(client.Player.ID, cmd.Amount); err != nil {
		return err
	}

	return nil
}

func (r *CommandRouter) handlePlayerFolds(client *connection.Client, cmd commands.PlayerFolds) error {
	if !r.lobby.IsInLobby(client.Player.ID) {
		return errors.New("client is not in the lobby")
	}

	table, err := r.lobby.GetTable(cmd.TableID)
	if err != nil {
		return err
	}

	hand, err := table.GetHandByID(cmd.HandID)
	if err != nil {
		return err
	}

	if err := hand.PlayerFolds(client.Player.ID); err != nil {
		return err
	}

	return nil
}

func (r *CommandRouter) handlePlayerPlacesAnte(client *connection.Client, cmd commands.PlayerPlacesAnte) error {
	if !r.lobby.IsInLobby(client.Player.ID) {
		return errors.New("client is not in the lobby")
	}

	table, err := r.lobby.GetTable(cmd.TableID)
	if err != nil {
		return err
	}

	hand, err := table.GetHandByID(cmd.HandID)
	if err != nil {
		return err
	}

	if err := hand.PlayerPlacesAnte(client.Player.ID, cmd.Amount); err != nil {
		return err
	}

	return nil
}

func (r *CommandRouter) handlePlayerPlacesContinuationBet(client *connection.Client, cmd commands.PlayerPlacesContinuationBet) error {
	if !r.lobby.IsInLobby(client.Player.ID) {
		return errors.New("client is not in the lobby")
	}

	table, err := r.lobby.GetTable(cmd.TableID)
	if err != nil {
		return err
	}

	hand, err := table.GetHandByID(cmd.HandID)
	if err != nil {
		return err
	}

	if err := hand.PlayerPlacesContinuationBet(client.Player.ID, cmd.Amount); err != nil {
		return err
	}

	return nil
}

func (r *CommandRouter) handlePlayerSelectsCommunityCard(client *connection.Client, cmd commands.PlayerSelectsCommunityCard) error {
	if !r.lobby.IsInLobby(client.Player.ID) {
		return errors.New("client is not in the lobby")
	}

	table, err := r.lobby.GetTable(cmd.TableID)
	if err != nil {
		return err
	}

	hand, err := table.GetHandByID(cmd.HandID)
	if err != nil {
		return err
	}

	if err := hand.PlayerSelectsCommunityCard(client.Player.ID, cmd.Card); err != nil {
		return err
	}

	return nil
}
