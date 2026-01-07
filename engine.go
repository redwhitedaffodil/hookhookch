package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ChessEngine represents a UCI chess engine (e.g., Stockfish)
type ChessEngine struct {
	Path     string
	Threads  int
	Hash     int // Hash size in MB
	MultiPV  int // Number of principal variations
	Depth    int // Search depth limit
	cmd      *exec.Cmd
	stdin    *bufio.Writer
	stdout   *bufio.Scanner
	ready    bool
}

// EngineAnalysis represents the engine's analysis of a position
type EngineAnalysis struct {
	BestMove  string
	Score     int
	Depth     int
	Nodes     int64
	Time      int // in milliseconds
	PV        []string
	Variations []EngineVariation
}

// EngineVariation represents a principal variation from MultiPV analysis
type EngineVariation struct {
	Move  string
	Score int
	PV    []string
}

// NewChessEngine creates a new chess engine instance
func NewChessEngine(path string, threads, hash, multipv, depth int) *ChessEngine {
	return &ChessEngine{
		Path:    path,
		Threads: threads,
		Hash:    hash,
		MultiPV: multipv,
		Depth:   depth,
		ready:   false,
	}
}

// Start initializes and starts the chess engine
func (e *ChessEngine) Start() error {
	e.cmd = exec.Command(e.Path)
	
	stdin, err := e.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	e.stdin = bufio.NewWriter(stdin)
	
	stdout, err := e.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	e.stdout = bufio.NewScanner(stdout)
	
	if err := e.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start engine: %w", err)
	}
	
	// Initialize UCI
	if err := e.sendCommand("uci"); err != nil {
		return err
	}
	
	// Wait for uciok
	for e.stdout.Scan() {
		line := e.stdout.Text()
		if strings.HasPrefix(line, "uciok") {
			break
		}
	}
	
	// Set options
	if err := e.sendCommand(fmt.Sprintf("setoption name Threads value %d", e.Threads)); err != nil {
		return err
	}
	if err := e.sendCommand(fmt.Sprintf("setoption name Hash value %d", e.Hash)); err != nil {
		return err
	}
	if e.MultiPV > 1 {
		if err := e.sendCommand(fmt.Sprintf("setoption name MultiPV value %d", e.MultiPV)); err != nil {
			return err
		}
	}
	
	// Send isready and wait for readyok
	if err := e.sendCommand("isready"); err != nil {
		return err
	}
	
	for e.stdout.Scan() {
		line := e.stdout.Text()
		if strings.HasPrefix(line, "readyok") {
			e.ready = true
			break
		}
	}
	
	return nil
}

// sendCommand sends a command to the engine
func (e *ChessEngine) sendCommand(cmd string) error {
	if _, err := e.stdin.WriteString(cmd + "\n"); err != nil {
		return fmt.Errorf("failed to write command: %w", err)
	}
	return e.stdin.Flush()
}

// AnalyzePosition analyzes a chess position and returns the best move
func (e *ChessEngine) AnalyzePosition(fen string, thinkTime time.Duration) (*EngineAnalysis, error) {
	if !e.ready {
		return nil, fmt.Errorf("engine not ready")
	}
	
	// Start new game
	if err := e.sendCommand("ucinewgame"); err != nil {
		return nil, err
	}
	
	// Wait for ready
	if err := e.sendCommand("isready"); err != nil {
		return nil, err
	}
	for e.stdout.Scan() {
		line := e.stdout.Text()
		if strings.HasPrefix(line, "readyok") {
			break
		}
	}
	
	// Set position
	if err := e.sendCommand(fmt.Sprintf("position fen %s", fen)); err != nil {
		return nil, err
	}
	
	// Start analysis
	var goCmd string
	if e.Depth > 0 {
		goCmd = fmt.Sprintf("go depth %d", e.Depth)
	} else {
		goCmd = fmt.Sprintf("go movetime %d", thinkTime.Milliseconds())
	}
	
	if err := e.sendCommand(goCmd); err != nil {
		return nil, err
	}
	
	analysis := &EngineAnalysis{
		Variations: make([]EngineVariation, 0),
	}
	
	// Parse output
	for e.stdout.Scan() {
		line := e.stdout.Text()
		
		if strings.HasPrefix(line, "info") {
			// Parse info lines for score, depth, nodes, etc.
			parts := strings.Fields(line)
			for i, part := range parts {
				switch part {
				case "depth":
					if i+1 < len(parts) {
						fmt.Sscanf(parts[i+1], "%d", &analysis.Depth)
					}
				case "nodes":
					if i+1 < len(parts) {
						fmt.Sscanf(parts[i+1], "%d", &analysis.Nodes)
					}
				case "time":
					if i+1 < len(parts) {
						fmt.Sscanf(parts[i+1], "%d", &analysis.Time)
					}
				case "score":
					if i+2 < len(parts) && parts[i+1] == "cp" {
						fmt.Sscanf(parts[i+2], "%d", &analysis.Score)
					}
				case "pv":
					if i+1 < len(parts) {
						analysis.PV = parts[i+1:]
					}
				}
			}
		} else if strings.HasPrefix(line, "bestmove") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				analysis.BestMove = parts[1]
			}
			break
		}
	}
	
	if analysis.BestMove == "" {
		return nil, fmt.Errorf("no best move found")
	}
	
	return analysis, nil
}

// Stop stops the chess engine
func (e *ChessEngine) Stop() error {
	if e.cmd != nil && e.cmd.Process != nil {
		if err := e.sendCommand("quit"); err != nil {
			return err
		}
		return e.cmd.Wait()
	}
	return nil
}
