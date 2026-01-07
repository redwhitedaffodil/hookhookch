package main

import (
	"fmt"
	"net/http"
	"time"
)

// GameStrategy represents a strategy for playing games
type GameStrategy struct {
	Name        string `json:"name"`
	ThinkTimeMs int    `json:"think_time_ms"`
	TimeMode    string `json:"time_mode"`
	AutoMove    bool   `json:"auto_move"`
}

// GamePlayer manages playing a single game
type GamePlayer struct {
	client   *http.Client
	account  *Account
	strategy *GameStrategy
	engine   *ChessEngine
	gameID   string
}

// NewGamePlayer creates a new game player
func NewGamePlayer(client *http.Client, account *Account, strategy *GameStrategy, engine *ChessEngine, gameID string) *GamePlayer {
	return &GamePlayer{
		client:   client,
		account:  account,
		strategy: strategy,
		engine:   engine,
		gameID:   gameID,
	}
}

// PlayGame plays a single game
func (gp *GamePlayer) PlayGame() error {
	logger.Printf("[%s] Starting game %s with strategy '%s'\n", gp.account.Username, gp.gameID, gp.strategy.Name)
	
	// Create game client
	gameClient := NewGameClient(gp.account.Cookie)
	defer gameClient.Close()
	
	// Connect to game
	if err := gameClient.ConnectToGame(gp.gameID); err != nil {
		// If WebSocket connection fails, return error with guidance
		logger.Printf("[%s] WebSocket connection note: %v\n", gp.account.Username, err)
		return fmt.Errorf("live game playing not yet fully implemented - requires chess.com CometD protocol reverse engineering. Please use the userscript approach (./chesshook2 userscript generate) for browser-based game playing: %w", err)
	}
	
	// Main game loop
	for {
		position := gameClient.GetCurrentPosition()
		
		if !position.IsMyTurn {
			// Wait for opponent's move
			_, err := gameClient.WaitForMove(5 * time.Minute)
			if err != nil {
				return fmt.Errorf("error waiting for move: %w", err)
			}
			continue
		}
		
		// Analyze position and get best move
		thinkTime := time.Duration(gp.strategy.ThinkTimeMs) * time.Millisecond
		analysis, err := gp.engine.AnalyzePosition(position.FEN, thinkTime)
		if err != nil {
			return fmt.Errorf("error analyzing position: %w", err)
		}
		
		logger.Printf("[%s] Best move: %s (score: %d)\n", gp.account.Username, analysis.BestMove, analysis.Score)
		
		// Send move
		if err := gameClient.SendMove(analysis.BestMove); err != nil {
			return fmt.Errorf("error sending move: %w", err)
		}
		
		// Add delay for legit mode
		if gp.strategy.TimeMode == "legit" {
			delay := time.Duration(500+time.Now().UnixNano()%1500) * time.Millisecond
			time.Sleep(delay)
		}
	}
}

// PlayAllGamesForAccount plays all active games for an account
func PlayAllGamesForAccount(client *http.Client, account *Account, strategy *GameStrategy, engine *ChessEngine) error {
	logger.Printf("[%s] Looking for active games...\n", account.Username)
	
	games, err := FindActiveGames(client, account.Cookie)
	if err != nil {
		return fmt.Errorf("error finding active games: %w", err)
	}
	
	if len(games) == 0 {
		logger.Printf("[%s] No active games found\n", account.Username)
		return nil
	}
	
	logger.Printf("[%s] Found %d active games\n", account.Username, len(games))
	
	for _, game := range games {
		player := NewGamePlayer(client, account, strategy, engine, game.GameID)
		if err := player.PlayGame(); err != nil {
			logger.Printf("[%s] Error playing game %s: %v\n", account.Username, game.GameID, err)
			continue
		}
	}
	
	return nil
}
