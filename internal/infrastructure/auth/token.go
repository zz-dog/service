package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/wsc-zz/service/global"
	userapp "github.com/wsc-zz/service/internal/application/user"
)

// UserClaims JWT 载荷
type UserClaims struct {
	Username string `json:"username"`
	UserID   string `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTTokenIssuer 是 userapp.TokenIssuer 的 JWT 实现。
var _ userapp.TokenIssuer = (*JWTTokenIssuer)(nil)

// JWTTokenIssuer 使用 HS256 签发 JWT，密钥与过期时间来自全局配置。
type JWTTokenIssuer struct{}

// NewJWTTokenIssuer 构造 JWT 签发器。
func NewJWTTokenIssuer() *JWTTokenIssuer { return &JWTTokenIssuer{} }

// Issue 为指定用户签发 token。
func (t *JWTTokenIssuer) Issue(userID uint, username string) (string, error) {
	secret := global.Conf.Jwt.Secret
	if len(secret) < 32 {
		return "", errors.New("JWT secret must be at least 32 bytes")
	}
	if global.Conf.Jwt.ExpireHour <= 0 {
		return "", errors.New("JWT expiration must be positive")
	}
	expire := time.Hour * time.Duration(global.Conf.Jwt.ExpireHour)
	now := time.Now()

	claims := UserClaims{
		UserID:   strconv.Itoa(int(userID)),
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expire)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 解析并校验 token，供 HTTP 中间件使用。
func ParseToken(tokenString string) (*UserClaims, error) {
	secret := global.Conf.Jwt.Secret
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must be at least 32 bytes")
	}
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected JWT signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("token无效")
}
