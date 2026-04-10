package services

import (
	"context"
	"errors"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	authReq "swapngo-backend/pkg/requests/auth"
	"swapngo-backend/pkg/utils"
)

type UserService interface {
	RegisterUser(ctx context.Context, req *authReq.RegisterRequest) (models.User, error)
	VerifyUser(ctx context.Context, req *authReq.LoginRequest) (models.User, error)
}

type userService struct {
	userRepo repositories.UserRepository
}

func NewUserService(repo repositories.UserRepository) UserService {
	return &userService{
		userRepo: repo,
	}
}

func (s *userService) RegisterUser(ctx context.Context, req *authReq.RegisterRequest) (models.User, error) {
	// 1. Check if user already exists
	existingUser, err := s.userRepo.CheckExist(ctx, req.PhoneNumber, req.Email, req.Username)
	if err != nil {
		return models.User{}, err
	}
	if existingUser != nil {
		if existingUser.PhoneNumber == req.PhoneNumber {
			return models.User{}, errors.New("phone number is already registered")
		}
		if existingUser.Email == req.Email {
			return models.User{}, errors.New("email is already registered")
		}
		if existingUser.Username == req.Username {
			return models.User{}, errors.New("username is already registered")
		}
	}

	// 2. Check validity of password and pin
	if !utils.IsValidPassword(req.Password) {
		return models.User{}, errors.New("password must be at least 8 characters long and contain at least one letter, one number, and one symbol")
	}

	if !utils.IsValidPin(req.Pin) {
		return models.User{}, errors.New("payment PIN must be exactly 4 digits")
	}

	// 3. Hash password and pin
	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		return models.User{}, err
	}
	pinHash, err := utils.HashPassword(req.Pin)
	if err != nil {
		return models.User{}, err
	}

	// 4. Create user
	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PhoneNumber:  req.PhoneNumber,
		PasswordHash: passwordHash,
		PinHash:      pinHash,
	}

	// 5. Return user
	user, err = s.userRepo.Create(ctx, user)
	if err != nil {
		return models.User{}, err
	}
	return *user, nil
}

func (s *userService) VerifyUser(ctx context.Context, req *authReq.LoginRequest) (models.User, error) {
	// 1. Check if user exists
	user, err := s.userRepo.CheckExist(ctx, req.PhoneNumber, req.Email, req.Username)
	if err != nil {
		return models.User{}, err
	}
	if user == nil {
		return models.User{}, errors.New("user not found")
	}

	// 2. Verify password
	valid, err := utils.CheckPassword(user.PasswordHash, req.Password)
	if err != nil {
		return models.User{}, err
	}
	if !valid {
		return models.User{}, errors.New("invalid password")
	}

	return *user, nil
}
		
	
