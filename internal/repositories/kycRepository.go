package repositories

import (
	"swapngo-backend/internal/models"

	"gorm.io/gorm"
)

type KYCRepository interface {
	IBaseRepository[models.KYC]
}

type kycRepository struct {
	BaseRepository[models.KYC]
}

func NewKYCRepository(db *gorm.DB) KYCRepository {
	return &kycRepository{
		BaseRepository: *NewBaseRepository[models.KYC](db),
	}
}
