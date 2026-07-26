package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	categoryapp "github.com/wsc-zz/service/internal/application/category"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

type CategoryHandler struct {
	categorySvc *categoryapp.Service
}

func NewCategoryHandler(categorySvc *categoryapp.Service) *CategoryHandler {
	return &CategoryHandler{
		categorySvc: categorySvc,
	}
}

type CreateReq struct {
	Name     string `json:"name" binding:"required"`
	ParentID uint   `json:"parentId" `
	Sort     int    `json:"sort" default:"0"`
}

// CreateCategory 创建分类
// @Summary      创建分类
// @Description  创建一个新的商品分类；同名分类已存在时返回错误
// @Tags         分类
// @Accept       json
// @Produce      json
// @Param        request  body      CreateReq                  true  "分类信息"
// @Success      200      {object}  response.Response{data=categoryapp.CategoryDto}
// @Failure      400      {object}  response.Response  "参数校验失败"
// @Failure      500      {object}  response.Response  "服务器内部错误"
// @Router       /Category/Create [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	var req CreateReq
	// 绑定参数
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}

	in := categoryapp.CreateCategoryInput{
		Name:     req.Name,
		ParentID: req.ParentID,
		Sort:     req.Sort,
	}
	resp, err := h.categorySvc.Create(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, resp)
}

type UpdateReq struct {
	Name     *string `json:"name"`
	ParentID *uint   `json:"parentId"`
	Sort     *int    `json:"sort"`
}

// Update 更新分类
// @Summary      更新分类
// @Description  根据分类ID更新分类信息，支持部分更新（未传字段不修改）
// @Tags         分类
// @Accept       json
// @Produce      json
// @Param        id        path      int                        true  "分类ID"
// @Param        request   body      UpdateReq                  true  "分类信息（均为可选字段）"
// @Success      200       {object}  response.Response{data=categoryapp.CategoryDto}
// @Failure      400       {object}  response.Response  "参数校验失败"
// @Failure      500       {object}  response.Response  "服务器内部错误"
// @Router       /category/{id} [put]
func (h *CategoryHandler) Update(c *gin.Context) {
	// 从动态路由 :id 读取分类ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "分类ID格式错误")
		return
	}

	var req UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}

	in := categoryapp.UpdateCategoryInput{
		CategoryID: uint(id),
		Name:       req.Name,
		ParentID:   req.ParentID,
		Sort:       req.Sort,
	}
	resp, err := h.categorySvc.Update(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, resp)
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "分类ID格式错误")
		return
	}
	err = h.categorySvc.Delete(c.Request.Context(), uint(id))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, nil)
}
