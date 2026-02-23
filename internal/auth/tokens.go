package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(jwtSecret string, accessTTL time.Duration) (*Manager, error) {
	if jwtSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	if accessTTL <= 0 {
		return nil, errors.New("ACCESS_TOKEN_TTL must be positive")
	}
	return &Manager{
		secret: []byte(jwtSecret),
		ttl:    accessTTL,
	}, nil
}

// AccessToken 전용 Claims 구조체
type AccessTokenClaims struct {
	jwt.RegisteredClaims // 표준 JWT 필드 묶음
}

// JWT 생성 코드
func (m *Manager) GenerateAccessToken(userName string) (string, error) {
	now := time.Now()

	claims := AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userName,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// JWT 검증 코드
func (m *Manager) VerifyAccessToken(rawToken string) (*AccessTokenClaims, error) {
	var claims AccessTokenClaims

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), // HS256만 허용
	)

	token, err := parser.ParseWithClaims(rawToken, &claims, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	if claims.Subject == "" {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return &claims, nil
}
