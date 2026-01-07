package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
)

type SolutionPayload struct {
	LegacyPuzzleID  string     `json:"legacyPuzzleId"`
	Moves           []MoveType `json:"moves"`
	AttemptDuration string     `json:"attemptDuration"`
}

func getHeaders(cookie string) http.Header {
	headers := http.Header{}
	headers.Set("accept", "application/json, text/plain, */*")
	headers.Set("accept-language", "en-US,en;q=0.9")
	headers.Set("content-type", "application/json")
	headers.Set("sec-ch-ua", `"Not)A;Brand";v="8", "Chromium";v="138"`)
	headers.Set("sec-ch-ua-mobile", "?0")
	headers.Set("sec-ch-ua-platform", `"Linux"`)
	headers.Set("sec-fetch-dest", "empty")
	headers.Set("sec-fetch-mode", "cors")
	headers.Set("sec-fetch-site", "same-origin")
	headers.Set("cookie", cookie)
	headers.Set("Referer", "https://www.chess.com/puzzles/rated")
	return headers
}

func getNextPuzzle(client *http.Client, headers http.Header) (*GetRatedNextResponse, error) {
	req, err := http.NewRequest("POST", "https://www.chess.com/rpc/chesscom.puzzles.v1.PuzzleService/GetNextRated", strings.NewReader("{}"))
	if err != nil {
		return nil, err
	}
	req.Header = headers

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var puzzleResp GetRatedNextResponse
	err = json.Unmarshal(body, &puzzleResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal puzzle response: %w. Response body: %s", err, string(body))
	}

	if puzzleResp.UserPuzzle.Puzzle.LegacyPuzzleID == "" {
		return nil, fmt.Errorf("got empty puzzle ID. Response body: %s", string(body))
	}

	return &puzzleResp, nil
}

func submitSolution(client *http.Client, headers http.Header, puzzleResp *GetRatedNextResponse, strategy *Strategy) (*SubmitSolutionResponse, error) {
	var moves []MoveType
	for _, m := range puzzleResp.UserPuzzle.Puzzle.Moves {
		moves = append(moves, m.Move)
	}

	var attemptDuration float64
	switch strategy.TimeMode {
	case "hour":
		attemptDuration = 3600 + rand.Float64()*1800
	case "legit":
		attemptDuration = 15 + rand.Float64()*30
	case "zero":
		attemptDuration = 0.1 + rand.Float64()*0.3
	default:
		attemptDuration = 15.0
	}

	solution := SolutionPayload{
		LegacyPuzzleID:  puzzleResp.UserPuzzle.Puzzle.LegacyPuzzleID,
		Moves:           moves,
		AttemptDuration: fmt.Sprintf("%.3fs", attemptDuration),
	}

	payload, err := json.Marshal(solution)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://www.chess.com/rpc/chesscom.puzzles.v1.PuzzleService/SubmitRatedSolution", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header = headers

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var solutionResp SubmitSolutionResponse
	err = json.Unmarshal(body, &solutionResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal solution response: %w. Response body: %s", err, string(body))
	}
	solutionResp.AttemptDuration = attemptDuration

	return &solutionResp, nil
}

func getMembershipStatus(client *http.Client, cookie string) (*MembershipStatusResponse, error) {
	url := "https://www.chess.com/rpc/chesscom.payments.v1.ProductService/GetUserActiveMembership"
	req, err := http.NewRequest("POST", url, strings.NewReader("{}"))
	if err != nil {
		return nil, err
	}

	headers := getHeaders(cookie)
	req.Header = headers
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get membership status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var statusResp MembershipStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal membership status response: %w. Response body: %s", err, string(body))
	}

	return &statusResp, nil
}

func getTacticsStats(client *http.Client, cookie string) (*TacticsStatsResponse, error) {
	req, err := http.NewRequest("GET", "https://www.chess.com/callback/tactics/stats/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header = getHeaders(cookie)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var stats TacticsStatsResponse
	err = json.Unmarshal(body, &stats)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats response: %w. Response body: %s", err, string(body))
	}

	return &stats, nil
}

func getUserProfile(client *http.Client, cookie string) (*UserProfileResponse, error) {
	req, err := http.NewRequest("POST", "https://www.chess.com/rpc/chesscom.user_profile.v1.UserProfileService/GetProfileSettings", bytes.NewBuffer([]byte("{\"fieldMask\":\"\"}")))
	if err != nil {
		return nil, err
	}
	req.Header = getHeaders(cookie)
	req.Header.Set("Referer", "https://www.chess.com/settings/profile")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var profileResp UserProfileResponse
	err = json.Unmarshal(body, &profileResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user profile response: %w.", err)
	}

	return &profileResp, nil
}
