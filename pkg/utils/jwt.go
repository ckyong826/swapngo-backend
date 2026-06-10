package utils

import (
	"fmt"
	"time"

	config "swapngo-backend/pkg/configs"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateAccessToken embeds userID and role in the JWT.
func GenerateAccessToken(userID string, role string) (string, error) {
	duration := config.Env.JWTAccessTime
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(duration).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.Env.JWTSecret)
}

// GenerateRefreshToken uses JWTRefreshTime (e.g., 7 days)
func GenerateRefreshToken(userID string) (string, error) {
	duration := config.Env.JWTRefreshTime
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(duration).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.Env.JWTSecret)
}

// ParseJWTClaims extracts userID and role from a valid token.
func ParseJWTClaims(tokenString string) (userID string, role string, err error) {
	token, parseErr := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return config.Env.JWTSecret, nil
	})

	if parseErr != nil {
		return "", "", parseErr
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, _ = claims["user_id"].(string)
		role, _ = claims["role"].(string)
		if role == "" {
			role = "USER" // backwards-compat for tokens minted without role
		}
		return userID, role, nil
	}

	return "", "", fmt.Errorf("invalid token")
}

// ParseJWT extracts only the userID — kept for backwards compatibility.
func ParseJWT(tokenString string) (string, error) {
	userID, _, err := ParseJWTClaims(tokenString)
	return userID, err
}
