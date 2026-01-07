package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type AppConfig struct {
	DiscordWebhookURL     string `json:"discord_webhook_url"`
	MaxConcurrentAccounts int    `json:"max_concurrent_accounts"`
}

// Control when the account will stop submitting puzzles
type StopModeType string

const (
	StopModeRating  StopModeType = "stop_at_rating"            // Stop solving puzzles once a certain rating has been reached
	StopModePuzzles StopModeType = "stop_at_puzzles_completed" // Stop solving puzzles once a certain number has been completed
)

// These modes only affect the reported time to solve the puzzle sent to the API
// Delays between solving puzzles are handled with SubmitMode
type TimeModeType string

const (
	TimeModeLegit TimeModeType = "legit" // Make an effort to have a legitimate solve time
	TimeModeHour  TimeModeType = "hour"  // Submit an hour long time (to bring up the "time spent" statistic)
	TimeModeZero  TimeModeType = "zero"  // Submit zero as the solve tim
)

// The strategy for handling the actual delay between submitting puzzles
// The dashboard does not currently show the exact time a puzzle was submitted
// so this is for advanced stealth reasons
type SubmitModeType string

const (
	SubmitModeASAP  SubmitModeType = "asap"  // No delay, just send it off as soon as we can
	SubmitModeLegit SubmitModeType = "legit" // Sync up the time and delay
)

// This is the format for a strategy.
type Strategy struct {
	Name          string         `json:"name"`
	StopMode      StopModeType   `json:"stop_mode"`
	PuzzlesPerDay int            `json:"puzzles_per_day"`
	TargetRating  int            `json:"target_rating"`
	TimeMode      TimeModeType   `json:"time_mode"`
	SubmitMode    SubmitModeType `json:"submit_mode"`
}

type SolvedPuzzle struct {
	PuzzleID     string    `json:"puzzle_id"`
	Timestamp    time.Time `json:"timestamp"`
	RatingBefore int       `json:"rating_before"`
	RatingAfter  int       `json:"rating_after"`
	TimeTaken    float64   `json:"time_taken"`
	Success      bool      `json:"success"`
}

type Account struct {
	Username      string    `json:"username"`
	Cookie        string    `json:"cookie"`
	IsPremium     bool      `json:"is_premium"`
	StrategyName  string    `json:"strategy_name"`
	PremiumExpiry time.Time `json:"premium_expiry"`
	LastRun       time.Time `json:"last_run"`
	LastRating    int       `json:"last_rating"`
}

type Database struct {
	Accounts map[string]Account `json:"accounts"`
}

type StrategiesConfig struct {
	Strategies []Strategy `json:"strategies"`
}

func loadStrategies(path string) (map[string]Strategy, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			jsonString, err := json.MarshalIndent(&StrategiesConfig{
				Strategies: []Strategy{
					{
						Name:          "default",
						StopMode:      StopModePuzzles,
						PuzzlesPerDay: 3,
						TargetRating:  4000,
						TimeMode:      TimeModeLegit,
						SubmitMode:    SubmitModeLegit,
					},
				},
			}, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to create default strategies config: %w", err)
			}
			err = os.WriteFile(path, jsonString, 0644)
			if err != nil {
				return nil, fmt.Errorf("failed to write default strategies config: %w", err)
			}
			return loadStrategies(path)
		}
		return nil, err
	}

	var config StrategiesConfig
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}

	strategyMap := make(map[string]Strategy)
	for _, s := range config.Strategies {
		strategyMap[s.Name] = s
	}

	return strategyMap, nil
}

func loadAppConfig(path string) (*AppConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			jsonString, err := json.MarshalIndent(&AppConfig{
				DiscordWebhookURL:     "",
				MaxConcurrentAccounts: 5,
			}, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			err = os.WriteFile(path, jsonString, 0644)
			if err != nil {
				return nil, fmt.Errorf("failed to write default config: %w", err)
			}
			return &AppConfig{
				DiscordWebhookURL:     "",
				MaxConcurrentAccounts: 5,
			}, nil
		}
		return nil, err
	}

	var config AppConfig
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func loadDatabase(path string) (*Database, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			jsonString, err := json.MarshalIndent(&Database{
				Accounts: make(map[string]Account),
			}, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to create default database: %w", err)
			}
			err = os.WriteFile(path, jsonString, 0644)
			if err != nil {
				return nil, fmt.Errorf("failed to write default database: %w", err)
			}
			return &Database{
				Accounts: make(map[string]Account, 0),
			}, nil
		}
		return nil, err
	}

	var db Database
	err = json.Unmarshal(file, &db)
	if err != nil {
		return nil, err
	}

	return &db, nil
}

func saveDatabase(path string, db *Database) error {
	file, err := json.MarshalIndent(db, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, file, 0644)
}
