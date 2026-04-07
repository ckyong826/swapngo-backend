package repositories

import (
	"context"

	"swapngo-backend/internal/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	IBaseRepository[models.User]
	FindByPhoneNumber(ctx context.Context, phoneNumber string) (models.User, error)
	FindByEmail(ctx context.Context, email string) (models.User, error)
}

type userRepository struct {
	BaseRepository[models.User]
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		BaseRepository: *NewBaseRepository[models.User](db),
		db: db,
	}
}

func (r *userRepository) FindByPhoneNumber(ctx context.Context, phoneNumber string) (models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("phone_number = ?", phoneNumber).First(&user).Error
	return user, err
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	return user, err
}
