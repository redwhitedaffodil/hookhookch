package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var gameCmd = &cobra.Command{
	Use:   "game",
	Short: "Manage live chess games",
	Long:  "Commands for playing live chess games on chess.com using engine assistance",
}

var gamePlayCmd = &cobra.Command{
	Use:   "play",
	Short: "Play all active games for all accounts",
	Long:  "Automatically play all active live games for all accounts in db.json",
	Run:   runGamePlay,
}

var gamePlayOneCmd = &cobra.Command{
	Use:   "playOne [username]",
	Short: "Play all active games for a single account",
	Args:  cobra.ExactArgs(1),
	Run:   runGamePlayOne,
}

var gameSeekCmd = &cobra.Command{
	Use:   "seek [username] [time_control]",
	Short: "Create a game seek for an account",
	Long:  "Create a game seek on chess.com. Time control format: '5+0', '10+5', etc.",
	Args:  cobra.ExactArgs(2),
	Run:   runGameSeek,
}

func init() {
	gameCmd.AddCommand(gamePlayCmd)
	gameCmd.AddCommand(gamePlayOneCmd)
	gameCmd.AddCommand(gameSeekCmd)
}

func loadGameStrategies(path string) (map[string]GameStrategy, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config struct {
		Strategies []GameStrategy `json:"strategies"`
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}

	strategyMap := make(map[string]GameStrategy)
	for _, s := range config.Strategies {
		strategyMap[s.Name] = s
	}

	return strategyMap, nil
}

func runGamePlay(cmd *cobra.Command, args []string) {
	db, err := loadDatabase("db.json")
	if err != nil {
		log.Fatalf("Failed to load database: %v", err)
	}

	strategies, err := loadGameStrategies("game_strategies.json")
	if err != nil {
		log.Fatalf("Failed to load game strategies: %v", err)
	}

	if len(db.Accounts) == 0 {
		logger.Println("No accounts found in db.json.")
		return
	}

	// Initialize engine
	// Note: Engine path should be configurable
	engine := NewChessEngine("stockfish", 4, 256, 3, 20)
	if err := engine.Start(); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()

	client := &http.Client{}

	for username, account := range db.Accounts {
		logger.Printf("Processing games for account: %s\n", username)

		strategy, ok := strategies["default"]
		if !ok {
			logger.Printf("Warning: default strategy not found, skipping %s\n", username)
			continue
		}

		err := PlayAllGamesForAccount(client, &account, &strategy, engine)
		if err != nil {
			logger.Printf("Error playing games for %s: %v\n", username, err)
		}
	}

	logger.Println("All accounts processed.")
}

func runGamePlayOne(cmd *cobra.Command, args []string) {
	username := args[0]

	db, err := loadDatabase("db.json")
	if err != nil {
		log.Fatalf("Failed to load database: %v", err)
	}

	account, ok := db.Accounts[username]
	if !ok {
		log.Fatalf("Account '%s' not found in db.json", username)
	}

	strategies, err := loadGameStrategies("game_strategies.json")
	if err != nil {
		log.Fatalf("Failed to load game strategies: %v", err)
	}

	strategy, ok := strategies["default"]
	if !ok {
		log.Fatalf("Default strategy not found in game_strategies.json")
	}

	// Initialize engine
	engine := NewChessEngine("stockfish", 4, 256, 3, 20)
	if err := engine.Start(); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()

	client := &http.Client{}

	err = PlayAllGamesForAccount(client, &account, &strategy, engine)
	if err != nil {
		log.Fatalf("Error playing games: %v", err)
	}

	logger.Printf("Finished playing games for %s\n", username)
}

func runGameSeek(cmd *cobra.Command, args []string) {
	username := args[0]
	timeControl := args[1]

	db, err := loadDatabase("db.json")
	if err != nil {
		log.Fatalf("Failed to load database: %v", err)
	}

	account, ok := db.Accounts[username]
	if !ok {
		log.Fatalf("Account '%s' not found in db.json", username)
	}

	client := &http.Client{}

	seek := GameSeekRequest{
		TimeControl: timeControl,
		Color:       "random",
		RatingMin:   0,
		RatingMax:   3000,
	}

	logger.Printf("Creating game seek for %s with time control %s...\n", username, timeControl)

	gameInfo, err := CreateGameSeek(client, account.Cookie, seek)
	if err != nil {
		log.Fatalf("Failed to create game seek: %v", err)
	}

	logger.Printf("Game seek created successfully!\n")
	if gameInfo != nil {
		logger.Printf("Game ID: %s\n", gameInfo.GameID)
		logger.Printf("Game URL: %s\n", gameInfo.GameURL)
	}
}
