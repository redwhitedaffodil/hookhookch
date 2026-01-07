package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// EngineServer represents a WebSocket server for external engine access
type EngineServer struct {
	engine              *ChessEngine
	upgrader            websocket.Upgrader
	passKey             string
	engineInputChannel  chan string
	engineOutputChannel chan string
	users               map[*websocket.Conn]*EngineUser
	usersMu             sync.RWMutex
	address             string
	requireAuth         bool
	localhostBypass     bool
	engineLock          sync.Mutex
	engineOwner         *websocket.Conn
}

// EngineUser represents a connected user
type EngineUser struct {
	conn         *websocket.Conn
	authenticated bool
	subscribed    bool
	hasLock       bool
}

// EngineConfig represents engine server configuration
type EngineConfig struct {
	Address         string
	EnginePath      string
	Threads         int
	Hash            int
	MultiPV         int
	Depth           int
	RequireAuth     bool
	LocalhostBypass bool
}

// NewEngineServer creates a new engine server
func NewEngineServer(config *EngineConfig) (*EngineServer, error) {
	// Generate random passkey
	passKeyBytes := make([]byte, 16)
	if _, err := rand.Read(passKeyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate passkey: %w", err)
	}
	passKey := hex.EncodeToString(passKeyBytes)

	engine := NewChessEngine(config.EnginePath, config.Threads, config.Hash, config.MultiPV, config.Depth)

	return &EngineServer{
		engine: engine,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
		passKey:             passKey,
		engineInputChannel:  make(chan string, 100),
		engineOutputChannel: make(chan string, 100),
		users:               make(map[*websocket.Conn]*EngineUser),
		address:             config.Address,
		requireAuth:         config.RequireAuth,
		localhostBypass:     config.LocalhostBypass,
	}, nil
}

// Start starts the engine server
func (s *EngineServer) Start() error {
	logger.Printf("Starting engine server on %s\n", s.address)
	logger.Printf("Passkey: %s\n", s.passKey)

	// Start engine
	if err := s.engine.Start(); err != nil {
		return fmt.Errorf("failed to start engine: %w", err)
	}

	// Setup HTTP handlers
	http.HandleFunc("/ws", s.handleWebSocket)
	http.HandleFunc("/", s.handleRoot)

	// Start server
	return http.ListenAndServe(s.address, nil)
}

func (s *EngineServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head><title>ChessHook Engine Server</title></head>
<body>
<h1>ChessHook Engine Server</h1>
<p>Status: Running</p>
<p>Connect via WebSocket at: ws://` + s.address + `/ws</p>
<p>Passkey: <code>` + s.passKey + `</code></p>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (s *EngineServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	user := &EngineUser{
		conn:          conn,
		authenticated: false,
		subscribed:    false,
		hasLock:       false,
	}

	// Check if localhost bypass is enabled
	if s.localhostBypass && isLocalhost(r.RemoteAddr) {
		user.authenticated = true
	}

	s.usersMu.Lock()
	s.users[conn] = user
	s.usersMu.Unlock()

	logger.Printf("New WebSocket connection from %s\n", r.RemoteAddr)

	// Send greeting
	conn.WriteMessage(websocket.TextMessage, []byte("whoareyou"))

	go s.readPump(conn, user)
	go s.writePump(conn, user)
}

func (s *EngineServer) readPump(conn *websocket.Conn, user *EngineUser) {
	defer func() {
		s.usersMu.Lock()
		delete(s.users, conn)
		if user.hasLock && s.engineOwner == conn {
			s.engineOwner = nil
		}
		s.usersMu.Unlock()
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		msg := strings.TrimSpace(string(message))
		s.handleMessage(conn, user, msg)
	}
}

func (s *EngineServer) writePump(conn *websocket.Conn, user *EngineUser) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message := <-s.engineOutputChannel:
			if user.subscribed {
				conn.WriteMessage(websocket.TextMessage, []byte(message))
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *EngineServer) handleMessage(conn *websocket.Conn, user *EngineUser, msg string) {
	parts := strings.Fields(msg)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]

	switch cmd {
	case "iam":
		conn.WriteMessage(websocket.TextMessage, []byte("auth required"))
	case "auth":
		if len(parts) < 2 {
			conn.WriteMessage(websocket.TextMessage, []byte("auth failed: missing passkey"))
			return
		}
		if parts[1] == s.passKey {
			user.authenticated = true
			conn.WriteMessage(websocket.TextMessage, []byte("auth success"))
		} else {
			conn.WriteMessage(websocket.TextMessage, []byte("auth failed"))
		}
	case "lock":
		if !user.authenticated && s.requireAuth {
			conn.WriteMessage(websocket.TextMessage, []byte("error: not authenticated"))
			return
		}
		s.engineLock.Lock()
		if s.engineOwner != nil && s.engineOwner != conn {
			s.engineLock.Unlock()
			conn.WriteMessage(websocket.TextMessage, []byte("error: engine locked by another user"))
			return
		}
		s.engineOwner = conn
		user.hasLock = true
		s.engineLock.Unlock()
		conn.WriteMessage(websocket.TextMessage, []byte("lock acquired"))
	case "unlock":
		s.engineLock.Lock()
		if s.engineOwner == conn {
			s.engineOwner = nil
			user.hasLock = false
		}
		s.engineLock.Unlock()
		conn.WriteMessage(websocket.TextMessage, []byte("lock released"))
	case "sub":
		user.subscribed = true
		conn.WriteMessage(websocket.TextMessage, []byte("subscribed"))
	case "unsub":
		user.subscribed = false
		conn.WriteMessage(websocket.TextMessage, []byte("unsubscribed"))
	case "position":
		if !user.hasLock {
			conn.WriteMessage(websocket.TextMessage, []byte("error: engine not locked"))
			return
		}
		// Forward to engine
		s.engineInputChannel <- msg
	case "go":
		if !user.hasLock {
			conn.WriteMessage(websocket.TextMessage, []byte("error: engine not locked"))
			return
		}
		// Parse and execute
		if len(parts) >= 3 && parts[1] == "movetime" {
			fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" // Default starting position
			// In real implementation, extract FEN from last "position" command
			thinkTime := time.Duration(1000) * time.Millisecond
			if len(parts) >= 3 {
				var ms int
				fmt.Sscanf(parts[2], "%d", &ms)
				thinkTime = time.Duration(ms) * time.Millisecond
			}
			analysis, err := s.engine.AnalyzePosition(fen, thinkTime)
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte("error: "+err.Error()))
				return
			}
			conn.WriteMessage(websocket.TextMessage, []byte("bestmove "+analysis.BestMove))
		}
	default:
		conn.WriteMessage(websocket.TextMessage, []byte("error: unknown command"))
	}
}

func isLocalhost(addr string) bool {
	return strings.HasPrefix(addr, "127.0.0.1") || strings.HasPrefix(addr, "[::1]") || strings.HasPrefix(addr, "localhost")
}
