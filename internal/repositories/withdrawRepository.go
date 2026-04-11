package repositories

import (
	"context"
	"swapngo-backend/internal/models"
	"swapngo-backend/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WithdrawRepository interface {
	IBaseRepository[models.Withdrawal]
	LockByGatewayRef(ctx context.Context , refID string) (*models.Withdrawal, error)
}

type withdrawRepository struct {
	BaseRepository[models.Withdrawal]
	db *gorm.DB
}

func NewWithdrawRepository(db *gorm.DB) WithdrawRepository {
	return &withdrawRepository{
		BaseRepository: *NewBaseRepository[models.Withdrawal](db),
		db: db,
	}
}

func (r *withdrawRepository) LockByGatewayRef(ctx context.Context, refID string) (*models.Withdrawal, error) {
	var withdrawal models.Withdrawal
	
	err := database.GetDB(ctx, r.db).WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("gateway_ref_id = ?", refID).
		First(&withdrawal).Error
		
	if err != nil {
		return nil, err
	}
	return &withdrawal, nil
}