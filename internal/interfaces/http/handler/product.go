package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	productapp "github.com/wsc-zz/service/internal/application/product"
	domaincategory "github.com/wsc-zz/service/internal/domain/category"
	domainproduct "github.com/wsc-zz/service/internal/domain/product"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

// badRequestErrors 创建商品时属于用户输入问题的错误：返回 400 而非 500
var badRequestErrors = []error{
	productapp.ErrSpecNotBoundToCategory,
	productapp.ErrSpecValueNotInSpec,
	domaincategory.ErrCategoryNotFound,
	domainproduct.ErrEmptyProductName,
	domainproduct.ErrEmptySKUs,
	domainproduct.ErrEmptySKUCode,
	domainproduct.ErrSKUCodeMismatch,
	domainproduct.ErrDuplicateSpecInSKU,
	domainproduct.ErrDuplicateSpecCombination,
	domainproduct.ErrEmptySpecItems,
	domainproduct.ErrInvalidSpecItem,
	domainproduct.ErrInvalidPrice,
	domainproduct.ErrInvalidStock,
}

func isBadRequest(err error) bool {
	for _, target := range badRequestErrors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

type ProductHandler struct {
	productSvc *productapp.Service
}

func NewProductHandler(productSvc *productapp.Service) *ProductHandler {
	return &ProductHandler{
		productSvc: productSvc,
	}
}

type createProductReq struct {
	CategoryID uint   `json:"categoryId" binding:"required"` // 分类ID
	Name       string `json:"name" binding:"required"`       // 商品名称
	Desc       string `json:"desc"`                          // 商品描述
	SKUs       []SkU  `json:"skus" binding:"required"`       // 商品规格
}

type specItemReq struct {
	SpecID  uint `json:"specId" binding:"required"`  // 规格维度ID
	ValueID uint `json:"valueId" binding:"required"` // 规格值ID
	// 名称快照不收客户端值，由服务端从规格库取权威名称
}

type SkU struct {
	// SKUCode 不收客户端值，由服务端从规格组合派生，创建后在响应中返回
	SpecItems []specItemReq `json:"specItems" binding:"required,min=1,dive"` // 规格组合
	Price     int64         `json:"price" binding:"required,gt=0"`           // 价格
	Stock     int           `json:"stock" binding:"gte=0"`                   // 库存
}

// Create 创建商品
// @Summary      创建商品
// @Description  创建商品及其 SKU；SKU 编码和规格名称由服务端生成
// @Tags         商品
// @Accept       json
// @Produce      json
// @Param        request  body      createProductReq  true  "商品信息"
// @Success      200      {object}  response.Response{data=productapp.ProductDTO}
// @Failure      400      {object}  response.Response  "参数校验失败或商品信息无效"
// @Failure      500      {object}  response.Response  "服务器内部错误"
// @Router       /product/create [post]
func (h *ProductHandler) Create(c *gin.Context) {
	var req createProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}

	in := productapp.CreateProductInput{
		CategoryID: req.CategoryID,
		Name:       req.Name,
		Desc:       req.Desc,
		SKUs:       toSKUInputs(req.SKUs),
	}
	resp, err := h.productSvc.Create(c.Request.Context(), in)
	if err != nil {
		if isBadRequest(err) {
			response.BadRequest(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, resp)
}

type updateProductReq struct {
	ProductID  uint     `json:"productId" binding:"required"`
	CategoryID uint     `json:"categoryId"`
	Name       string   `json:"name"`
	Desc       string   `json:"desc"`
	Urls       []string `json:"urls"`
}

// Update 更新商品
// @Summary      更新商品
// @Description  根据商品ID更新商品的分类、名称、描述和图片地址
// @Tags         商品
// @Accept       json
// @Produce      json
// @Param        request  body      updateProductReq  true  "商品更新信息"
// @Success      200      {object}  response.Response{data=productapp.ProductDTO}
// @Failure      400      {object}  response.Response  "参数校验失败或商品信息无效"
// @Failure      500      {object}  response.Response  "服务器内部错误"
// @Router       /product/update [post]
func (h *ProductHandler) Update(c *gin.Context) {
	var req updateProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	in := productapp.UpdateProductInput{
		ProductID:  req.ProductID,
		CategoryID: req.CategoryID,
		Name:       req.Name,
		Desc:       req.Desc,
		Urls:       req.Urls,
	}
	resp, err := h.productSvc.Update(c.Request.Context(), in)
	if err != nil {
		if isBadRequest(err) {
			response.BadRequest(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, resp)
}

type ProdectListReq struct {
	CategoryID uint `form:"categoryId"`
	ProductID  uint `form:"productId"`
	Name       string
	Page       int `form:"page" binding:"required"`
	PageSize   int `form:"pageSize" binding:"required"`
	status     domainproduct.Status
}

// List 分页查询商品
// @Summary      分页查询商品
// @Description  根据分类、商品ID或名称筛选商品并分页返回
// @Tags         商品
// @Produce      json
// @Param        categoryId  query     int     false  "分类ID"
// @Param        productId   query     int     false  "商品ID"
// @Param        name        query     string  false  "商品名称"
// @Param        page        query     int     true   "页码"
// @Param        pageSize    query     int     true   "每页数量"
// @Success      200         {object}  response.Response{data=productapp.ProductListResult}
// @Failure      400         {object}  response.Response  "查询参数校验失败"
// @Failure      500         {object}  response.Response  "服务器内部错误"
// @Router       /product/list [post]
func (h *ProductHandler) List(c *gin.Context) {
	var req ProdectListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	result, err := h.productSvc.List(c.Request.Context(), domainproduct.ListQuery{
		CategoryID: req.CategoryID,
		Name:       req.Name,
		Page:       req.Page,
		PageSize:   req.PageSize,
		ProductID:  req.ProductID,
		Status:     req.status,
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, result)
}
func toSKUInputs(skus []SkU) []productapp.SKUInput {
	out := make([]productapp.SKUInput, 0, len(skus))
	for _, s := range skus {
		items := make([]productapp.SpecItemInput, 0, len(s.SpecItems))
		for _, item := range s.SpecItems {
			items = append(items, productapp.SpecItemInput{
				SpecID:  item.SpecID,
				ValueID: item.ValueID,
			})
		}
		out = append(out, productapp.SKUInput{
			SpecItems: items,
			Price:     s.Price,
			Stock:     s.Stock,
		})
	}
	return out
}
