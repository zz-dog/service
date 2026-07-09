package utils

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword 对明文密码进行 bcrypt 加密
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword 校验明文密码和哈希密码是否匹配
func CheckPassword(password, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
