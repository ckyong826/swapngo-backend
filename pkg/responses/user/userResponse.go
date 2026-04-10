package user

type UserResponse struct {
	ID string `json:"id"`
	Username string `json:"username"`
	Email string `json:"email"`
	PhoneNumber string `json:"phone_number"`
}