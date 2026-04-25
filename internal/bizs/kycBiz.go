package bizs

import (
	"context"
	"fmt"
	"log"
	"time"

	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	kycReq "swapngo-backend/pkg/requests/kyc"
	"swapngo-backend/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KYCBiz interface {
	SubmitKYC(ctx context.Context, userID string, req *kycReq.SubmitKYCRequest) (any, error)
	GetKYCStatus(ctx context.Context, userID string) (any, error)
}

type kycBiz struct {
	db         *gorm.DB
	kycRepo    repositories.KYCRepository
	userRepo   repositories.UserRepository
	encryptKey []byte
}

func NewKYCBiz(db *gorm.DB, kycRepo repositories.KYCRepository, userRepo repositories.UserRepository, encryptKey []byte) KYCBiz {
	return &kycBiz{
		db:         db,
		kycRepo:    kycRepo,
		userRepo:   userRepo,
		encryptKey: encryptKey,
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

	kyc := &models.KYC{
		UserID:       uid,
		FullName:     req.FullName,
		ICNumber:     encIC,
		ICFrontPhoto: encFront,
		ICBackPhoto:  encBack,
		Status:       models.KYCStatusPending,
	}

	created, err := b.kycRepo.Create(ctx, kyc)
	if err != nil {
		return nil, err
	}

	go b.simulateApproval(created.ID, uid)

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
		"created_at": kyc.CreatedAt,
		"updated_at": kyc.UpdatedAt,
	}, nil
}

// simulateApproval auto-approves the KYC after 30 seconds.
func (b *kycBiz) simulateApproval(kycID uuid.UUID, userID uuid.UUID) {
	time.Sleep(30 * time.Second)

	ctx := context.Background()

	kyc, err := b.kycRepo.FindByID(ctx, kycID)
	if err != nil || kyc == nil || kyc.Status != models.KYCStatusPending {
		return
	}

	kyc.Status = models.KYCStatusApproved
	if _, err := b.kycRepo.Update(ctx, kyc); err != nil {
		log.Printf("KYC auto-approval: failed to update KYC %s: %v", kycID, err)
		return
	}

	user, err := b.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		log.Printf("KYC auto-approval: failed to fetch user %s: %v", userID, err)
		return
	}
	user.KycStatus = models.KycApproved
	if _, err := b.userRepo.Update(ctx, user); err != nil {
		log.Printf("KYC auto-approval: failed to update user KYC status for %s: %v", userID, err)
		return
	}

	log.Printf("KYC auto-approved for user %s", userID)
}
