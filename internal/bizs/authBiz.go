package bizs

import (
	"context"
	"strings"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/services"
	"swapngo-backend/pkg/database"
	authReq "swapngo-backend/pkg/requests/auth"
	authRes "swapngo-backend/pkg/responses/auth"
	userRes "swapngo-backend/pkg/responses/user"
	"swapngo-backend/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthBiz interface {
	Register(ctx context.Context, req *authReq.RegisterRequest) (authRes.LoginResponse, error)
	Login(ctx context.Context, req *authReq.LoginRequest) (authRes.LoginResponse, error)
}

type authBiz struct {
	db             *gorm.DB // used for transaction
	userService    services.UserService
	accountService services.AccountService
	walletService  services.WalletService
}

func NewAuthBiz(db *gorm.DB, userService services.UserService, accountService services.AccountService, walletService services.WalletService) AuthBiz {
	return &authBiz{
		db:             db,
		userService:    userService,
		accountService: accountService,
		walletService:  walletService,
	}
}

func (s *authBiz) Register(ctx context.Context, req *authReq.RegisterRequest) (authRes.LoginResponse, error) {
	var responseData authRes.LoginResponse

	// Auto-fill optional fields from email
	if req.Username == "" {
		parts := strings.SplitN(req.Email, "@", 2)
		req.Username = parts[0] + "-" + uuid.New().String()[:6]
	}
	if req.PhoneNumber == "" {
		req.PhoneNumber = "auto-" + uuid.New().String()[:12]
	}
	if req.Pin == "" {
		req.Pin = "0000"
	}
	if req.AccountName == "" {
		req.AccountName = req.Username
	}
	if req.CustodyType == "" {
		req.CustodyType = models.AccountServerManaged
	}

	// Run in tx to ensure atomicity
	err := database.RunInTx(s.db, ctx, func(txCtx context.Context) error {

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
		err = s.walletService.GenerateWalletsForAccount(txCtx, account.ID)
		if err != nil {
			return err
		}

		// 4. Generate JWT tokens for auto-login
		responseData, err = generateToken(user)
		if err != nil {
			return err
		}

		return nil
	})

	// If transaction failed, responseData will be nil, and err will contain the reason
	return responseData, err
}

func (s *authBiz) Login(ctx context.Context, req *authReq.LoginRequest) (authRes.LoginResponse, error) {
	// 1. Verify credentials
	user, err := s.userService.VerifyUser(ctx, req)
	if err != nil {
		return authRes.LoginResponse{}, err 
	}

	return generateToken(user)
}

/*
* Generate JWT tokens for auto-login
* @param user: User model
* @return authRes.LoginResponse: Login response
*/
func generateToken(user models.User) (authRes.LoginResponse, error) {
	userIDStr := user.ID.String()
	accessToken, err := utils.GenerateAccessToken(userIDStr)
	if err != nil {
		return authRes.LoginResponse{}, err
	}

	refreshToken, err := utils.GenerateRefreshToken(userIDStr)
	if err != nil {
		return authRes.LoginResponse{}, err
	}

	return authRes.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: userRes.UserResponse{
			ID:          user.ID.String(),
			Username:    user.Username,
			Email:       user.Email,
			PhoneNumber: user.PhoneNumber,
		},
	}, nil
}