package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	productapp "github.com/wsc-zz/service/internal/application/product"
	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
	domainproduct "github.com/wsc-zz/service/internal/domain/product"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

// 商户端商品接口:挂在 catalog 的 /merchant 路由组(JWTAuth;归属校验在应用层)。
// N:N 鉴权模型:userID 取自 token,"当前店铺"取自 X-Merchant-Id 头,
// (userId, merchantId) 的成员关系与角色由 merchant RPC 实时校验,body 传什么都不采信。

// merchantContext 统一取"当前登录用户 + 当前店铺":
// 头缺失返回 false,由调用方回 400;归属是否成立由应用层裁决。
func merchantContext(c *gin.Context) (userID, merchantID uint, ok bool) {
	userID, ok = getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return 0, 0, false
	}
	merchantID, ok = getCurrentMerchantID(c)
	if !ok {
		response.BadRequest(c, "缺少 " + HeaderMerchantID + " 店铺上下文头")
		return 0, 0, false
	}
	return userID, merchantID, true
}

// writeMerchantProductError 商品域商户端错误 → HTTP 状态码:
// 归属类 404(不泄露存在性)、权限类 403、规则类 400、其余 500。
func writeMerchantProductError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainmerchant.ErrMerchantNotFound),
		errors.Is(err, domainproduct.ErrProductNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, domainmerchant.ErrPermissionDenied):
		response.Forbidden(c, err.Error())
	case errors.Is(err, domainmerchant.ErrMerchantNotOperating):
		response.BadRequest(c, err.Error())
	default:
		if isBadRequest(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, err.Error())
	}
}

// MerchantPublish 商户发布商品
//
//	@Summary		发布商品(商户)
//	@Description	商户发布商品,服务端实时校验 (userId, merchantId) 成员关系;
//	             	owner/admin 可发布,staff 无权;商品进入待审核,审核通过才对 C 端可见
//	@Tags			商户端-商品
//	@Accept			json
//	@Produce		json
//	@Param			X-Merchant-Id	header		int					true	"当前店铺ID"
//	@Param			request			body		createProductReq	true	"商品信息(merchantId 不收,走店铺上下文头)"
//	@Success		200				{object}	response.Response{data=productapp.ProductDTO}
//	@Failure		400				{object}	response.Response	"参数校验失败/缺店铺上下文/店铺未营业"
//	@Failure		401				{object}	response.Response	"未登录或 token 失效"
//	@Failure		403				{object}	response.Response	"是成员但角色不够(staff 不能发布)"
//	@Failure		404				{object}	response.Response	"店铺不存在或非成员(不泄露存在性)"
//	@Failure		500				{object}	response.Response	"服务器内部错误"
//	@Router			/catalog/merchant/product [post]
func (h *ProductHandler) MerchantPublish(c *gin.Context) {
	userID, merchantID, ok := merchantContext(c)
	if !ok {
		return
	}
	var req createProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	in := productapp.CreateProductInput{
		CategoryID: req.CategoryID,
		Name:       req.Name,
		Desc:       req.Desc,
		SKUs:       toSKUInputs(req.SKUs),
		Medias:     toMediaInputs(req.Medias),
	}
	resp, err := h.productSvc.MerchantPublish(c.Request.Context(), userID, merchantID, in)
	if err != nil {
		writeMerchantProductError(c, err)
		return
	}
	response.Success(c, resp)
}

type merchantProductListReq struct {
	CategoryID uint   `form:"categoryId"`
	Name       string `form:"name"`
	Page       int    `form:"page" binding:"required"`
	PageSize   int    `form:"pageSize" binding:"required"`
}

// MyProducts 我的商品(商户)
//
//	@Summary		我的商品列表(全状态)
//	@Description	当前店铺的全部商品,含待审核/已驳回(带理由)/已下架;归属即过滤条件
//	@Tags			商户端-商品
//	@Produce		json
//	@Param			X-Merchant-Id	header	int		true	"当前店铺ID"
//	@Param			categoryId		query	int		false	"分类ID"
//	@Param			name			query	string	false	"商品名称(模糊匹配)"
//	@Param			page			query	int		true	"页码"
//	@Param			pageSize		query	int		true	"每页数量"
//	@Success		200				{object}	response.Response{data=productapp.ProductListResult}
//	@Failure		400				{object}	response.Response	"查询参数校验失败/缺店铺上下文"
//	@Failure		401				{object}	response.Response	"未登录或 token 失效"
//	@Failure		403				{object}	response.Response	"是成员但角色不够"
//	@Failure		404				{object}	response.Response	"店铺不存在或非成员(不泄露存在性)"
//	@Failure		500				{object}	response.Response	"服务器内部错误"
//	@Router			/catalog/merchant/product/my [get]
func (h *ProductHandler) MyProducts(c *gin.Context) {
	userID, merchantID, ok := merchantContext(c)
	if !ok {
		return
	}
	var req merchantProductListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	result, err := h.productSvc.MyProducts(c.Request.Context(), userID, merchantID, domainproduct.ListQuery{
		CategoryID: req.CategoryID,
		Name:       req.Name,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		writeMerchantProductError(c, err)
		return
	}
	response.Success(c, result)
}

type merchantUpdateProductReq struct {
	CategoryID uint       `json:"categoryId" binding:"required"`
	Name       string     `json:"name" binding:"required"`
	Desc       string     `json:"desc"`
	Medias     []mediaReq `json:"medias"` // 整体替换:不传则清空
}

// MerchantUpdate 商户修改商品
//
//	@Summary		修改商品(商户)
//	@Description	修改商品资料,内容变更回待审核;违规下架的商品不可修改。owner/admin 可操作
//	@Tags			商户端-商品
//	@Accept			json
//	@Produce		json
//	@Param			X-Merchant-Id	header		int							true	"当前店铺ID"
//	@Param			id				path		int							true	"商品ID"
//	@Param			request			body		merchantUpdateProductReq	true	"商品更新信息"
//	@Success		200				{object}	response.Response{data=productapp.ProductDTO}
//	@Failure		400				{object}	response.Response	"参数校验失败/商品不存在(含非本店商品)"
//	@Failure		401				{object}	response.Response	"未登录或 token 失效"
//	@Failure		403				{object}	response.Response	"是成员但角色不够(staff 不能修改商品)"
//	@Failure		404				{object}	response.Response	"店铺或商品不存在(含非成员/非本店商品,不泄露存在性)"
//	@Failure		500				{object}	response.Response	"服务器内部错误"
//	@Router			/catalog/merchant/product/{id} [put]
func (h *ProductHandler) MerchantUpdate(c *gin.Context) {
	userID, merchantID, ok := merchantContext(c)
	if !ok {
		return
	}
	id := parseUintParam(c, "id")
	if id == 0 {
		response.BadRequest(c, "商品ID格式错误")
		return
	}
	var req merchantUpdateProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	in := productapp.UpdateProductInput{
		ProductID:  id,
		CategoryID: req.CategoryID,
		Name:       req.Name,
		Desc:       req.Desc,
		Medias:     toMediaInputs(req.Medias),
	}
	resp, err := h.productSvc.MerchantUpdate(c.Request.Context(), userID, merchantID, in)
	if err != nil {
		writeMerchantProductError(c, err)
		return
	}
	response.Success(c, resp)
}

// MerchantDelete 商户删除商品
//
//	@Summary		删除商品(商户)
//	@Description	仅已下架/已驳回的商品可删除(软删除)。owner/admin 可操作
//	@Tags			商户端-商品
//	@Produce		json
//	@Param			X-Merchant-Id	header	int		true	"当前店铺ID"
//	@Param			id				path	int		true	"商品ID"
//	@Success		200				{object}	response.Response
//	@Failure		400				{object}	response.Response	"商品不存在(含非本店商品)或当前状态不允许删除"
//	@Failure		401				{object}	response.Response	"未登录或 token 失效"
//	@Failure		403				{object}	response.Response	"是成员但角色不够(staff 不能删除商品)"
//	@Failure		404				{object}	response.Response	"店铺或商品不存在(含非成员,不泄露存在性)"
//	@Failure		500				{object}	response.Response	"服务器内部错误"
//	@Router			/catalog/merchant/product/{id} [delete]
func (h *ProductHandler) MerchantDelete(c *gin.Context) {
	userID, merchantID, ok := merchantContext(c)
	if !ok {
		return
	}
	id := parseUintParam(c, "id")
	if id == 0 {
		response.BadRequest(c, "商品ID格式错误")
		return
	}
	err := h.productSvc.MerchantDelete(c.Request.Context(), userID, merchantID, id)
	if err != nil {
		writeMerchantProductError(c, err)
		return
	}
	response.Success(c, nil)
}

// MerchantOnShelf 商户上架
//
//	@Summary		上架(商户)
//	@Description	已下架的商品重新上架,免重审(下架是经营决策,不是合规否决)
//	@Tags			商户端-商品
//	@Produce		json
//	@Param			X-Merchant-Id	header	int		true	"当前店铺ID"
//	@Param			id				path	int		true	"商品ID"
//	@Success		200				{object}	response.Response
//	@Failure		400				{object}	response.Response	"商品不存在(含非本店商品)或当前状态不允许上架"
//	@Failure		401				{object}	response.Response	"未登录或 token 失效"
//	@Failure		403				{object}	response.Response	"是成员但角色不够"
//	@Failure		404				{object}	response.Response	"店铺或商品不存在(含非成员,不泄露存在性)"
//	@Failure		500				{object}	response.Response	"服务器内部错误"
//	@Router			/catalog/merchant/product/{id}/on-shelf [post]
func (h *ProductHandler) MerchantOnShelf(c *gin.Context) {
	userID, merchantID, ok := merchantContext(c)
	if !ok {
		return
	}
	id := parseUintParam(c, "id")
	if id == 0 {
		response.BadRequest(c, "商品ID格式错误")
		return
	}
	err := h.productSvc.MerchantOnShelf(c.Request.Context(), userID, merchantID, id)
	if err != nil {
		writeMerchantProductError(c, err)
		return
	}
	response.Success(c, nil)
}

// MerchantOffShelf 商户下架
//
//	@Summary		下架(商户)
//	@Description	已上架的商品下架,下架后 C 端不可见
//	@Tags			商户端-商品
//	@Produce		json
//	@Param			X-Merchant-Id	header	int		true	"当前店铺ID"
//	@Param			id				path	int		true	"商品ID"
//	@Success		200				{object}	response.Response
//	@Failure		400				{object}	response.Response	"商品不存在(含非本店商品)或当前状态不允许下架"
//	@Failure		401				{object}	response.Response	"未登录或 token 失效"
//	@Failure		403				{object}	response.Response	"是成员但角色不够"
//	@Failure		404				{object}	response.Response	"店铺或商品不存在(含非成员,不泄露存在性)"
//	@Failure		500				{object}	response.Response	"服务器内部错误"
//	@Router			/catalog/merchant/product/{id}/off-shelf [post]
func (h *ProductHandler) MerchantOffShelf(c *gin.Context) {
	userID, merchantID, ok := merchantContext(c)
	if !ok {
		return
	}
	id := parseUintParam(c, "id")
	if id == 0 {
		response.BadRequest(c, "商品ID格式错误")
		return
	}
	err := h.productSvc.MerchantOffShelf(c.Request.Context(), userID, merchantID, id)
	if err != nil {
		writeMerchantProductError(c, err)
		return
	}
	response.Success(c, nil)
}

type merchantInventoryReq struct {
	SKUCode string `json:"skuCode" binding:"required"`
	Qty     int    `json:"qty" binding:"required,gt=0"`
}

// MerchantAddInventory 商户补货
//
//	@Summary		补货(商户)
//	@Description	为本店商品的指定 SKU 增加库存;经营数据,不回审核队列
//	@Tags			商户端-商品
//	@Accept			json
//	@Produce		json
//	@Param			X-Merchant-Id	header		int						true	"当前店铺ID"
//	@Param			id				path		int						true	"商品ID"
//	@Param			request			body		merchantInventoryReq	true	"库存信息"
//	@Success		200				{object}	response.Response
//	@Failure		400				{object}	response.Response	"参数校验失败/商品或规格不存在(含非本店商品)"
//	@Failure		401				{object}	response.Response	"未登录或 token 失效"
//	@Failure		403				{object}	response.Response	"是成员但角色不够"
//	@Failure		404				{object}	response.Response	"店铺或商品不存在(含非成员,不泄露存在性)"
//	@Failure		500				{object}	response.Response	"服务器内部错误"
//	@Router			/catalog/merchant/product/{id}/inventory [post]
func (h *ProductHandler) MerchantAddInventory(c *gin.Context) {
	userID, merchantID, ok := merchantContext(c)
	if !ok {
		return
	}
	id := parseUintParam(c, "id")
	if id == 0 {
		response.BadRequest(c, "商品ID格式错误")
		return
	}
	var req merchantInventoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	err := h.productSvc.MerchantAddInventory(c.Request.Context(), userID, merchantID, id, req.SKUCode, req.Qty)
	if err != nil {
		writeMerchantProductError(c, err)
		return
	}
	response.Success(c, nil)
}

type merchantUpdateSKUReq struct {
	SKUCode string `json:"skuCode" binding:"required"`
	Price   int64  `json:"price" binding:"required,gt=0"`
	Picture string `json:"picture" binding:"required"`
}

// MerchantUpdateSKU 商户调价/换图
//
//	@Summary		调价/换图(商户)
//	@Description	修改本店商品 SKU 的价格和图片;经营数据,不回审核队列
//	@Tags			商户端-商品
//	@Accept			json
//	@Produce		json
//	@Param			X-Merchant-Id	header		int						true	"当前店铺ID"
//	@Param			id				path		int						true	"商品ID"
//	@Param			request			body		merchantUpdateSKUReq	true	"SKU更新信息"
//	@Success		200				{object}	response.Response
//	@Failure		400				{object}	response.Response	"参数校验失败/商品或规格不存在(含非本店商品)"
//	@Failure		401				{object}	response.Response	"未登录或 token 失效"
//	@Failure		403				{object}	response.Response	"是成员但角色不够"
//	@Failure		404				{object}	response.Response	"店铺或商品不存在(含非成员,不泄露存在性)"
//	@Failure		500				{object}	response.Response	"服务器内部错误"
//	@Router			/catalog/merchant/product/{id}/sku [put]
func (h *ProductHandler) MerchantUpdateSKU(c *gin.Context) {
	userID, merchantID, ok := merchantContext(c)
	if !ok {
		return
	}
	id := parseUintParam(c, "id")
	if id == 0 {
		response.BadRequest(c, "商品ID格式错误")
		return
	}
	var req merchantUpdateSKUReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	err := h.productSvc.MerchantUpdateSKU(c.Request.Context(), userID, merchantID, id, req.SKUCode, req.Price, req.Picture)
	if err != nil {
		writeMerchantProductError(c, err)
		return
	}
	response.Success(c, nil)
}

// MerchantDeleteSKU 商户删规格
//
//	@Summary		删除规格(商户)
//	@Description	删除本店商品的指定 SKU;仅已下架商品可删,且至少保留一个 SKU
//	@Tags			商户端-商品
//	@Produce		json
//	@Param			X-Merchant-Id	header	int		true	"当前店铺ID"
//	@Param			id				path	int		true	"商品ID"
//	@Param			skuCode			path	string	true	"SKU编码"
//	@Success		200				{object}	response.Response
//	@Failure		400				{object}	response.Response	"商品或规格不存在(含非本店商品)/仅下架可删/至少保留一个SKU"
//	@Failure		401				{object}	response.Response	"未登录或 token 失效"
//	@Failure		403				{object}	response.Response	"是成员但角色不够"
//	@Failure		404				{object}	response.Response	"店铺或商品不存在(含非成员,不泄露存在性)"
//	@Failure		500				{object}	response.Response	"服务器内部错误"
//	@Router			/catalog/merchant/product/{id}/sku/{skuCode} [delete]
func (h *ProductHandler) MerchantDeleteSKU(c *gin.Context) {
	userID, merchantID, ok := merchantContext(c)
	if !ok {
		return
	}
	id := parseUintParam(c, "id")
	if id == 0 {
		response.BadRequest(c, "商品ID格式错误")
		return
	}
	skuCode := c.Param("skuCode")
	if skuCode == "" {
		response.BadRequest(c, "SKU编码不能为空")
		return
	}
	err := h.productSvc.MerchantDeleteSKU(c.Request.Context(), userID, merchantID, id, skuCode)
	if err != nil {
		writeMerchantProductError(c, err)
		return
	}
	response.Success(c, nil)
}
