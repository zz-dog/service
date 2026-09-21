package router

import (
	"github.com/wsc-zz/service/global"
	userapp "github.com/wsc-zz/service/internal/application/user"
	rediscache "github.com/wsc-zz/service/internal/infrastructure/cache/redis"
)

func NewTokenBlacklist() userapp.TokenBlacklist {
	return rediscache.NewTokenBlacklist(global.RedisClient)
}
