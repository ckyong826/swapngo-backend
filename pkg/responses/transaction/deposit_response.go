package transaction

import (
	"swapngo-backend/internal/models"
	"swapngo-backend/pkg/utils"
)

type DepositResponse struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	AmountMYR         float64 `json:"amount_myr"`
	MYRCMinted        float64 `json:"myrc_minted"`
	BillplzPaymentURL string  `json:"billplz_payment_url"`
	ErrorMessage      *string `json:"error_message"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

func ToDepositResponse(d *models.Deposit) DepositResponse {
	return DepositResponse{
		ID:                d.ID.String(),
		Status:            utils.NormalizeStatus(d.Status),
		AmountMYR:         d.AmountMYR,
		MYRCMinted:        d.AmountMYRC,
		BillplzPaymentURL: d.PaymentURL,
		CreatedAt:         d.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:         d.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func ToDepositResponses(deposits []*models.Deposit) []DepositResponse {
	result := make([]DepositResponse, len(deposits))
	for i, d := range deposits {
		result[i] = ToDepositResponse(d)
	}
	return result
}
