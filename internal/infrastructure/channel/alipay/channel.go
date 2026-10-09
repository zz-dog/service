package alipay

import (
	"context"
	"strconv"

	alipay "github.com/smartwalle/alipay/v3"

	prepayapp "github.com/wsc-zz/service/internal/application/payment"
)

type Chnnel struct {
	channel *alipay.Client
}

func NewChannel(appId string, privateKey string, publickey string) (*Chnnel, error) {
	client, err := alipay.New(appId, privateKey, false)
	if err != nil {
		return nil, err
	}
	if err := client.LoadAliPayPublicKey(publickey); err != nil {
		return nil, err
	}
	return &Chnnel{channel: client}, nil
}

func (c *Chnnel) Prepay(ctx context.Context, req prepayapp.PrepayRequest) (string, error) {
	var p = alipay.TradeWapPay{
		Trade: alipay.Trade{
			OutTradeNo:  req.PayNo,
			TotalAmount: toYuan(req.Amount),
			Subject:     req.Subject,
			ProductCode: "QUICK_WAP_WAY",
			NotifyURL:   "https://www.baidu.com",
		},
	}

	result, err := c.channel.TradeWapPay(p)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

func toYuan(f int64) string {
	return strconv.FormatFloat(float64(f/100), 'f', 2, 64)
}
