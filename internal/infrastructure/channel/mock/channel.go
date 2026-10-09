package mock

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	prepayapp "github.com/wsc-zz/service/internal/application/payment"
)

// MockChannel mock 渠道:把"第三方支付"整体塞进一个端口实现里。
// 收银台和异步通知端点由 payment 自己的路由暴露(渠道 URL 指向它们),
// 验签密钥来自配置——除了没有真钱,流程与真实渠道同构:
// 预支付 → 用户在渠道收银台确认 → 渠道异步通知商户 → 商户验签后记账。
type MockChannel struct {
	secret  []byte // HMAC 密钥,来自 payment.mock_channel_secret
	baseURL string // 网关地址,拼收银台跳转链接
}

func NewMockChannel(baseURL string, secret string) *MockChannel {
	return &MockChannel{secret: []byte(secret), baseURL: baseURL}
}
func (c *MockChannel) Prepay(ctx context.Context, req prepayapp.PrepayRequest) (payUrl string, err error) {
	// 签名只覆盖 payNo:同一支付单重复预支付得到同一 URL,天然支持复用
	
	return fmt.Sprintf("%s/api/payment/mock/cashier?pay_no=%s&sign=%s",
		c.baseURL, req.PayNo, c.sign(req.PayNo)), nil
}

func (c *MockChannel) sign(s string) string {
	mac := hmac.New(sha256.New, c.secret)
	mac.Write([]byte(s))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify 供 handler 验签:hmac.Equal 是恒时比较,防时序侧信道——细节但免费
func (c *MockChannel) Verify(payNo, sign string) bool {
	return hmac.Equal([]byte(c.sign(payNo)), []byte(sign))
}
