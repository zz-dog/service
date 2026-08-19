package handler

import (
	"github.com/gin-gonic/gin"
	specapp "github.com/wsc-zz/service/internal/application/spec"
)

type SpecHandler struct {
	specSvc *specapp.Service
}

func NewSpecHandler(specSvc *specapp.Service) *SpecHandler {
	return &SpecHandler{
		specSvc: specSvc,
	}
}

type CreateInput struct {
	Name string `json:"name" binding:"required"`
	Sort int    `json:"sort" binding:"required"`
}

func (h *SpecHandler) Create(c *gin.Context) {
	var req CreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	in := specapp.SpecInput{
		Name: req.Name,
		Sort: req.Sort,
	}
	spec, err := h.specSvc.Create(c, in)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": spec})
}
