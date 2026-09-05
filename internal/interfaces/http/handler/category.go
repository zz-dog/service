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
// @Router       /category/create [post]
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

// Delete 删除分类
// @Summary      删除分类
// @Description  根据分类ID删除分类
// @Tags         分类
// @Produce      json
// @Param        id  path  int  true  "分类ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response  "分类ID格式错误"
// @Failure      500  {object}  response.Response  "服务器内部错误"
// @Router       /category/{id} [delete]
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

// FindAll 查询所有分类
// @Summary      查询所有分类
// @Description  返回全部分类列表（不分页）
// @Tags         分类
// @Produce      json
// @Success      200  {object}  response.Response{data=[]categoryapp.CategoryDto}
// @Failure      500  {object}  response.Response  "服务器内部错误"
// @Router       /category/findAll [get]
func (h *CategoryHandler) FindAll(c *gin.Context) {

	cs, err := h.categorySvc.FindAll(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, cs)

}

type BindSpecReq struct {
	CategoryID uint `json:"categoryId" binding:"required"` // 分类ID
	SpecID     uint `json:"specId" binding:"required"`     // 规格ID
	Sort       int  `json:"sort"`                          // 排序
	Required   bool `json:"required"`                      // 是否必填
}

// BindSpec 绑定规格到分类
// @Summary      绑定规格到分类
// @Description  建立分类与规格维度的多对多绑定；重复绑定返回错误
// @Tags         分类
// @Accept       json
// @Produce      json
// @Param        request  body      BindSpecReq  true  "绑定信息"
// @Success      200      {object}  response.Response{data=categoryapp.CategorySpecDto}
// @Router       /category/bindSpec [post]
func (h *CategoryHandler) BindSpec(c *gin.Context) {
	var req BindSpecReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}

	in := categoryapp.BindSpecInput{
		CategoryID: req.CategoryID,
		SpecID:     req.SpecID,
		Sort:       req.Sort,
		Required:   req.Required,
	}
	resp, err := h.categorySvc.BindSpec(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, resp)
}

// UnbindSpec 解除分类与规格的绑定
// @Summary      解除规格绑定
// @Description  删除分类与规格维度的绑定关系
// @Tags         分类
// @Produce      json
// @Param        categoryId  query  int  true  "分类ID"
// @Param        specId      query  int  true  "规格ID"
// @Success      200  {object}  response.Response
// @Router       /category/unbindSpec [delete]
func (h *CategoryHandler) UnbindSpec(c *gin.Context) {
	categoryID, err := strconv.ParseUint(c.Query("categoryId"), 10, 64)
	if err != nil || categoryID == 0 {
		response.BadRequest(c, http.StatusBadRequest, "分类ID格式错误")
		return
	}
	specID, err := strconv.ParseUint(c.Query("specId"), 10, 64)
	if err != nil || specID == 0 {
		response.BadRequest(c, http.StatusBadRequest, "规格ID格式错误")
		return
	}

	in := categoryapp.UnbindSpecInput{CategoryID: uint(categoryID), SpecID: uint(specID)}
	if err := h.categorySvc.UnbindSpec(c.Request.Context(), in); err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, nil)
}

// ListSpecs 查询分类下绑定的规格
// @Summary      查询分类绑定的规格
// @Description  返回分类下全部绑定的规格维度（含规格值、排序、是否必选）
// @Tags         分类
// @Produce      json
// @Param        id  path  int  true  "分类ID"
// @Success      200  {object}  response.Response{data=[]categoryapp.CategorySpecDto}
// @Router       /category/findSpecs/{id} [get]
func (h *CategoryHandler) ListSpecs(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, http.StatusBadRequest, "分类ID格式错误")
		return
	}

	specs, err := h.categorySvc.ListSpecs(c.Request.Context(), uint(id))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, specs)
}
