package global

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var RedisClient *redis.Client

func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:         Conf.Redis.Host + ":" + strconv.Itoa(Conf.Redis.Port),
		Password:     Conf.Redis.Password,
		DB:           Conf.Redis.DB,
		PoolSize:     Conf.Redis.PoolSize, // 连接池大小
		MinIdleConns: Conf.Redis.MinIdle,  // 连接池中最小空闲连接数

		// 公网链路抖动大,显式超时保证 Redis 卡顿时请求快速失败,而不是吊死
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		MaxRetries:   2,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		Logger.Fatal("Redis 连接失败", zap.Error(err))
	}
	Logger.Info("Redis 连接成功")
}
