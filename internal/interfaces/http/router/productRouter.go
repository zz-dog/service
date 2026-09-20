package router

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	categoryapp "github.com/wsc-zz/service/internal/application/category"
	productapp "github.com/wsc-zz/service/internal/application/product"
	categorypo "github.com/wsc-zz/service/internal/infrastructure/persistence/category"
	productpo "github.com/wsc-zz/service/internal/infrastructure/persistence/product"
	specpo "github.com/wsc-zz/service/internal/infrastructure/persistence/spec"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"

	categoryCache "github.com/wsc-zz/service/internal/infrastructure/cache/redis"
)

func registerProductRouter(r *gin.RouterGroup) {
	categoryRepo := categorypo.NewCategoryRepository(global.DB)
	categorySpecRepo := categorypo.NewCategorySpecRepository(global.DB)

	specRepo := specpo.NewSpecRepository(global.DB)

	categoryCache := categoryCache.NewCategoryCache(global.RedisClient)
	categorySvc := categoryapp.NewService(categoryRepo, specRepo, categorySpecRepo, categoryCache)

	productRepo := productpo.NewProductRepository(global.DB)
	productSvc := productapp.NewService(productRepo, categorySpecDirectory{svc: categorySvc})
	h := handler.NewProductHandler(productSvc)
	productApi := r.Group("/product")
	{
		productApi.POST("/create", h.Create)
		productApi.POST("/update", h.Update)
		productApi.POST("/list", h.List)
	}
}

type categorySpecDirectory struct {
	svc *categoryapp.Service
}

func (d categorySpecDirectory) SpecsForCategory(ctx context.Context, categoryID uint) ([]*productapp.CategorySpecView, error) {
	dtos, err := d.svc.ListSpecs(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	views := make([]*productapp.CategorySpecView, 0, len(dtos))
	for _, dto := range dtos {
		v := &productapp.CategorySpecView{
			SpecID:   dto.SpecID,
			SpecName: dto.SpecName,
		}
		for _, val := range dto.Values {
			v.Values = append(v.Values, productapp.SpecValueView{ValueID: val.ValueID, Name: val.Name})
		}
		views = append(views, v)
	}
	return views, nil
}
