package auth

import "swapngo-backend/pkg/responses/user"

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User user.UserResponse `json:"user"`
}
