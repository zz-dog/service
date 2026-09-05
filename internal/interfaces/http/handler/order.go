package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	orderapp "github.com/wsc-zz/service/internal/application/order"
	domainorder "github.com/wsc-zz/service/internal/domain/order"
	domainproduct "github.com/wsc-zz/service/internal/domain/product"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

// OrderHandler 是订单相关的 HTTP 处理器。
type OrderHandler struct {
	orderSvc *orderapp.Service
}

func NewOrderHandler(orderSvc *orderapp.Service) *OrderHandler {
	return &OrderHandler{orderSvc: orderSvc}
}

// ---- 请求结构体（带 gin binding 标签，仅接口层感知 Web 框架）----

type orderItemRequest struct {
	ProductID uint   `json:"productId" binding:"required"`
	SKUCode   string `json:"skuCode" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
}
type createOrderRequest struct {
	Items            []orderItemRequest `json:"items" binding:"required,min=1,dive"`
	ConsigneeName    string             `json:"consigneeName" binding:"required"`
	ConsigneePhone   string             `json:"consigneePhone" binding:"required"`
	ConsigneeAddress string             `json:"consigneeAddress" binding:"required"`
}

// Create 创建订单
func (h *OrderHandler) Create(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	items := make([]orderapp.OrderItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, orderapp.OrderItemInput{
			ProductID: it.ProductID,
			SKUCode:   it.SKUCode,
			Quantity:  it.Quantity,
		})
	}

	// 调用应用层服务创建订单
	result, err := h.orderSvc.Create(c.Request.Context(), orderapp.CreateOrderInput{
		UserID:           userID,
		Items:            items,
		ConsigneeName:    req.ConsigneeName,
		ConsigneePhone:   req.ConsigneePhone,
		ConsigneeAddress: req.ConsigneeAddress,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "createOrder", result)
}

// Cancel 取消订单
type CancelOrderRequest struct {
	OrderID uint `json:"orderId" binding:"required"`
	UserID  uint `json:"userId" binding:"required"`
}

func (h *OrderHandler) Cancel(c *gin.Context) {
	var req CancelOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	err := h.orderSvc.CancelOrder(c.Request.Context(), orderapp.CancelOrderInput{
		OrderID: req.OrderID,
		UserID:  req.UserID,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "cancelOrder", "订单取消成功")
}

type OrderListRequest struct {
	Page     int  `json:"page" binding:"omitempty,min=1"`
	PageSize int  `json:"pageSize" binding:"omitempty,min=1,max=100"`
	Status   int  `json:"status" binding:"omitempty,oneof=0 1 2"` // 可选，若不传则查询所有状态的订单
	UserID   uint `json:"userId" binding:"required"`
}

func (h *OrderHandler) List(c *gin.Context) {
	var req OrderListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	result, err := h.orderSvc.List(c.Request.Context(), orderapp.QueryOrdersInput{
		UserID:   req.UserID,
		Page:     req.Page,
		PageSize: req.PageSize,
		Status:   req.Status,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "queryOrders", result)
}

// getCurrentUserID 从 gin 上下文取 JWT 中间件写入的 userID。
// JWT 中间件写入的是 string（见 auth.UserClaims.UserID），这里转成 uint。
// 提示：后续其他 handler 也会用到，可提取到 handler 包的公共文件 base.go。
func getCurrentUserID(c *gin.Context) (uint, bool) {
	v, exists := c.Get("userId")
	if !exists {
		return 0, false
	}
	s, ok := v.(string)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}

//

// writeError 将领域/应用错误映射为对应的 HTTP 响应。
func (h *OrderHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainorder.ErrOrderNotFound):
		response.NotFound(c, http.StatusNotFound, err.Error())
	case errors.Is(err, domainproduct.ErrProductNotFound),
		errors.Is(err, domainproduct.ErrSKUNotFound):
		response.NotFound(c, http.StatusNotFound, err.Error())
	case errors.Is(err, domainproduct.ErrProductOffShelf),
		errors.Is(err, domainproduct.ErrInsufficientStock):
		response.BadRequest(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, domainorder.ErrInvalidStatusTransition),
		errors.Is(err, domainorder.ErrOrderAlreadyCancelled),
		errors.Is(err, domainorder.ErrEmptyOrderItems),
		errors.Is(err, domainorder.ErrInvalidQuantity),
		errors.Is(err, domainorder.ErrInvalidPrice),
		errors.Is(err, domainorder.ErrConsigneeIncomplete):
		response.BadRequest(c, http.StatusBadRequest, err.Error())
	default:
		response.ServerError(c, http.StatusInternalServerError, err.Error())
	}
}
