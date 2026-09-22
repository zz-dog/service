package redis

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type DedupStore struct {
	client *redis.Client
}

func NewDedupStore(client *redis.Client) *DedupStore {
	return &DedupStore{
		client: client,
	}
}

func (s *DedupStore) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {

	return s.client.SetNX(ctx, key, "1", ttl).Result()
}
