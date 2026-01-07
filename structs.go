package main

import "time"

type UserProfileResponse struct {
	UserProfileSettings struct {
		ID        int    `json:"id"`
		UUID      string `json:"uuid"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Status    string `json:"status"`
		Avatar    struct {
			AvatarURL string `json:"avatarUrl"`
			HasImage  bool   `json:"hasImage"`
		} `json:"avatar"`
		Membership struct {
			Name  string `json:"name"`
			Code  string `json:"code"`
			Level int    `json:"level"`
		} `json:"membership"`
		Country struct {
			ID            int    `json:"id"`
			Name          string `json:"name"`
			NameLocalized string `json:"nameLocalized"`
			Code          string `json:"code"`
		} `json:"country"`
		Location string `json:"location"`
		Timezone string `json:"timezone"`
		Language struct {
			Locale string `json:"locale"`
		} `json:"language"`
		ContentLanguage        string `json:"contentLanguage"`
		CustomContentLanguages []struct {
			Locale string `json:"locale"`
		} `json:"customContentLanguages"`
		CreatedDate time.Time `json:"createdDate"`
	} `json:"userProfileSettings"`
}

type MoveType struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type GetRatedNextResponse struct {
	UserPuzzle struct {
		Puzzle struct {
			LegacyPuzzleID string `json:"legacyPuzzleId"`
			Pgn            string `json:"pgn"`
			Moves          []struct {
				Move               MoveType `json:"move"`
				MoveClassification string   `json:"moveClassification"`
				PuzzleHintSpeech   []struct {
					Locale string `json:"locale"`
					Speech struct {
						Sentences []struct {
							SentenceFragments []struct {
								Fragment string `json:"fragment"`
							} `json:"sentenceFragments"`
							AudioURLHash string `json:"audioUrlHash"`
						} `json:"sentences"`
					} `json:"speech"`
				} `json:"puzzleHintSpeech,omitempty"`
			} `json:"moves"`
			UserPosition string `json:"userPosition"`
			Goal         struct {
				TacticalGoal struct {
					WinMaterial struct {
					} `json:"winMaterial"`
					GoalType string `json:"goalType"`
				} `json:"tacticalGoal"`
			} `json:"goal"`
			Themes []struct {
				Type string `json:"type"`
			} `json:"themes"`
			Fen4 string `json:"fen4"`
		} `json:"puzzle"`
		PuzzleStats struct {
			AverageSolutionDuration string `json:"averageSolutionDuration"` // Tends to be zero?
			Ratings                 []struct {
				Rating     int    `json:"rating"`
				RatingType string `json:"ratingType"`
			} `json:"ratings"`
			PassedCount  string `json:"passedCount"`
			AttemptCount string `json:"attemptCount"`
		} `json:"puzzleStats"`
		UserStats struct {
			CurrentStreak      int  `json:"currentStreak"`
			HighestStreak      int  `json:"highestStreak"`
			IsNewHighestStreak bool `json:"isNewHighestStreak"`
		} `json:"userStats"`
		UserPuzzleProjection struct {
			TargetSolutionDuration string `json:"targetSolutionDuration"`
			RelativeDifficulty     string `json:"relativeDifficulty"`
		} `json:"userPuzzleProjection"`
	} `json:"userPuzzle"`
}

type TacticsStatsResponse struct {
	Rating             int  `json:"rating"`
	HighestRating      int  `json:"highestRating"`
	PercentCorrect     int  `json:"percentCorrect"`
	TodayAttempted     int  `json:"todayAttempted"`
	TotalAttempted     int  `json:"totalAttempted"`
	LastPositiveStreak int  `json:"lastPositiveStreak"`
	CurrentStreak      int  `json:"currentStreak"`
	HighestStreak      int  `json:"highestStreak"`
	IsNewHighestStreak bool `json:"isNewHighestStreak"`
	PuzzlePath         struct {
		Xp                        int `json:"xp"`
		BestStreak                int `json:"bestStreak"`
		TotalEasyPuzzles          int `json:"totalEasyPuzzles"`
		TotalHardPuzzles          int `json:"totalHardPuzzles"`
		TotalXHardPuzzles         int `json:"totalXHardPuzzles"`
		Tier                      int `json:"tier"`
		Level                     int `json:"level"`
		CurrentStreak             int `json:"currentStreak"`
		HardestSolvedPuzzleRating int `json:"hardestSolvedPuzzleRating"`
		PrestigeLevel             int `json:"prestigeLevel"`
	} `json:"puzzlePath"`
}

type SubmitSolutionResponse struct {
	SolutionResult string `json:"solutionResult"`
	UserRatings    []struct {
		Rating         int    `json:"rating"`
		RatingChange   int    `json:"ratingChange"`
		RatingType     string `json:"ratingType"`
		RatingUpdated  bool   `json:"ratingUpdated"`
		PreviousRating int    `json:"previousRating"`
	} `json:"userRatings"`
	PuzzleRatings []struct {
		Rating         int    `json:"rating"`
		RatingType     string `json:"ratingType"`
		PreviousRating int    `json:"previousRating"`
	} `json:"puzzleRatings"`
	AttemptDuration float64
}

type MembershipStatusResponse struct {
	MembershipLevel string `json:"membershipLevel"`
	Product         struct {
		Name        string `json:"name"`
		Sku         string `json:"sku"`
		PriceAmount struct {
			Amount       string `json:"amount"`
			CurrencyCode string `json:"currencyCode"`
		} `json:"priceAmount"`
		Subscription struct {
			Length int    `json:"length"`
			Unit   string `json:"unit"`
		} `json:"subscription"`
		Description    string `json:"description"`
		Image          string `json:"image"`
		ProductType    string `json:"productType"`
		MembershipCode string `json:"membershipCode"`
	} `json:"product"`
	MembershipName        string    `json:"membershipName"`
	WillExpire            bool      `json:"willExpire"`
	StatusCode            string    `json:"statusCode"`
	ExpiryDate            time.Time `json:"expiryDate"`
	MembershipDescription string    `json:"membershipDescription"`
	IsGift                bool      `json:"isGift"`
	IsFree                bool      `json:"isFree"`
	HasActiveBilling      bool      `json:"hasActiveBilling"`
}
