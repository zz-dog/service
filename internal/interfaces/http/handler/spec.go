package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	specapp "github.com/wsc-zz/service/internal/application/spec"
	domainSpec "github.com/wsc-zz/service/internal/domain/spec"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

type SpecHandler struct {
	specSvc *specapp.Service
}

func NewSpecHandler(specSvc *specapp.Service) *SpecHandler {
	return &SpecHandler{
		specSvc: specSvc,
	}
}

type SpecValueInput struct {
	Name string `json:"name" binding:"required"`
	Sort int    `json:"sort"`
}

type CreateInput struct {
	Name   string           `json:"name" binding:"required"`
	Sort   int              `json:"sort" `
	Values []SpecValueInput `json:"values"`
}

// Create 创建规格
// @Summary      创建规格
// @Description  创建一个新的规格维度（如颜色、尺寸等），可同时传入规格值列表一次性创建；同名规格已存在时返回错误
// @Tags         规格
// @Accept       json
// @Produce      json
// @Param        request  body      CreateInput  true  "规格信息"
// @Success      200      {object}  response.Response{data=specapp.SpecDTO}
// @Failure      400      {object}  response.Response  "参数校验失败"
// @Failure      500      {object}  response.Response  "服务器内部错误"
// @Router       /spec/register [post]
func (h *SpecHandler) Create(c *gin.Context) {
	var req CreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	values := make([]specapp.SpecValueInput, 0, len(req.Values))
	for _, v := range req.Values {
		values = append(values, specapp.SpecValueInput{
			Name: v.Name,
			Sort: v.Sort,
		})
	}
	in := specapp.SpecInput{
		Name:   req.Name,
		Sort:   req.Sort,
		Values: values,
	}
	spec, err := h.specSvc.Create(c, in)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, spec)
}

type UpdateValueInput struct {
	ValueID uint   `json:"specValueId"` // 规格值ID，0 表示新增
	Name    string `json:"name" binding:"required"`
	Sort    int    `json:"sort"`
}

type UpdateInput struct {
	Name   string             `json:"name" binding:"required"`
	Sort   int                `json:"sort"`
	Values []UpdateValueInput `json:"values"`
}

// Update 更新规格
// @Summary      整包更新规格
// @Description  根据规格ID更新规格维度及规格值。Values 须传该规格的完整值列表：带ID的值被更新、不带ID的值被新增、库中缺席的值被删除；单事务保证一致
// @Tags         规格
// @Accept       json
// @Produce      json
// @Param        id       path  int          true  "规格ID"
// @Param        request  body  UpdateInput  true  "规格信息"
// @Success      200      {object}  response.Response{data=specapp.SpecDTO}
// @Failure      400      {object}  response.Response  "参数校验失败"
// @Failure      500      {object}  response.Response  "服务器内部错误"
// @Router       /spec/update/{id} [put]
func (h *SpecHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "ID格式错误")
		return
	}
	var req UpdateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	values := make([]specapp.UpdateSpecValueInput, 0, len(req.Values))
	for _, v := range req.Values {
		values = append(values, specapp.UpdateSpecValueInput{
			ValueID: v.ValueID,
			Name:    v.Name,
			Sort:    v.Sort,
		})
	}
	spec, err := h.specSvc.Update(c, uint(id), specapp.UpdateSpecInput{
		Name:   req.Name,
		Sort:   req.Sort,
		Values: values,
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, spec)
}

// Delete 删除规格
// @Summary      删除规格
// @Description  根据规格ID删除规格维度
// @Tags         规格
// @Produce      json
// @Param        id  path  int  true  "规格ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response  "规格ID格式错误"
// @Failure      500  {object}  response.Response  "服务器内部错误"
// @Router       /spec/delete/{id} [delete]
func (h *SpecHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "ID格式错误")
		return
	}
	err = h.specSvc.Delete(c, uint(id))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, nil)
}

// FindByID 根据ID查询规格
// @Summary      根据ID查询规格
// @Description  根据规格ID查询规格详情（含规格值列表）
// @Tags         规格
// @Produce      json
// @Param        id  path  int  true  "规格ID"
// @Success      200  {object}  response.Response{data=specapp.SpecDTO}
// @Failure      400  {object}  response.Response  "规格ID格式错误"
// @Failure      500  {object}  response.Response  "服务器内部错误"
// @Router       /spec/findById/{id} [get]
func (h *SpecHandler) FindByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "ID格式错误")
		return
	}
	spec, err := h.specSvc.FindByID(c, uint(id))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
	}
	response.Success(c, spec)
}

// FindByName 根据名称查询规格
// @Summary      根据名称查询规格
// @Description  根据规格名称精确查询规格详情（含规格值列表）
// @Tags         规格
// @Produce      json
// @Param        name  path  string  true  "规格名称"
// @Success      200  {object}  response.Response{data=specapp.SpecDTO}
// @Failure      500  {object}  response.Response  "服务器内部错误"
// @Router       /spec/findByName/{name} [get]
func (h *SpecHandler) FindByName(c *gin.Context) {
	name := c.Param("name")
	spec, err := h.specSvc.FindByName(c, name)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, spec)
}

type ListInput struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"pageSize" binding:"omitempty,min=1,max=100"`
	Name     string `form:"name"`
	SpecID   uint   `form:"specId"`
}

// List 分页查询规格列表
// @Summary      分页查询规格列表
// @Description  分页查询规格维度列表，支持按名称模糊搜索；返回列表与总数
// @Tags         规格
// @Produce      json
// @Param        page     query  int     false  "页码（默认1）"
// @Param        pageSize query  int     false  "每页数量（默认10，最大100）"
// @Param        name     query  string  false  "规格名称（模糊搜索）"
// @Param        specId   query  int     false  "规格ID"
// @Success      200  {object}  response.Response{data=specapp.ListDto}
// @Failure      400  {object}  response.Response  "参数校验失败"
// @Failure      500  {object}  response.Response  "服务器内部错误"
// @Router       /spec/list [get]
func (h *SpecHandler) List(c *gin.Context) {
	var req ListInput
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
	}
	result, err := h.specSvc.List(c, domainSpec.SpecListQuery{
		Page:     req.Page,
		PageSize: req.PageSize,
		Name:     req.Name,
		SpecID:   req.SpecID,
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.Success(c, result)
}
