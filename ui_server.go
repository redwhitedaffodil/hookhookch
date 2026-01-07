package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

//go:embed ui/templates/*
var templatesFS embed.FS

// UIServer represents the web UI server
type UIServer struct {
	address      string
	engineServer *EngineServer
	mu           sync.RWMutex
	running      bool
	config       *UIConfig
}

// UIConfig represents the UI and engine configuration
type UIConfig struct {
	EnginePath      string `json:"enginePath"`
	Threads         int    `json:"threads"`
	Hash            int    `json:"hash"`
	Depth           int    `json:"depth"`
	MultiPV         int    `json:"multipv"`
	Address         string `json:"address"`
	AuthWrite       bool   `json:"authWrite"`
	LocalhostBypass bool   `json:"localhostBypass"`
	Passkey         string `json:"passkey"`
}

// NewUIServer creates a new UI server
func NewUIServer(address string) *UIServer {
	return &UIServer{
		address: address,
		config: &UIConfig{
			EnginePath:      "stockfish",
			Threads:         4,
			Hash:            256,
			Depth:           20,
			MultiPV:         3,
			Address:         "localhost:8080",
			AuthWrite:       true,
			LocalhostBypass: true,
		},
	}
}

// Start starts the UI server
func (s *UIServer) Start() error {
	logger.Printf("Starting UI server on %s\n", s.address)

	// Setup routes
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/config", s.handleConfig)
	http.HandleFunc("/api/userscript", s.handleUserscript)
	http.HandleFunc("/api/server/start", s.handleServerStart)
	http.HandleFunc("/api/server/stop", s.handleServerStop)

	return http.ListenAndServe(s.address, nil)
}

func (s *UIServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Try to read from filesystem first, fallback to embedded
	var tmplData []byte
	var err error

	tmplData, err = os.ReadFile("ui/templates/index.html")
	if err != nil {
		// Fallback to embedded
		tmplData, err = templatesFS.ReadFile("ui/templates/index.html")
		if err != nil {
			http.Error(w, "Template not found", http.StatusInternalServerError)
			return
		}
	}

	tmpl, err := template.New("index").Parse(string(tmplData))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl.Execute(w, nil)
}

func (s *UIServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		defer s.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.config)

	case http.MethodPost:
		var newConfig UIConfig
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		s.config = &newConfig
		s.mu.Unlock()

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *UIServer) handleUserscript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqConfig struct {
		Engine   string `json:"engine"`
		AutoMove bool   `json:"autoMove"`
		ArrowColor string `json:"arrowColor"`
		WsURL    string `json:"wsURL"`
		PassKey  string `json:"passKey"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqConfig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate userscript
	config := UserscriptConfig{
		Engine:            reqConfig.Engine,
		AutoMove:          fmt.Sprintf("%t", reqConfig.AutoMove),
		ArrowColor:        reqConfig.ArrowColor,
		ExternalEngineURL: reqConfig.WsURL,
		PassKey:           reqConfig.PassKey,
	}

	var builder strings.Builder
	if err := GenerateUserscriptToWriter(&builder, config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Content-Disposition", "attachment; filename=chesshook.user.js")
	io.WriteString(w, builder.String())
}

func (s *UIServer) handleServerStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		http.Error(w, "Server already running", http.StatusConflict)
		return
	}

	engineConfig := &EngineConfig{
		Address:         s.config.Address,
		EnginePath:      s.config.EnginePath,
		Threads:         s.config.Threads,
		Hash:            s.config.Hash,
		MultiPV:         s.config.MultiPV,
		Depth:           s.config.Depth,
		RequireAuth:     s.config.AuthWrite,
		LocalhostBypass: s.config.LocalhostBypass,
	}

	engineServer, err := NewEngineServer(engineConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.engineServer = engineServer
	s.config.Passkey = engineServer.passKey

	// Start in background
	go func() {
		if err := engineServer.Start(); err != nil {
			logger.Printf("Engine server error: %v\n", err)
		}
	}()

	s.running = true

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"passkey": engineServer.passKey,
	})
}

func (s *UIServer) handleServerStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		http.Error(w, "Server not running", http.StatusConflict)
		return
	}

	if s.engineServer != nil && s.engineServer.engine != nil {
		s.engineServer.engine.Stop()
	}

	s.running = false
	s.engineServer = nil

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
