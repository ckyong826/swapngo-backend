package transaction

import (
	"swapngo-backend/internal/models"
	"swapngo-backend/pkg/utils"
)

type SwapResponse struct {
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	FromToken      string  `json:"from_token"`
	ToToken        string  `json:"to_token"`
	Amount         float64 `json:"amount"`
	ReceivedAmount float64 `json:"received_amount"`
	SuiTxHash      *string `json:"sui_tx_hash"`
	ErrorMessage   *string `json:"error_message"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func ToSwapResponse(s *models.Swap) SwapResponse {
	r := SwapResponse{
		ID:             s.ID.String(),
		Status:         utils.NormalizeStatus(s.Status),
		FromToken:      string(s.FromToken),
		ToToken:        string(s.ToToken),
		Amount:         s.FromAmount,
		ReceivedAmount: s.ActualToAmount,
		CreatedAt:      s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      s.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if s.TxHash != "" {
		r.SuiTxHash = &s.TxHash
	}
	return r
}

func ToSwapResponses(swaps []*models.Swap) []SwapResponse {
	result := make([]SwapResponse, len(swaps))
	for i, s := range swaps {
		result[i] = ToSwapResponse(s)
	}
	return result
}
