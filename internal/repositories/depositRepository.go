package repositories

import (
	"context"
	"swapngo-backend/internal/models"
	"swapngo-backend/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DepositRepository interface {
	IBaseRepository[models.Deposit]
	LockByGatewayRef(ctx context.Context , refID string) (*models.Deposit, error)
}

type depositRepository struct {
	BaseRepository[models.Deposit]
	db *gorm.DB
}

func NewDepositRepository(db *gorm.DB) DepositRepository {
	return &depositRepository{
		BaseRepository: *NewBaseRepository[models.Deposit](db),
		db: db,
	}
}

/*
* Pessimistic Lock to prevent race condition
*/
func (r *depositRepository) LockByGatewayRef(ctx context.Context, refID string) (*models.Deposit, error) {
	var deposit models.Deposit
	
	err := database.GetDB(ctx, r.db).WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("gateway_ref_id = ?", refID).
		First(&deposit).Error
		
	if err != nil {
		return nil, err
	}
	return &deposit, nil
}