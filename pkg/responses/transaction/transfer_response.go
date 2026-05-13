package transaction

import (
	"swapngo-backend/internal/models"
	"swapngo-backend/pkg/utils"
)

type TransferResponse struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	Recipient    string  `json:"recipient"`
	Token        string  `json:"token"`
	Amount       float64 `json:"amount"`
	SuiTxHash    *string `json:"sui_tx_hash"`
	ErrorMessage *string `json:"error_message"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func ToTransferResponse(t *models.Transfer) TransferResponse {
	r := TransferResponse{
		ID:        t.ID.String(),
		Status:    utils.NormalizeStatus(t.Status),
		Recipient: t.ToAddress,
		Token:     "MYRC",
		Amount:    t.Amount,
		CreatedAt: t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: t.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if t.TxHash != "" {
		r.SuiTxHash = &t.TxHash
	}
	return r
}

func ToTransferResponses(transfers []*models.Transfer) []TransferResponse {
	result := make([]TransferResponse, len(transfers))
	for i, t := range transfers {
		result[i] = ToTransferResponse(t)
	}
	return result
}
