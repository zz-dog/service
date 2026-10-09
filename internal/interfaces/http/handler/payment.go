package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	paymentapp "github.com/wsc-zz/service/internal/application/payment"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"

	domianpayment "github.com/wsc-zz/service/internal/domain/payment"
)

type paymentHandler struct {
	paymentSvc *paymentapp.Service
}

func NewPaymentHandler(paymentSvc *paymentapp.Service) *paymentHandler {
	return &paymentHandler{paymentSvc: paymentSvc}
}

type PrepayRequest struct {
	OrderNo string                `json:"orderNo" binding:"required"`
	UserID  uint                  `json:"userId" binding:"required"`
	Channel domianpayment.Channel `json:"channel" binding:"required"`
}

func (h *paymentHandler) Prepay(c *gin.Context) {
	var req PrepayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return

	}
	dto, err := h.paymentSvc.Prepay(c.Request.Context(), paymentapp.PrepayReq{
		OrderNo: req.OrderNo,
		UserID:  req.UserID,
		Channel: req.Channel,
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, validator.ErrorMsg(err))
		return
	}
	response.Success(c, dto)
}

type NotifyRequest struct {
	PayNo string `json:"payNo" binding:"required"`
}

func (h *paymentHandler) Notify(c *gin.Context) {
	var req NotifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return

	}
	err := h.paymentSvc.HandleChannelNotify(c.Request.Context(), req.PayNo)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, validator.ErrorMsg(err))
		return
	}
	response.Success(c, nil)
}
