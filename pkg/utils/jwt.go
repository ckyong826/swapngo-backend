package utils

import (
	"fmt"
	"time"

	config "swapngo-backend/pkg/configs"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateAccessToken uses JWTAccessTime (e.g., 15 minutes)
func GenerateAccessToken(userID string) (string, error) {
	duration := config.Env.JWTAccessTime
	claims := jwt.MapClaims{
		"user_id": userID,
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

// ParseJWT extracts the userID from a valid token
func ParseJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return config.Env.JWTSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims["user_id"].(string), nil
	}

	return "", fmt.Errorf("invalid token")
}