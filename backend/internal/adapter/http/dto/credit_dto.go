// DesignDocument/7_BackendDesign.md 7.5.3 Credit（残高・履歴）
package dto

type BalanceResponse struct {
	Balance int `json:"balance"`
}

type TransactionItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Amount      int    `json:"amount"`
	Status      string `json:"status"`
	Reason      string `json:"reason"`
	ExternalRef string `json:"externalRef"`
	CreatedAt   string `json:"createdAt"`
}

type TransactionsResponse struct {
	Items      []TransactionItem `json:"items"`
	NextCursor *string           `json:"nextCursor"`
}

// DesignDocument/6_AuthAccountCahrgingDesign.md 6.8 初回ボーナス

type BonusResponse struct {
	Granted bool `json:"granted"`
	Credits int  `json:"credits"`
	Balance int  `json:"balance"`
}
