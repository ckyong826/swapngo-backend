package bizs

import (
	"context"
	"swapngo-backend/internal/services"
	"swapngo-backend/pkg/database"
	"swapngo-backend/pkg/requests"

	"gorm.io/gorm"
)

type AuthBiz interface {
	Register(ctx context.Context, req *requests.RegisterRequest) (any, error)
}

type authBiz struct {
	db             *gorm.DB // used for transaction
	userService services.UserService
	accountService services.AccountService
	walletService services.WalletService
}

func NewAuthBiz(db *gorm.DB, userService services.UserService, accountService services.AccountService, walletService services.WalletService) AuthBiz {
	return &authBiz{
		db: db,
		userService: userService,
		accountService: accountService,
		walletService: walletService,
	}
}

func (s *authBiz) Register(ctx context.Context, req *requests.RegisterRequest) (any, error) {
	// Run in tx to ensure atomicity
	return nil, database.RunInTx(s.db, ctx, func(txCtx context.Context) error {
		
		// 1. Register User
		user, err := s.userService.RegisterUser(txCtx, req)
		if err != nil {
			return err 
		}

		// 2. Create Account
		account, err := s.accountService.CreateAccount(txCtx, req, user.ID)
		if err != nil {
			return err 
		}

		// 3. Create Wallets
		err = s.walletService.GenerateWalletsForAccount(txCtx, req, account.ID)
		if err != nil {
			return err 
		}

		return nil 
	})
}