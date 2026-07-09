package jwt

import (
	"demo/global"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	Username string `json:"username"`
	UserId   string `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userId, username string) (string, error) {
	secret := global.Conf.Jwt.Secret
	expire := time.Hour * time.Duration(global.Conf.Jwt.ExpireHour)
	claims := UserClaims{
		UserId:   userId,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)), // 过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),             // 签发时间
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))

}

func ParseToken(tokenString string) (*UserClaims, error) {
	secret := global.Conf.Jwt.Secret
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
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
