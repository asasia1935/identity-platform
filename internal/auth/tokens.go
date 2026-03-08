package auth

import (
	"errors"
	"time"

	"github.com/asasia1935/identity-platform/internal/util"
	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	secret    []byte
	accessTTL time.Duration
}

func NewTokenManager(jwtSecret string, accessTTL time.Duration) (*TokenManager, error) {
	if jwtSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	if accessTTL <= 0 {
		return nil, errors.New("ACCESS_TOKEN_TTL must be positive")
	}
	return &TokenManager{
		secret:    []byte(jwtSecret),
		accessTTL: accessTTL,
	}, nil
}

// AccessToken 전용 Claims 구조체
type AccessTokenClaims struct {
	jwt.RegisteredClaims // 표준 JWT 필드 묶음
}

// JWT 생성 코드
func (m *TokenManager) GenerateAccessToken(userName string) (string, error) {
	now := time.Now()

	claims := AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userName,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// JWT 검증 코드
func (m *TokenManager) VerifyAccessToken(rawToken string) (*AccessTokenClaims, error) {
	var claims AccessTokenClaims

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), // HS256만 허용
	)

	token, err := parser.ParseWithClaims(rawToken, &claims, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	})

	if err != nil {
		// 토큰 만료/무효 (errors.Is로 에러 확인)
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		// 그 외는 전부 invalid로 취급 (서명불일치/포맷오류/기타)
		return nil, ErrTokenInvalid
	}

	if token == nil || !token.Valid {
		return nil, ErrTokenInvalid
	}

	if claims.Subject == "" {
		return nil, ErrTokenInvalid
	}

	return &claims, nil
}

// RefreshToken 전용 Claims 구조체
type RefreshTokenClaims struct {
	JTI string `json:"jti"` // 고유 식별자 (JWT ID)
	jwt.RegisteredClaims
}

// Refresh Token 생성 코드
func (m *TokenManager) GenerateRefreshToken(uid string) (string, string, error) {
	now := time.Now()

	jti, err := util.NewRandomID()
	if err != nil {
		return "", "", err
	}

	claims := RefreshTokenClaims{
		JTI: jti,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uid,
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", "", err
	}

	return signed, jti, nil
}
