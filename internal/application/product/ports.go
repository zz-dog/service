package productapp

import "context"

// MerchantInfo catalog 应用层对"商户"的全部认知:只认业务语义,不认 gRPC。
type MerchantInfo struct {
	MerchantID uint
	Name       string
	Status     int // domainmerchant.MStatus*,catalog 据此裁决是否营业中
}

// MerchantGateway 商品应用层对"商户能力"的端口(策略在应用层,协议在适配器):
// 商户身份不能信前端——发布/归属校验前,拿 JWT 的 userId 实时查一次。
// 不把 merchantId 塞 JWT:token 是快照,店可能随时被封禁/停业,token 里的身份会变成僵尸身份;
// 发布是低频操作,一跳 RPC 换实时正确性,值。
type MerchantGateway interface {
	GetMerchantByUserID(ctx context.Context, userID uint) (MerchantInfo, error)
}
