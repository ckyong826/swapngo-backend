package bizs

import (
	"context"
	"fmt"

	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	"swapngo-backend/internal/ws"
	kycReq "swapngo-backend/pkg/requests/kyc"
	"swapngo-backend/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KYCBiz interface {
	SubmitKYC(ctx context.Context, userID string, req *kycReq.SubmitKYCRequest) (any, error)
	GetKYCStatus(ctx context.Context, userID string) (any, error)
	// Admin methods
	ListPendingKYC(ctx context.Context) (any, error)
	ApproveKYC(ctx context.Context, kycID string) (any, error)
	RejectKYC(ctx context.Context, kycID string, req *kycReq.RejectKYCRequest) (any, error)
}

type kycBiz struct {
	db         *gorm.DB
	kycRepo    repositories.KYCRepository
	userRepo   repositories.UserRepository
	encryptKey []byte
	hub        *ws.Hub
}

func NewKYCBiz(db *gorm.DB, kycRepo repositories.KYCRepository, userRepo repositories.UserRepository, encryptKey []byte, hub *ws.Hub) KYCBiz {
	return &kycBiz{
		db:         db,
		kycRepo:    kycRepo,
		userRepo:   userRepo,
		encryptKey: encryptKey,
		hub:        hub,
	}
}

func (b *kycBiz) SubmitKYC(ctx context.Context, userID string, req *kycReq.SubmitKYCRequest) (any, error) {
	uid := uuid.Must(uuid.Parse(userID))

	existing, err := b.kycRepo.FirstBy(ctx, "user_id = ?", uid)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.Status == models.KYCStatusApproved {
			return nil, fmt.Errorf("KYC already approved")
		}
		if existing.Status == models.KYCStatusPending {
			return nil, fmt.Errorf("KYC already submitted and pending review")
		}
		// rejected — allow resubmission by deleting the old record
		if _, err := b.kycRepo.Delete(ctx, existing); err != nil {
			return nil, err
		}
	}

	encIC, err := utils.EncryptAES(b.encryptKey, req.ICNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt IC number")
	}
	encFront, err := utils.EncryptAES(b.encryptKey, req.ICFrontPhoto)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt IC front photo")
	}
	encBack, err := utils.EncryptAES(b.encryptKey, req.ICBackPhoto)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt IC back photo")
	}
	encSelfie, err := utils.EncryptAES(b.encryptKey, req.SelfiePhoto)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt selfie photo")
	}

	kyc := &models.KYC{
		UserID:       uid,
		FullName:     req.FullName,
		ICNumber:     encIC,
		ICFrontPhoto: encFront,
		ICBackPhoto:  encBack,
		SelfiePhoto:  encSelfie,
		Status:       models.KYCStatusPending,
	}

	created, err := b.kycRepo.Create(ctx, kyc)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"kyc_id": created.ID,
		"status": created.Status,
	}, nil
}

func (b *kycBiz) GetKYCStatus(ctx context.Context, userID string) (any, error) {
	uid := uuid.Must(uuid.Parse(userID))

	kyc, err := b.kycRepo.FirstBy(ctx, "user_id = ?", uid)
	if err != nil {
		return nil, err
	}
	if kyc == nil {
		return map[string]any{"status": "NOT_SUBMITTED"}, nil
	}

	return map[string]any{
		"kyc_id":     kyc.ID,
		"full_name":  kyc.FullName,
		"status":     kyc.Status,
		"remarks":    kyc.Remarks,
		"created_at": kyc.CreatedAt,
		"updated_at": kyc.UpdatedAt,
	}, nil
}

// ----- Admin methods -----

func (b *kycBiz) ListPendingKYC(ctx context.Context) (any, error) {
	kycs, err := b.kycRepo.FindBy(ctx, "status = ?", models.KYCStatusPending)
	if err != nil {
		return nil, err
	}

	type kycItem struct {
		KYCID        string `json:"kyc_id"`
		UserID       string `json:"user_id"`
		FullName     string `json:"full_name"`
		ICNumber     string `json:"ic_number"`
		ICFrontPhoto string `json:"ic_front_photo"` // decrypted base64
		ICBackPhoto  string `json:"ic_back_photo"`  // decrypted base64
		SelfiePhoto  string `json:"selfie_photo"`   // decrypted base64
		Status       string `json:"status"`
		CreatedAt    any    `json:"created_at"`
	}

	items := make([]kycItem, 0, len(kycs))
	for _, k := range kycs {
		icNum, _ := utils.DecryptAES(b.encryptKey, k.ICNumber)
		icFront, _ := utils.DecryptAES(b.encryptKey, k.ICFrontPhoto)
		icBack, _ := utils.DecryptAES(b.encryptKey, k.ICBackPhoto)
		selfie, _ := utils.DecryptAES(b.encryptKey, k.SelfiePhoto)

		items = append(items, kycItem{
			KYCID:        k.ID.String(),
			UserID:       k.UserID.String(),
			FullName:     k.FullName,
			ICNumber:     icNum,
			ICFrontPhoto: icFront,
			ICBackPhoto:  icBack,
			SelfiePhoto:  selfie,
			Status:       k.Status,
			CreatedAt:    k.CreatedAt,
		})
	}
	return items, nil
}

func (b *kycBiz) ApproveKYC(ctx context.Context, kycID string) (any, error) {
	id := uuid.Must(uuid.Parse(kycID))

	kyc, err := b.kycRepo.FindByID(ctx, id)
	if err != nil || kyc == nil {
		return nil, fmt.Errorf("KYC record not found")
	}
	if kyc.Status != models.KYCStatusPending {
		return nil, fmt.Errorf("KYC is not in PENDING state")
	}

	kyc.Status = models.KYCStatusApproved
	if _, err := b.kycRepo.Update(ctx, kyc); err != nil {
		return nil, fmt.Errorf("failed to update KYC status: %w", err)
	}

	user, err := b.userRepo.FindByID(ctx, kyc.UserID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("failed to fetch user for KYC")
	}
	user.KycStatus = models.KycApproved
	if _, err := b.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user KYC status: %w", err)
	}

	// Notify user via WebSocket
	b.hub.SendToUser(kyc.UserID.String(), ws.Event("KYC_APPROVED", map[string]any{
		"kyc_id": kyc.ID,
	}))

	return map[string]any{"kyc_id": kyc.ID, "status": kyc.Status}, nil
}

func (b *kycBiz) RejectKYC(ctx context.Context, kycID string, req *kycReq.RejectKYCRequest) (any, error) {
	id := uuid.Must(uuid.Parse(kycID))

	kyc, err := b.kycRepo.FindByID(ctx, id)
	if err != nil || kyc == nil {
		return nil, fmt.Errorf("KYC record not found")
	}
	if kyc.Status != models.KYCStatusPending {
		return nil, fmt.Errorf("KYC is not in PENDING state")
	}

	kyc.Status = models.KYCStatusRejected
	kyc.Remarks = req.Remarks
	if _, err := b.kycRepo.Update(ctx, kyc); err != nil {
		return nil, fmt.Errorf("failed to update KYC status: %w", err)
	}

	user, err := b.userRepo.FindByID(ctx, kyc.UserID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("failed to fetch user for KYC")
	}
	user.KycStatus = models.KycRejected
	if _, err := b.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user KYC status: %w", err)
	}

	// Notify user via WebSocket
	b.hub.SendToUser(kyc.UserID.String(), ws.Event("KYC_REJECTED", map[string]any{
		"kyc_id":  kyc.ID,
		"remarks": req.Remarks,
	}))

	return map[string]any{"kyc_id": kyc.ID, "status": kyc.Status, "remarks": kyc.Remarks}, nil
}
