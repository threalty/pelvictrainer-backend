package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTClaims структура JWT токена
type JWTClaims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"` // НОВОЕ: роль пользователя
	jwt.RegisteredClaims
}

// JWTService сервис для работы с JWT токенами
type JWTService struct {
	secretKey      []byte
	accessTokenTTL time.Duration
}

// NewJWTService создаёт новый JWT сервис
func NewJWTService(secretKey string) *JWTService {
	return &JWTService{
		secretKey:      []byte(secretKey),
		accessTokenTTL: 15 * time.Minute, // Access токен живёт 15 минут
	}
}

// GenerateAccessToken создаёт access токен
func (s *JWTService) GenerateAccessToken(userID int, email string, role string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role, // НОВОЕ: добавляем роль
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "pelvictrainer-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// GenerateRefreshToken создаёт refresh токен (UUID)
func (s *JWTService) GenerateRefreshToken() string {
	return uuid.New().String()
}

// ValidateToken проверяет и парсит JWT токен
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем что алгоритм подписи - HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}