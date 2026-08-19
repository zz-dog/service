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

type SkU struct {
	Spec  string `json:"spec"`  // 规格
	Price int64  `json:"price"` // 价格
	Stock int    `json:"stock"` // 库存
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req createProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
	}
}
