package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// GameClient represents a WebSocket client for chess.com live games
type GameClient struct {
	conn           *websocket.Conn
	cookie         string
	gameID         string
	myColor        string
	currentFEN     string
	isMyTurn       bool
	moveChannel    chan string
	positionUpdate chan GamePosition
	stopChan       chan bool
	mu             sync.RWMutex
}

// GamePosition represents the current state of a game
type GamePosition struct {
	FEN      string
	MyColor  string
	IsMyTurn bool
	GameID   string
}

// GameMove represents a move in a game
type GameMove struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// NewGameClient creates a new game client
func NewGameClient(cookie string) *GameClient {
	return &GameClient{
		cookie:         cookie,
		moveChannel:    make(chan string, 10),
		positionUpdate: make(chan GamePosition, 10),
		stopChan:       make(chan bool),
	}
}

// ConnectToGame connects to a chess.com live game via WebSocket
func (gc *GameClient) ConnectToGame(gameID string) error {
	gc.mu.Lock()
	gc.gameID = gameID
	gc.mu.Unlock()
	
	// Note: This is a placeholder implementation
	// In reality, you would need to:
	// 1. Connect to wss://live.chess.com/cometd
	// 2. Handle CometD protocol handshake
	// 3. Subscribe to the game channel
	// 4. Parse game state updates
	
	// For now, return an error indicating this needs implementation
	return fmt.Errorf("live game WebSocket connection not yet implemented - requires reverse engineering chess.com's CometD protocol")
}

// SendMove sends a move to the game via WebSocket
func (gc *GameClient) SendMove(move string) error {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	
	if gc.conn == nil {
		return fmt.Errorf("not connected to game")
	}
	
	// Note: This is a placeholder
	// Would need to format move according to chess.com's protocol
	moveData := map[string]interface{}{
		"move":   move,
		"gameId": gc.gameID,
	}
	
	return gc.conn.WriteJSON(moveData)
}

// GetCurrentPosition returns the current game position
func (gc *GameClient) GetCurrentPosition() GamePosition {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	
	return GamePosition{
		FEN:      gc.currentFEN,
		MyColor:  gc.myColor,
		IsMyTurn: gc.isMyTurn,
		GameID:   gc.gameID,
	}
}

// WaitForMove waits for a move update with timeout
func (gc *GameClient) WaitForMove(timeout time.Duration) (string, error) {
	select {
	case move := <-gc.moveChannel:
		return move, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout waiting for move")
	case <-gc.stopChan:
		return "", fmt.Errorf("client stopped")
	}
}

// Close closes the game client connection
func (gc *GameClient) Close() error {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	
	close(gc.stopChan)
	
	if gc.conn != nil {
		return gc.conn.Close()
	}
	return nil
}

// readPump reads messages from the WebSocket connection
func (gc *GameClient) readPump() {
	defer func() {
		gc.Close()
	}()
	
	for {
		select {
		case <-gc.stopChan:
			return
		default:
			_, message, err := gc.conn.ReadMessage()
			if err != nil {
				logger.Printf("Error reading message: %v\n", err)
				return
			}
			
			// Parse game state update
			var update map[string]interface{}
			if err := json.Unmarshal(message, &update); err != nil {
				logger.Printf("Error parsing message: %v\n", err)
				continue
			}
			
			// Update game state
			gc.handleGameUpdate(update)
		}
	}
}

// handleGameUpdate handles a game state update from the server
func (gc *GameClient) handleGameUpdate(update map[string]interface{}) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	
	// Note: This would need to be implemented based on
	// the actual chess.com WebSocket protocol
	
	// Example placeholder logic:
	if fen, ok := update["fen"].(string); ok {
		gc.currentFEN = fen
	}
	
	if myTurn, ok := update["isMyTurn"].(bool); ok {
		gc.isMyTurn = myTurn
	}
	
	// Notify position update
	select {
	case gc.positionUpdate <- gc.GetCurrentPosition():
	default:
	}
}
