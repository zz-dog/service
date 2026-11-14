package imapp

import (
	domainim "github.com/wsc-zz/service/internal/domain/im"
)

type Actor struct {
	Side       domainim.Side
	UserID     uint64
	MerchantID uint64 // 仅 SideMerchant 有值,来自 merchant_rpc 推导
}
