package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var logger = NewLogger()

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "chesshook2",
	Short: "A bot for solving chess.com puzzles.",
	Long:  `chesshook2 is a feature-rich bot for automatically solving chess.com puzzles for multiple accounts.`,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the puzzle solver for all accounts in db.json",
	Run:   runSolver,
}

var runOneCmd = &cobra.Command{
	Use:   "runOne",
	Short: "Run the puzzle solver for a single account",
	Args:  cobra.ExactArgs(1),
	Run:   runSolverForOne,
}

var loginCmd = &cobra.Command{
	Use:   "login [username] [password]",
	Short: "Login to a chess.com account to get a cookie (stub).",
	Long:  "This command is a placeholder for future functionality to log in to a chess.com account and retrieve an authentication cookie.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]
		// password := args[1]
		logger.Printf("Attempting to log in as %s...\n", username)
		logger.Printf("This functionality is not yet implemented.\n")
	},
}

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage accounts in db.json",
}

var addAccountCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new account to db.json by pasting a cURL command",
	Long:  "Adds a new account by parsing the cookie from an authenticated cURL request. The command will prompt you to paste the cURL command directly.",
	Run:   addAccount,
}

var changeStrategyCmd = &cobra.Command{
	Use:     "changestrategy",
	Aliases: []string{"cs"},
	Short:   "Change the strategy of account(s)",
	Long:    "Changes the strategy of one or more accounts. You can specify the strategy name and the accounts to change.",
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		strategyName := args[0]
		accounts := args[1:]
		db, err := loadDatabase("db.json")
		if err != nil {
			log.Fatalf("Failed to load database: %v", err)
		}
		if len(db.Accounts) == 0 {
			logger.Println("No accounts found in db.json.")
			return
		}
		strategies, err := loadStrategies("strategies.json")
		if err != nil {
			log.Fatalf("Failed to load strategies: %v", err)
		}
		if _, ok := strategies[strategyName]; !ok {
			log.Fatalf("Strategy '%s' not found in strategies.json.", strategyName)
		}
		for _, accountName := range accounts {
			account, ok := db.Accounts[accountName]
			if !ok {
				log.Fatalf("Account '%s' not found in db.json.", accountName)
			}
			account.StrategyName = strategyName
			db.Accounts[accountName] = account
			logger.Printf("Changed strategy for account '%s' to '%s'.\n", accountName, strategyName)
		}

		if err := saveDatabase("db.json", db); err != nil {
			log.Fatalf("Failed to save database: %v", err)
		}
	},
}

var listAccountsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts in db.json",
	Long:  "Lists all accounts stored in db.json, showing their usernames and membership status.",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := loadDatabase("db.json")
		if err != nil {
			log.Fatalf("Failed to load database: %v", err)
		}

		if len(db.Accounts) == 0 {
			logger.Println("No accounts found.")
			return
		}

		logger.Println("Accounts in db.json:")
		for _, account := range db.Accounts {
			status := "Free"
			if account.IsPremium {
				status = "Premium"
			}
			logger.Printf("- %s (%s)\n", account.Username, status)
		}
	},
}

var pruneAccountsCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune accounts that are no longer valid",
	Long:  "Prunes accounts that are no longer valid (empty token)",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := loadDatabase("db.json")
		if err != nil {
			log.Fatalf("Failed to load database: %v", err)
		}

		if len(db.Accounts) == 0 {
			logger.Println("No accounts found in db.json.")
			return
		}

		for username, account := range db.Accounts {
			if account.Cookie == "" {
				logger.Printf("Removing account '%s' with empty cookie.\n", username)
				delete(db.Accounts, username)
			} else {
				logger.Printf("Keeping account '%s'.\n", username)
			}
		}

		if err := saveDatabase("db.json", db); err != nil {
			log.Fatalf("Failed to save database: %v", err)
		}

		logger.Println("Pruning complete. Remaining accounts:")
		for _, account := range db.Accounts {
			logger.Printf("- %s\n", account.Username)
		}
	},
}

var refreshAccountsCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh account data",
	Long:  "Refresh account data, such as premium status, rating, and token validity",
	Run:   refreshAccounts,
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(runOneCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(accountsCmd)
	rootCmd.AddCommand(changeStrategyCmd)
	rootCmd.AddCommand(gameCmd)
	rootCmd.AddCommand(userscriptCmd)
	rootCmd.AddCommand(serveCmd)
	accountsCmd.AddCommand(addAccountCmd)
	accountsCmd.AddCommand(listAccountsCmd)
	accountsCmd.AddCommand(refreshAccountsCmd)
	accountsCmd.AddCommand(pruneAccountsCmd)
}

func parseCookieFromCurl(curlCmd string) (string, error) {
	// Regex to find cookie from -b or --cookie flag
	reCookieFlag := regexp.MustCompile(`(?:-b|--cookie)\s+'([^']+)'`)
	matches := reCookieFlag.FindStringSubmatch(curlCmd)
	if len(matches) > 1 {
		return matches[1], nil
	}

	// Regex to find cookie from -H 'cookie: ...' header
	reCookieHeader := regexp.MustCompile(`-H\s+'cookie:\s*([^']*)'`)
	matches = reCookieHeader.FindStringSubmatch(curlCmd)
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", errors.New("could not find cookie in cURL command")
}

func addAccount(cmd *cobra.Command, args []string) {
	logger.Println("Please paste the authenticated cURL command from your browser's devtools.")
	logger.Println("Press Ctrl+D (or Ctrl+Z on Windows) when you are finished:")

	curlBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to read cURL command from input: %v", err)
	}
	curlCmd := string(curlBytes)

	cookie, err := parseCookieFromCurl(curlCmd)
	if err != nil {
		log.Fatalf("Error parsing cURL command: %v", err)
	}

	db, err := loadDatabase("db.json")
	if err != nil {
		log.Fatalf("Failed to load database: %v", err)
	}

	newAccount := Account{
		Username:      "",
		Cookie:        cookie,
		IsPremium:     false,
		PremiumExpiry: time.Time{},
		StrategyName:  "default",
		LastRun:       time.Time{},
		LastRating:    0,
	}

	client := &http.Client{}
	refreshAccount(client, &newAccount)

	if _, ok := db.Accounts[newAccount.Username]; ok {
		logger.Printf("Account with username '%s' already exists.\n", newAccount.Username)
		os.Exit(1)
	}

	db.Accounts[newAccount.Username] = newAccount

	if err := saveDatabase("db.json", db); err != nil {
		log.Fatalf("Failed to save database: %v", err)
	}

	logger.Printf("\nSuccessfully added account: %s\n", newAccount.Username)
	if newAccount.IsPremium {
		logger.Println("This account has a premium membership.")
	} else {
		logger.Println("This account has a free membership.")
	}
}

type ProcessResult struct {
	AccountUsername string
	PuzzlesSolved   int
	Strategy        *Strategy
	Error           error
}

func runSolver(cmd *cobra.Command, args []string) {
	appConfig, err := loadAppConfig("config.json")
	if err != nil {
		log.Fatalf("failed to load app config: %v", err)
	}

	db, err := loadDatabase("db.json")
	if err != nil {
		log.Fatalf("failed to load database: %v", err)
	}

	strategies, err := loadStrategies("strategies.json")
	if err != nil {
		log.Fatalf("failed to load strategies: %v", err)
	}

	var wg sync.WaitGroup
	client := &http.Client{}

	var accountNames []string
	for _, account := range db.Accounts {
		accountNames = append(accountNames, account.Username)
	}

	startEmbed := Embed{
		Title:       "chesshook2 run starting...",
		Description: "Starting processing for the following accounts:",
		Color:       3447003, // Blue
		Fields: []EmbedField{
			{Name: "Accounts", Value: strings.Join(accountNames, "\n"), Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	SendWebhook(appConfig.DiscordWebhookURL, WebhookPayload{Embeds: []Embed{startEmbed}})

	resultsChan := make(chan ProcessResult, len(db.Accounts))

	limit := appConfig.MaxConcurrentAccounts
	if limit <= 0 {
		limit = 1
	}
	semaphore := make(chan struct{}, limit)

	for _, account := range db.Accounts {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(account *Account) {
			defer func() {
				<-semaphore
				wg.Done()
			}()
			processAccount(client, account, appConfig.DiscordWebhookURL, strategies, resultsChan)
		}(&account)
	}

	wg.Wait()
	close(resultsChan)

	var results []ProcessResult
	for result := range resultsChan {
		results = append(results, result)
	}

	err = saveDatabase("db.json", db)
	if err != nil {
		log.Fatalf("failed to save database: %v", err)
	}

	logger.Printf("All accounts processed.\n")

	var successfulAccounts, cooldownAccounts, errorAccounts []string

	for _, result := range results {
		if result.Error != nil {
			if strings.Contains(result.Error.Error(), "cooldown") {
				cooldownAccounts = append(cooldownAccounts, result.AccountUsername)
			} else {
				errorAccounts = append(errorAccounts, result.AccountUsername)
			}
		} else {
			if result.Strategy != nil {
				successfulAccounts = append(successfulAccounts, fmt.Sprintf("%s (%d/%d puzzles)", result.AccountUsername, result.PuzzlesSolved, result.Strategy.PuzzlesPerDay))
			} else {
				successfulAccounts = append(successfulAccounts, fmt.Sprintf("%s (%d puzzles)", result.AccountUsername, result.PuzzlesSolved))
			}
		}
	}

	endEmbed := Embed{
		Title:       "chesshook2 execution summary",
		Description: "Summary of the execution for all accounts.",
		Color:       3447003,
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields:      []EmbedField{},
	}

	if len(successfulAccounts) > 0 {
		endEmbed.Fields = append(endEmbed.Fields, EmbedField{Name: "✅ Success", Value: strings.Join(successfulAccounts, "\n"), Inline: false})
	}
	if len(cooldownAccounts) > 0 {
		endEmbed.Fields = append(endEmbed.Fields, EmbedField{Name: "⚠️ Cooldown", Value: strings.Join(cooldownAccounts, "\n"), Inline: false})
	}
	if len(errorAccounts) > 0 {
		endEmbed.Fields = append(endEmbed.Fields, EmbedField{Name: "❌ Errors", Value: strings.Join(errorAccounts, "\n"), Inline: false})
	}
	SendWebhook(appConfig.DiscordWebhookURL, WebhookPayload{Embeds: []Embed{endEmbed}})
}

func runSolverForOne(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		log.Fatal("You must specify exactly one account username.")
	}

	username := args[0]

	appConfig, err := loadAppConfig("config.json")
	if err != nil {
		log.Fatalf("failed to load app config: %v", err)
	}

	db, err := loadDatabase("db.json")
	if err != nil {
		log.Fatalf("failed to load database: %v", err)
	}

	strategies, err := loadStrategies("strategies.json")
	if err != nil {
		log.Fatalf("failed to load strategies: %v", err)
	}

	if _, ok := db.Accounts[username]; !ok {
		log.Fatalf("Account with username '%s' not found in db.json.", username)
	}
	account := db.Accounts[username]

	client := &http.Client{}

	startEmbed := Embed{
		Title:       "chesshook2 runOne starting...",
		Description: fmt.Sprintf("Starting processing for account: %s", account.Username),
		Color:       3447003,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	SendWebhook(appConfig.DiscordWebhookURL, WebhookPayload{Embeds: []Embed{startEmbed}})

	resultsChan := make(chan ProcessResult, 1)

	processAccount(client, &account, appConfig.DiscordWebhookURL, strategies, resultsChan)

	result := <-resultsChan
	close(resultsChan)

	err = saveDatabase("db.json", db)
	if err != nil {
		log.Fatalf("failed to save database: %v", err)
	}

	logger.Printf("Account %s processed.\n", account.Username)

	endEmbed := Embed{
		Title:       "chesshook2 execution summary",
		Description: fmt.Sprintf("Summary of the execution for account %s.", account.Username),
		Color:       3447003,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if result.Error != nil {
		endEmbed.Fields = append(endEmbed.Fields, EmbedField{Name: "❌ Error", Value: result.Error.Error(), Inline: false})
	} else {
		endEmbed.Fields = append(endEmbed.Fields, EmbedField{Name: "✅ Success", Value: fmt.Sprintf("%s solved %d puzzles with strategy '%s'", result.AccountUsername, result.PuzzlesSolved, result.Strategy.Name), Inline: false})
	}
	SendWebhook(appConfig.DiscordWebhookURL, WebhookPayload{Embeds: []Embed{endEmbed}})
}

func refreshAccounts(cmd *cobra.Command, args []string) {
	db, err := loadDatabase("db.json")
	if err != nil {
		log.Fatalf("failed to load database: %v", err)
	}

	client := &http.Client{}

	if len(db.Accounts) == 0 {
		logger.Println("No accounts found in db.json. Please add accounts first using the 'add' command.")
		return
	}

	logger.Printf("Found %d accounts. Refreshing membership status and tactics stats...\n", len(db.Accounts))
	for username, account := range db.Accounts {
		err = refreshAccount(client, &account)
		if err != nil {
			logger.Printf("%s\n", err.Error())
		}
		db.Accounts[username] = account
	}

	if err := saveDatabase("db.json", db); err != nil {
		log.Fatalf("failed to save database: %v", err)
	}

	logger.Println("All accounts refreshed successfully.")
}

func refreshAccount(client *http.Client, account *Account) error {
	membershipStatus, err := getMembershipStatus(client, account.Cookie)
	if err != nil {
		if strings.Contains(err.Error(), "403 Forbidden") { // TODO: handle this better
			account.Cookie = ""
			logger.Printf("Got 403, concluding that cookie is invalid for %s. Invalidating it. Consider running `accounts prune`.\n", account.Username)
		}
		return fmt.Errorf("failed to get membership status for account %s: %w", account.Username, err)
	}

	account.IsPremium = !membershipStatus.IsFree
	account.PremiumExpiry = membershipStatus.ExpiryDate

	logger.Printf("Account %s membership refreshed. Got: %s (expiry: %s)\n", account.Username, membershipStatus.MembershipLevel, membershipStatus.ExpiryDate.Format(time.RFC822))

	accountProfile, err := getUserProfile(client, account.Cookie)
	if err != nil {
		return fmt.Errorf("failed to get user profile for account %s: %w", account.Username, err)
	}

	account.Username = accountProfile.UserProfileSettings.Username

	logger.Printf("Account %s profile refreshed. Username: %s\n", account.Username, account.Username)

	accountData, err := getTacticsStats(client, account.Cookie)
	if err != nil {
		return fmt.Errorf("failed to get tactics stats for account %s: %w", account.Username, err)
	}

	account.LastRating = accountData.Rating
	logger.Printf("Account %s tactics stats refreshed. Rating: %d\n", account.Username, account.LastRating)

	return nil
}

func processAccount(client *http.Client, account *Account, webhookURL string, strategies map[string]Strategy, resultsChan chan<- ProcessResult) {
	strategy, ok := strategies[account.StrategyName]
	if !ok {
		err := fmt.Errorf("strategy not found: %s", account.StrategyName)
		logger.AddLine(account.Username, fmt.Sprintf("[%s] %v", account.Username, err))
		resultsChan <- ProcessResult{AccountUsername: account.Username, Error: err}
		return
	}

	logger.AddLine(account.Username, fmt.Sprintf("[%s] Starting with strategy '%s'", account.Username, strategy.Name))
	initialStats, err := getTacticsStats(client, account.Cookie)
	if err != nil {
		logger.AddLine(account.Username, fmt.Sprintf("[%s] Error getting initial stats: %v", account.Username, err))
	}

	var finalError error

	solvedCount := 0
	if !account.LastRun.IsZero() && time.Since(account.LastRun) < 24*time.Hour && !account.IsPremium {
		finalError = fmt.Errorf("on cooldown until %s", account.LastRun.Add(24*time.Hour).Format(time.RFC822))
	} else {
		shouldStop := false
		lastRating := 0
		for !shouldStop {
			switch strategy.StopMode {
			case StopModePuzzles:
				logger.AddLine(account.Username, fmt.Sprintf("[%s] %s Solving puzzle %d/%d...", account.Username, ProgressBarUtil(solvedCount+1, strategy.PuzzlesPerDay), solvedCount+1, strategy.PuzzlesPerDay))
			case StopModeRating:
				logger.AddLine(account.Username, fmt.Sprintf("[%s] %s Solving puzzle rating: %d/%d...", account.Username, ProgressBarUtil(lastRating, strategy.TargetRating), lastRating, strategy.TargetRating))
			}
			solvedPuzzle, err := solvePuzzleForAccount(client, account, &strategy)
			if err != nil {
				finalError = err
				logger.AddLine(account.Username, fmt.Sprintf("[%s] Error solving puzzle: %v", account.Username, err))
				break
			}
			solvedCount++
			lastRating = solvedPuzzle.RatingAfter

			switch strategy.StopMode {
			case StopModePuzzles:
				shouldStop = strategy.PuzzlesPerDay > 0 && solvedCount >= strategy.PuzzlesPerDay
			case StopModeRating:
				shouldStop = strategy.TargetRating > 0 && lastRating >= strategy.TargetRating
			}

			if !shouldStop && strategy.SubmitMode != SubmitModeASAP {
				delay := time.Duration(solvedPuzzle.TimeTaken) * time.Second
				startTime := time.Now()
				go func() {
					timeLeft := time.Until(startTime.Add(delay)).Round(time.Second)
					for timeLeft > 0 {
						logger.AddLine(account.Username, fmt.Sprintf("[%s] %s %d/%d Waiting for %s...", account.Username, ProgressBarUtil(solvedCount, strategy.PuzzlesPerDay), solvedCount, strategy.PuzzlesPerDay, timeLeft.Round(time.Second)))
						time.Sleep(time.Second)
						timeLeft = time.Until(startTime.Add(delay))
					}
				}()
				time.Sleep(delay)
			}
		}
		if strategy.PuzzlesPerDay > 0 && finalError == nil {
			account.LastRun = time.Now()
		}
	}

	finalStats, err := getTacticsStats(client, account.Cookie)
	if err != nil {
		logger.AddLine(account.Username, fmt.Sprintf("[%s] Error getting final stats: %v", account.Username, err))
	}

	embed := buildCompletionEmbed(account, initialStats, finalStats, &strategy, finalError, solvedCount)
	SendWebhook(webhookURL, WebhookPayload{Embeds: []Embed{embed}})

	logger.RemoveLine(account.Username)
	if finalError != nil {
		logger.Printf("[%s] Finished with error: %v\n", account.Username, finalError)
	} else {
		logger.Printf("[%s] Finished successfully after solving %d puzzles.\n", account.Username, solvedCount)
	}

	resultsChan <- ProcessResult{
		AccountUsername: account.Username,
		PuzzlesSolved:   solvedCount,
		Strategy:        &strategy,
		Error:           finalError,
	}
}

func solvePuzzleForAccount(client *http.Client, account *Account, strategy *Strategy) (*SolvedPuzzle, error) {
	statsBefore, err := getTacticsStats(client, account.Cookie)
	if err != nil {
		logger.AddLine(account.Username, fmt.Sprintf("[%s] could not get stats before puzzle: %v", account.Username, err))
	}

	headers := getHeaders(account.Cookie)
	logger.AddLine(account.Username, fmt.Sprintf("[%s] Fetching next puzzle...", account.Username))
	puzzleResp, err := getNextPuzzle(client, headers)
	if err != nil {
		return nil, err
	}

	logger.AddLine(account.Username, fmt.Sprintf("[%s] Submitting solution for puzzle %s...", account.Username, puzzleResp.UserPuzzle.Puzzle.LegacyPuzzleID))
	solutionResp, err := submitSolution(client, headers, puzzleResp, strategy)
	if err != nil {
		return nil, err
	}

	newRating := 0
	if len(solutionResp.UserRatings) > 0 {
		newRating = solutionResp.UserRatings[0].Rating
	}

	ratingBefore := 0
	if statsBefore != nil {
		ratingBefore = statsBefore.Rating
	}

	return &SolvedPuzzle{
		PuzzleID:     puzzleResp.UserPuzzle.Puzzle.LegacyPuzzleID,
		Timestamp:    time.Now(),
		TimeTaken:    solutionResp.AttemptDuration,
		RatingBefore: ratingBefore,
		RatingAfter:  newRating,
		Success:      true,
	}, nil
}

func buildCompletionEmbed(account *Account, initialStats, finalStats *TacticsStatsResponse, strategy *Strategy, err error, puzzlesSolvedThisRun int) Embed {
	var statusDesc string
	var color int
	if err != nil {
		statusDesc = fmt.Sprintf("Completed with issue: %v", err)
		if strings.Contains(err.Error(), "cooldown") {
			color = 16776960 // Yellow
		} else {
			color = 15158332 // Red
		}
	} else {
		statusDesc = "Completed successfully."
		color = 3066993 // Green
	}

	initialRating := "N/A"
	if initialStats != nil {
		initialRating = fmt.Sprintf("%d", initialStats.Rating)
	}
	finalRating := "N/A"
	if finalStats != nil {
		finalRating = fmt.Sprintf("%d", finalStats.Rating)
	}

	puzzlesAttempted := "0"
	if strategy != nil {
		puzzlesAttempted = fmt.Sprintf("%d/%d", puzzlesSolvedThisRun, strategy.PuzzlesPerDay)
	}

	return Embed{
		Title:       fmt.Sprintf("Report for %s", account.Username),
		Description: statusDesc,
		Color:       color,
		Fields: []EmbedField{
			{Name: "Strategy", Value: account.StrategyName, Inline: true},
			{Name: "Puzzles Attempted", Value: puzzlesAttempted, Inline: true},
			{Name: "Initial Rating", Value: initialRating, Inline: true},
			{Name: "Final Rating", Value: finalRating, Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}
