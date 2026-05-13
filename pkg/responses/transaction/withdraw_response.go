package transaction

import (
	"swapngo-backend/internal/models"
	"swapngo-backend/pkg/utils"
)

type WithdrawResponse struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	Token        string  `json:"token"`
	Amount       float64 `json:"amount"`
	SuiTxHash    *string `json:"sui_tx_hash"`
	ErrorMessage *string `json:"error_message"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func ToWithdrawResponse(w *models.Withdrawal) WithdrawResponse {
	r := WithdrawResponse{
		ID:        w.ID.String(),
		Status:    utils.NormalizeStatus(w.Status),
		Token:     "MYRC",
		Amount:    w.AmountMYRC,
		CreatedAt: w.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: w.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if w.TxHash != "" {
		r.SuiTxHash = &w.TxHash
	}
	return r
}

func ToWithdrawResponses(withdrawals []*models.Withdrawal) []WithdrawResponse {
	result := make([]WithdrawResponse, len(withdrawals))
	for i, w := range withdrawals {
		result[i] = ToWithdrawResponse(w)
	}
	return result
}
