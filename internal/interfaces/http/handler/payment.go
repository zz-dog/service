package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
	paymentapp "github.com/wsc-zz/service/internal/application/payment"
	alipaychannel "github.com/wsc-zz/service/internal/infrastructure/channel/alipay"
	"github.com/wsc-zz/service/internal/infrastructure/channel/mock"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"

	domianpayment "github.com/wsc-zz/service/internal/domain/payment"
)

type paymentHandler struct {
	paymentSvc *paymentapp.Service
	// mock/alipay 按装配传 nil:路由按 payment.channel 只注册对应的通知端点,
	// handler 里的 nil 分支是装配错误的保险丝,不是正常路径
	mock   *mock.MockChannel
	alipay *alipaychannel.Channel
}

func NewPaymentHandler(paymentSvc *paymentapp.Service, mockCh *mock.MockChannel, alipayCh *alipaychannel.Channel) *paymentHandler {
	return &paymentHandler{paymentSvc: paymentSvc, mock: mockCh, alipay: alipayCh}
}

type prepayRequest struct {
	OrderNo string `json:"orderNo" binding:"required"`
}

// Prepay 预支付:生成(或复用)支付单,向渠道下单,返回收银台跳转地址。
//
//	@Summary		预支付
//	@Description	按订单号发起支付。金额与归属以订单为准,userID 取自 JWT,不采信请求参数
//	@Tags			支付
//	@Security		ApiKeyAuth
//	@Param			request	body		prepayRequest	true	"订单号"
//	@Success		200		{object}	response.Response{data=paymentapp.PrepayDto}
//	@Failure		400		{object}	response.Response	"订单不可支付/参数错误"
//	@Failure		404		{object}	response.Response	"订单不存在(含他人的订单)"
//	@Router			/payment/prepay [post]
func (h *paymentHandler) Prepay(c *gin.Context) {
	// userID 永远取自 token,不信任请求参数——body 传 userId 是越权后门
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未识别登录身份")
		return
	}
	var req prepayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	dto, err := h.paymentSvc.Prepay(c.Request.Context(), userID, req.OrderNo)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.Success(c, dto)
}

// mapError 把领域错误翻译成 HTTP 语义:归属("别人的订单")和不存在同码,
// 不泄露存在性;可付状态是调用方的问题,400;其余 500。
func (h *paymentHandler) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domianpayment.ErrOrderNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, domianpayment.ErrOrderNotPayable):
		response.BadRequest(c, err.Error())
	default:
		response.ServerError(c, err.Error())
	}
}

type mockNotifyRequest struct {
	PayNo string `json:"payNo" binding:"required"`
}

// MockNotify 模拟渠道异步通知。绝不能挂 JWT 中间件:渠道服务器没有用户令牌,
// 身份证明是签名(header X-Mock-Sign)。验签失败 → 403;成功 → 200 受理。
// 真实渠道对响应体格式有严格要求(如 {"code":"SUCCESS"}),mock 不挑。
//
//	@Summary		mock 渠道异步通知
//	@Description	模拟渠道服务器回调:HMAC 验签 → 状态机推进 → 通知订单。渠道回调不挂 JWT
//	@Tags			支付
//	@Param			request	body		mockNotifyRequest	true	"支付单号"
//	@Success		200		{object}	response.Response	"受理成功"
//	@Failure		403		{string}	string				"签名无效"
//	@Router			/payment/mock/notify [post]
func (h *paymentHandler) MockNotify(c *gin.Context) {
	if h.mock == nil {
		response.ServerError(c, "当前渠道不是 mock,无 mock 通知端点")
		return
	}
	var req mockNotifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	if !h.mock.Verify(req.PayNo, c.GetHeader("X-Mock-Sign")) {
		response.Forbidden(c, "签名无效")
		return
	}
	// mock 收银台按支付单原额支付,金额核对不适用,传 0 表示跳过
	if err := h.paymentSvc.HandleChannelNotify(c.Request.Context(), req.PayNo, 0); err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.SuccessMsg(c, "notify", "受理成功")
}

// MockCashier 模拟渠道收银台:验签后渲染"确认支付"按钮,点击即以渠道身份发通知。
// 纯图个真实感,curl 直接打 notify 也行。
//
//	@Summary		mock 渠道收银台
//	@Description	payUrl 指向这里:验签通过后渲染确认页,点击按钮模拟渠道内部扣款并发起异步通知
//	@Tags			支付
//	@Param			pay_no	query	string	true	"支付单号"
//	@Param			sign	query	string	true	"HMAC-SHA256 签名"
//	@Success		200		{string}	string	"收银台页面"
//	@Failure		400		{string}	string	"签名无效"
//	@Router			/payment/mock/cashier [get]
func (h *paymentHandler) MockCashier(c *gin.Context) {
	if h.mock == nil {
		c.String(http.StatusOK, "当前渠道不是 mock,无收银台")
		return
	}
	payNo := c.Query("pay_no")
	sign := c.Query("sign")
	if !h.mock.Verify(payNo, sign) {
		c.String(http.StatusForbidden, "签名无效")
		return
	}
	// payNo 是纯数字雪花、sign 是 hex,拼进 HTML/JS 无注入风险
	page := fmt.Sprintf(`<!doctype html>
<html><head><meta charset="utf-8"><title>mock 收银台</title></head>
<body style="font-family:sans-serif;text-align:center;padding-top:80px">
  <p>模拟渠道收银台 —— 支付单 %s</p>
  <button onclick="pay()">确认支付</button>
  <p id="result"></p>
<script>
function pay() {
  fetch('/api/payment/mock/notify', {
    method: 'POST',
    headers: {'Content-Type': 'application/json', 'X-Mock-Sign': '%s'},
    body: JSON.stringify({payNo: '%s'})
  }).then(r => r.json())
    .then(d => document.getElementById('result').textContent = '受理结果: ' + (d.code === 200 ? '成功' : JSON.stringify(d)))
    .catch(e => document.getElementById('result').textContent = '失败: ' + e);
}
</script>
</body></html>`, payNo, sign, payNo)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}

// AlipayNotify 支付宝异步通知(渠道服务器 POST form 过来,无 JWT,身份证明是验签)。
// 六步:ParseForm → 验签解析 → 终态过滤 → 金额核对(应用层) → 记账 → ack。
// 成功必须回字面量 success(JSON 不认);失败回非 success,支付宝按递减间隔重试十几次。
//
//	@Summary		支付宝异步通知
//	@Description	支付宝服务器回调:公钥验签 → 终态过滤 → 金额核对 → 记账 → ack success
//	@Tags			支付
//	@Param			form	formData	string	true	"支付宝异步通知表单"
//	@Success		200		{string}	string	"success / fail"
//	@Router			/payment/notify [post]
func (h *paymentHandler) AlipayNotify(c *gin.Context) {
	if h.alipay == nil {
		c.String(http.StatusOK, "fail")
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusOK, "fail")
		return
	}
	noti, err := h.alipay.ParseNotify(c.Request.Context(), c.Request.Form)
	if err != nil {
		// 验签失败:通知可能被伪造,后续所有步骤都不该发生
		global.Logger.Warn("支付宝通知验签/解析失败", zap.Error(err))
		c.String(http.StatusOK, "fail")
		return
	}
	if !noti.TradeSuccess {
		// 中间态(WAIT_BUYER_PAY)或已关闭(TRADE_CLOSED)也回 success——
		// 单子还没付/无需记账,回 fail 只会招来无意义的重试
		alipaychannel.AckNotification(c.Writer)
		return
	}
	// 金额核对在应用层(先比对再 MarkPaid),金额以通知为准换算成分
	if err := h.paymentSvc.HandleChannelNotify(c.Request.Context(), noti.PayNo, noti.AmountFen); err != nil {
		global.Logger.Error("支付宝通知记账失败",
			zap.String("pay_no", noti.PayNo), zap.Error(err))
		c.String(http.StatusOK, "fail") // 绝不回 success:账没记成就闭嘴,只能靠对账捞
		return
	}
	alipaychannel.AckNotification(c.Writer)
}
