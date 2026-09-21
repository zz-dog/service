package userapp

import (
	"context"
	"time"
)

type TokenBlacklist interface {
	// Revoke 拉黑 jti,ttl 为 token 剩余有效期,到点自动出黑名单
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
	// IsRevoked 判断 jti 是否已被拉黑
	IsRevoked(ctx context.Context, jti string) (bool, error)
}
