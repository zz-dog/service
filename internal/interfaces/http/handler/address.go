package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	addressapp "github.com/wsc-zz/service/internal/application/address"
	domainaddress "github.com/wsc-zz/service/internal/domain/address"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

// AddressHandler 是收货地址的 HTTP 处理器，持有应用服务以处理请求。
type AddressHandler struct {
	addressSvc *addressapp.Service
}

// NewAddressHandler 构造处理器，注入地址应用服务。
func NewAddressHandler(addressSvc *addressapp.Service) *AddressHandler {
	return &AddressHandler{addressSvc: addressSvc}
}

// parseAddressID 解析 URL 里的地址 ID，非法返回 0。
func parseAddressID(c *gin.Context) uint {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(id)
}

// createAddressReq 新增地址请求结构体（带 gin binding 标签，仅接口层感知 Web 框架）
type createAddressReq struct {
	Receiver  string `json:"receiver" binding:"required,max=32"`
	Phone     string `json:"phone" binding:"required,len=11"`
	Province  string `json:"province" binding:"required,max=32"`
	City      string `json:"city" binding:"required,max=32"`
	District  string `json:"district" binding:"required,max=32"`
	Detail    string `json:"detail" binding:"required,max=255"`
	IsDefault bool   `json:"isDefault"`
}

// updateAddressReq 部分更新请求结构体，全部指针，未传不修改。
type updateAddressReq struct {
	Receiver *string `json:"receiver" binding:"omitempty,max=32"`
	Phone    *string `json:"phone" binding:"omitempty,len=11"`
	Province *string `json:"province" binding:"omitempty,max=32"`
	City     *string `json:"city" binding:"omitempty,max=32"`
	District *string `json:"district" binding:"omitempty,max=32"`
	Detail   *string `json:"detail" binding:"omitempty,max=255"`
}

// Create 新增收货地址
//
//	@Summary		新增收货地址
//	@Description	新增当前登录用户的收货地址。第一条地址自动设为默认；isDefault=true 时同事务清掉其它默认标记。需要登录（JWT）
//	@Tags			地址
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			request	body	createAddressReq	true	"地址信息"
//	@Success		200	{object}	response.Response{data=addressapp.AddressDto}
//	@Failure		400	{object}	response.Response	"参数校验失败或已达地址上限"
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/user/address [post]
func (h *AddressHandler) Create(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	var req createAddressReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	dto, err := h.addressSvc.Create(c.Request.Context(), addressapp.CreateInput{
		UserID:    userID,
		Receiver:  req.Receiver,
		Phone:     req.Phone,
		Province:  req.Province,
		City:      req.City,
		District:  req.District,
		Detail:    req.Detail,
		IsDefault: req.IsDefault,
	})
	if err != nil {
		writeAddressError(c, err)
		return
	}
	response.SuccessMsg(c, "createAddress", dto)
}

// List 查询当前用户地址列表
//
//	@Summary		查询当前用户地址列表
//	@Description	查询 JWT 对应用户的全部收货地址，默认地址排最前。需要登录（JWT）
//	@Tags			地址
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.Response{data=[]addressapp.AddressDto}
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/user/address [get]
func (h *AddressHandler) List(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	dtos, err := h.addressSvc.List(c.Request.Context(), userID)
	if err != nil {
		writeAddressError(c, err)
		return
	}
	response.SuccessMsg(c, "listAddresses", dtos)
}

// GetAddress 查询地址详情
//
//	@Summary		地址详情
//	@Description	查询指定地址详情；地址不存在或不属于当前用户都返回 404。需要登录（JWT）
//	@Tags			地址
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path	int	true	"地址ID"
//	@Success		200	{object}	response.Response{data=addressapp.AddressDto}
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		404	{object}	response.Response	"收货地址不存在"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/user/address/{id} [get]
func (h *AddressHandler) GetAddress(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	id := parseAddressID(c)
	dto, err := h.addressSvc.Get(c.Request.Context(), userID, id)
	if err != nil {
		writeAddressError(c, err)
		return
	}
	response.SuccessMsg(c, "getAddress", dto)
}

// Update 部分更新地址
//
//	@Summary		更新收货地址
//	@Description	部分更新指定地址，未传字段不修改；地址不存在或不属于当前用户返回 404。需要登录（JWT）
//	@Tags			地址
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path	int				true	"地址ID"
//	@Param			request	body	updateAddressReq	true	"待更新字段"
//	@Success		200		{object}	response.Response{data=addressapp.AddressDto}
//	@Failure		400		{object}	response.Response	"参数校验失败或字段值非法"
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		404		{object}	response.Response	"收货地址不存在"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/user/address/{id} [put]
func (h *AddressHandler) Update(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	id := parseAddressID(c)
	var req updateAddressReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	dto, err := h.addressSvc.Update(c.Request.Context(), addressapp.UpdateInput{
		AddressID: id,
		UserID:    userID,
		Receiver:  req.Receiver,
		Phone:     req.Phone,
		Province:  req.Province,
		City:      req.City,
		District:  req.District,
		Detail:    req.Detail,
	})
	if err != nil {
		writeAddressError(c, err)
		return
	}
	response.SuccessMsg(c, "updateAddress", dto)
}

// Delete 删除地址（软删）
//
//	@Summary		删除收货地址
//	@Description	软删除指定地址；地址不存在或不属于当前用户返回 404。需要登录（JWT）
//	@Tags			地址
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path	int	true	"地址ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		404	{object}	response.Response	"收货地址不存在"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/user/address/{id} [delete]
func (h *AddressHandler) Delete(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	id := parseAddressID(c)
	err := h.addressSvc.Delete(c.Request.Context(), userID, id)
	if err != nil {
		writeAddressError(c, err)
		return
	}
	response.SuccessMsg(c, "deleteAddress", nil)
}

// SetDefault 把指定地址设为默认
//
//	@Summary		设为默认地址
//	@Description	把指定地址设为默认地址，同事务清掉其它默认标记，保证默认地址唯一。需要登录（JWT）
//	@Tags			地址
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path	int	true	"地址ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		404	{object}	response.Response	"收货地址不存在"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/user/address/{id}/default [put]
func (h *AddressHandler) SetDefault(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	id := parseAddressID(c)
	err := h.addressSvc.SetDefault(c.Request.Context(), userID, id)
	if err != nil {
		writeAddressError(c, err)
		return
	}
	response.SuccessMsg(c, "setDefault", nil)
}

// writeAddressError 将领域/应用错误映射为对应的 HTTP 响应。
func writeAddressError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainaddress.ErrInvalidPhone),
		errors.Is(err, domainaddress.ErrInvalidReceiver),
		errors.Is(err, domainaddress.ErrInvalidRegion),
		errors.Is(err, domainaddress.ErrInvalidDetail),
		errors.Is(err, domainaddress.ErrAddressLimit):
		response.BadRequest(c, err.Error())
	case errors.Is(err, domainaddress.ErrAddressNotFound),
		errors.Is(err, domainaddress.ErrAddressNotOwned):
		response.NotFound(c, err.Error())
	default:
		response.ServerError(c, err.Error())
	}
}
