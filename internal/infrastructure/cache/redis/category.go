package redis

import (
	"context"
	"encoding/json"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/wsc-zz/service/global"
	categoryapp "github.com/wsc-zz/service/internal/application/category"
	"go.uber.org/zap"
)

// key 命名三段式:业务域:资源:范围
const (
	categoryAllKey   = "catalog:category:all"
	categoryAllTTL   = 10 * time.Minute // 正常数据 TTL:写时失效失败时脏读的最长时间
	categoryEmptyTTL = 60 * time.Second // 空结果短 TTL:防缓存穿透
)

type CategoryCache struct {
	client *redis.Client
}

// NewCategoryCache 创建分类缓存对象
func NewCategoryCache(client *redis.Client) *CategoryCache {
	return &CategoryCache{client: client}
}

// GetCategories 从缓存获取所有分类
func (c *CategoryCache) GetCategories(ctx context.Context) ([]*categoryapp.CategoryDto, bool, error) {
	val, err := c.client.Get(ctx, categoryAllKey).Result()
	if err == redis.Nil {
		return nil, false, nil // 未命中,属正常路径
	}
	if err != nil {
		global.Logger.Warn("读取分类缓存失败，回源数据库", zap.Error(err))
		return nil, false, err

	}
	var dtos []*categoryapp.CategoryDto
	if err := json.Unmarshal([]byte(val), &dtos); err != nil {
		global.Logger.Warn("分类缓存反序列化失败，回源数据库", zap.Error(err))
		return nil, false, nil
	}
	return dtos, true, nil
}

func (c *CategoryCache) SetCategories(ctx context.Context, categories []*categoryapp.CategoryDto) error {
	val, err := json.Marshal(categories)
	if err != nil {
		global.Logger.Warn("分类数据序列化失败", zap.Error(err))
		return err
	}

	ttl := categoryAllTTL
	if len(categories) == 0 {
		ttl = categoryEmptyTTL
	}
	return c.client.Set(ctx, categoryAllKey, val, ttl).Err()
}

func (c *CategoryCache) Invalidate(ctx context.Context) error {
	return c.client.Del(ctx, categoryAllKey).Err()
}
