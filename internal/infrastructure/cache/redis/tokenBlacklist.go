package redis

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type TokenBlacklist struct {
	client *redis.Client
}

func NewTokenBlacklist(client *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{
		client: client,
	}
}

func (t *TokenBlacklist) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	return t.client.Set(ctx, jti, 1, ttl).Err()
}

func (t *TokenBlacklist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	_, err := t.client.Get(ctx, jti).Result()
	if err == redis.Nil {
		return false, nil
	}

	return true, err
}
