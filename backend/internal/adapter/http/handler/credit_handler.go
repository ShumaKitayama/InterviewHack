// DesignDocument/7_BackendDesign.md 7.5.3 Credit（残高・履歴）
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/interviewhack/backend/internal/adapter/http/dto"
	"github.com/interviewhack/backend/internal/adapter/http/presenter"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/credit"
)

type CreditHandler struct {
	creditRepo repository.CreditRepository
	bonusUC    *credit.BonusUsecase
}

func NewCreditHandler(creditRepo repository.CreditRepository, bonusUC *credit.BonusUsecase) *CreditHandler {
	return &CreditHandler{creditRepo: creditRepo, bonusUC: bonusUC}
}

// GetBalance handles GET /credits/balance
func (h *CreditHandler) GetBalance(c *gin.Context) {
	userID := c.GetString("user_id")
	balance, err := h.creditRepo.GetBalance(c.Request.Context(), userID)
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	presenter.RespondOK(c, dto.BalanceResponse{Balance: balance.Balance})
}

// GetTransactions handles GET /credits/transactions
func (h *CreditHandler) GetTransactions(c *gin.Context) {
	userID := c.GetString("user_id")
	limit := 50
	var cursor *string
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		cursor = &cursorStr
	}

	txs, nextCursor, err := h.creditRepo.ListTransactions(c.Request.Context(), userID, limit, cursor)
	if err != nil {
		presenter.RespondError(c, err)
		return
	}

	items := make([]dto.TransactionItem, 0, len(txs))
	for _, tx := range txs {
		items = append(items, dto.TransactionItem{
			ID:          tx.ID,
			Type:        tx.Type.String(),
			Amount:      tx.Amount,
			Status:      tx.Status.String(),
			Reason:      tx.Reason,
			ExternalRef: tx.ExternalRef,
			CreatedAt:   tx.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	presenter.RespondOK(c, dto.TransactionsResponse{
		Items:      items,
		NextCursor: nextCursor,
	})
}

// ClaimOnboardingBonus handles POST /credits/onboarding-bonus
// DesignDocument/6_AuthAccountCahrgingDesign.md 6.8
func (h *CreditHandler) ClaimOnboardingBonus(c *gin.Context) {
	userID := c.GetString("user_id")
	result, err := h.bonusUC.GrantOnboardingBonus(c.Request.Context(), userID)
	if err != nil {
		presenter.RespondError(c, err)
		return
	}

	presenter.RespondOK(c, dto.BonusResponse{
		Granted: result.Granted,
		Credits: result.Credits,
		Balance: result.Balance,
	})
}
