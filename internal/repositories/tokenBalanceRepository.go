package repositories

import (
	"context"
	"errors"

	"swapngo-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TokenBalanceRepository interface {
	// GetBalance returns the current balance for a given account + token.
	// Returns 0 (not an error) if no row exists yet.
	GetBalance(ctx context.Context, accountID uuid.UUID, token models.TokenType) (float64, error)

	// GetAllForAccount returns all token balances for an account.
	GetAllForAccount(ctx context.Context, accountID uuid.UUID) ([]*models.TokenBalance, error)

	// AddBalance credits amount to the balance (creates row if missing).
	AddBalance(ctx context.Context, accountID uuid.UUID, token models.TokenType, amount float64) error

	// DeductBalance debits amount from the balance.
	// Returns an error if the resulting balance would go negative.
	DeductBalance(ctx context.Context, accountID uuid.UUID, token models.TokenType, amount float64) error
}

type tokenBalanceRepository struct {
	db *gorm.DB
}

func NewTokenBalanceRepository(db *gorm.DB) TokenBalanceRepository {
	return &tokenBalanceRepository{db: db}
}

func (r *tokenBalanceRepository) GetBalance(ctx context.Context, accountID uuid.UUID, token models.TokenType) (float64, error) {
	var tb models.TokenBalance
	err := r.db.WithContext(ctx).
		Where("account_id = ? AND token = ?", accountID, token).
		First(&tb).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return tb.Balance, nil
}

func (r *tokenBalanceRepository) GetAllForAccount(ctx context.Context, accountID uuid.UUID) ([]*models.TokenBalance, error) {
	var balances []*models.TokenBalance
	err := r.db.WithContext(ctx).
		Where("account_id = ?", accountID).
		Find(&balances).Error
	return balances, err
}

func (r *tokenBalanceRepository) AddBalance(ctx context.Context, accountID uuid.UUID, token models.TokenType, amount float64) error {
	// Upsert: insert or update balance atomically
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tb models.TokenBalance
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_id = ? AND token = ?", accountID, token).
			First(&tb).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// First time this token is credited — create the row
			tb = models.TokenBalance{
				AccountID: accountID,
				Token:     token,
				Balance:   amount,
			}
			return tx.Create(&tb).Error
		}
		if err != nil {
			return err
		}

		tb.Balance += amount
		return tx.Save(&tb).Error
	})
}

func (r *tokenBalanceRepository) DeductBalance(ctx context.Context, accountID uuid.UUID, token models.TokenType, amount float64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tb models.TokenBalance
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_id = ? AND token = ?", accountID, token).
			First(&tb).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("insufficient balance: no holding found")
		}
		if err != nil {
			return err
		}
		if tb.Balance < amount {
			return errors.New("insufficient balance")
		}

		tb.Balance -= amount
		return tx.Save(&tb).Error
	})
}
