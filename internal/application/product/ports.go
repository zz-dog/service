package productapp

import (
	"context"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
)

// MemberInfo catalog 应用层对"商户成员关系"的全部认知:只认业务语义,不认 gRPC。
type MemberInfo struct {
	MerchantID uint
	Name       string
	Status     int // domainmerchant.MStatus*,catalog 据此裁决是否营业中
	Role       domainmerchant.MemberRole
}

// MerchantGateway 商品应用层对"商户能力"的端口(策略在应用层,协议在适配器):
// N:N 后商户端显式携带 merchantId(X-Merchant-Id 头),这里实时校验 (userId, merchantId)
// 的成员关系并带回角色——归属不进 token,也不采信任何请求参数:
// token 是快照,店可能随时被封禁/停业/成员被移除,快照身份会变成僵尸身份;
// 发布是低频操作,一跳 RPC 换实时正确性,值。
type MerchantGateway interface {
	AuthorizeMember(ctx context.Context, userID, merchantID uint) (MemberInfo, error)
}
