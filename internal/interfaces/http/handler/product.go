package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	productapp "github.com/wsc-zz/service/internal/application/product"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

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
	SpecID    uint   `json:"specId" binding:"required"`
	ValueID   uint   `json:"valueId" binding:"required"`
	SpecName  string `json:"specName" binding:"required"`  // 维度名快照
	ValueName string `json:"valueName" binding:"required"` // 值名快照
}

type SkU struct {
	SKUCode   string        `json:"skuCode" binding:"required"` // 商品规格编码
	SpecItems []specItemReq `json:"specItems" binding:"required,min=1,dive"`
	Price     int64         `json:"price" binding:"required,gt=0"` // 价格
	Stock     int           `json:"stock" binding:"gte=0"`         // 库存
}

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
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, resp)
}

func toSKUInputs(skus []SkU) []productapp.SKUInput {
	out := make([]productapp.SKUInput, 0, len(skus))
	for _, s := range skus {
		items := make([]productapp.SpecItemInput, 0, len(s.SpecItems))
		for _, item := range s.SpecItems {
			items = append(items, productapp.SpecItemInput{
				SpecID:    item.SpecID,
				ValueID:   item.ValueID,
				SpecName:  item.SpecName,
				ValueName: item.ValueName,
			})
		}
		out = append(out, productapp.SKUInput{
			SKUCode:   s.SKUCode,
			SpecItems: items,
			Price:     s.Price,
			Stock:     s.Stock,
		})
	}
	return out
}
