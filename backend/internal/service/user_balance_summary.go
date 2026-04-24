package service

type UserBalanceRankingItem struct {
	UserID   int64   `json:"user_id"`
	Email    string  `json:"email"`
	Username string  `json:"username"`
	Status   string  `json:"status"`
	Balance  float64 `json:"balance"`
}

type UserBalanceSummary struct {
	TotalBalance float64                  `json:"total_balance"`
	UserCount    int64                    `json:"user_count"`
	Ranking      []UserBalanceRankingItem `json:"ranking"`
}
