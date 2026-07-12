package security

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/wsc-zz/service/internal/domain/user"
)

// BcryptHasher 是 user.PasswordHasher 的 bcrypt 实现。
var _ user.PasswordHasher = (*BcryptHasher)(nil)

// BcryptHasher 使用 bcrypt 算法做密码哈希与校验。
type BcryptHasher struct{}

// NewBcryptHasher 构造 bcrypt 哈希器。
func NewBcryptHasher() *BcryptHasher { return &BcryptHasher{} }

func (h *BcryptHasher) Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (h *BcryptHasher) Compare(hashed, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}
