package alipay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	alipay "github.com/smartwalle/alipay/v3"

	prepayapp "github.com/wsc-zz/service/internal/application/payment"
	settlementapp "github.com/wsc-zz/service/internal/application/settlement"
)

// Channel 支付宝渠道(公钥模式);isProduction=false 时 SDK 走沙箱网关。
type Channel struct {
	client    *alipay.Client
	notifyURL string // 异步通知地址,必须公网可达;绝不能带 query 参数(支付宝生成签名不含 URL 参数,验签却把收到的全部参数算进去——挂了参数验签必炸)
}

func NewChannel(appId string, privateKey string, publickey string, notifyURL string) (*Channel, error) {
	client, err := alipay.New(appId, privateKey, false) // false = 沙箱网关
	if err != nil {
		return nil, err
	}
	// 加载支付宝公钥:验回调签名用——是平台发的那把,不是你上传的应用公钥
	if err := client.LoadAliPayPublicKey(publickey); err != nil {
		return nil, err
	}
	return &Channel{client: client, notifyURL: notifyURL}, nil
}

func (c *Channel) Prepay(ctx context.Context, req prepayapp.PrepayRequest) (string, error) {
	var p = alipay.TradeWapPay{
		Trade: alipay.Trade{
			OutTradeNo:  req.PayNo, // 商户单号 = payNo,回调靠它认单(一单可能多张支付单,不能用 orderNo)
			TotalAmount: toYuan(req.Amount),
			Subject:     req.Subject,
			ProductCode: "QUICK_WAP_WAY", // 手机网站支付
			NotifyURL:   c.notifyURL,
		},
	}

	result, err := c.client.TradeWapPay(p)
	if err != nil {
		return "", err
	}
	// TradeWapPay 不发 HTTP 请求,只是本地拼参数+签名,返回指向支付宝收银台的 URL;
	// 真正的下单发生在用户打开 URL 之后
	return result.String(), nil
}

// NotifyResult 渠道通知的解析结果:handler 只认这几个字段,不摸 SDK 类型。
type NotifyResult struct {
	PayNo        string // out_trade_no,我们侧的支付单号
	TradeSuccess bool   // TRADE_SUCCESS / TRADE_FINISHED——终态且该记账
	AmountFen    int64  // total_amount 换算成分;非成功终态为 0
}

// ParseNotify 验签 + 解析支付宝异步通知。验签失败返回错误——通知可能被伪造,
// 后续所有步骤都不该发生。金额换算也在这里做完,handler 不碰"元"字符串。
func (c *Channel) ParseNotify(ctx context.Context, form url.Values) (*NotifyResult, error) {
	// 内部已用支付宝公钥验签,失败即签名被篡改或密钥配错
	noti, err := c.client.DecodeNotification(ctx, form)
	if err != nil {
		return nil, err
	}
	r := &NotifyResult{PayNo: noti.OutTradeNo}
	switch noti.TradeStatus {
	case alipay.TradeStatusSuccess, alipay.TradeStatusFinished:
		r.TradeSuccess = true
		fen, err := toFen(noti.TotalAmount)
		if err != nil {
			return nil, err
		}
		r.AmountFen = fen
	default:
		// WAIT_BUYER_PAY 等中间态、TRADE_CLOSED(超时关闭/全额退款):不是错误,
		// 单子还没付或不需记账,渠道之后会再推最终态
	}
	return r, nil
}

func (c *Channel) Settle(ctx context.Context, req settlementapp.SettleRequest) error {
	return errors.New("alipay 分账未对接")

}

// AckNotification 通知处理成功的回执:字面量 "success",回 JSON 支付宝不认。
func AckNotification(w http.ResponseWriter) {
	alipay.ACKNotification(w)
}

// toYuan 12345 分 → "123.45"。支付宝金额是"元"字符串、两位小数。
// 整数分两段各自转,不走 float——浮点在钱上没有生存权(f/100 的整数除法会把分砍掉)。
func toYuan(fen int64) string {
	return fmt.Sprintf("%d.%02d", fen/100, fen%100)
}

// toFen "123.45"(元) → 12345(分)。按小数点拆段各自转整数,拒绝负数;
// 一位小数按整十补齐("0.5" 视作 "0.50")。
func toFen(yuan string) (int64, error) {
	intPart, fracPart, hasDot := strings.Cut(yuan, ".")
	y, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil || y < 0 {
		return 0, fmt.Errorf("金额格式非法: %q", yuan)
	}
	var f int64
	if hasDot {
		if len(fracPart) == 1 {
			fracPart += "0"
		}
		if len(fracPart) != 2 {
			return 0, fmt.Errorf("金额小数位非法: %q", yuan)
		}
		if f, err = strconv.ParseInt(fracPart, 10, 64); err != nil {
			return 0, fmt.Errorf("金额格式非法: %q", yuan)
		}
	}
	return y*100 + f, nil
}
