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

// newProductHandler 商品应用服务的装配:依赖分类应用服务(规格目录),
// 管理端/C 端两个注册函数共用一套
func newProductHandler() *handler.ProductHandler {
	categoryRepo := categorypo.NewCategoryRepository(global.DB)
	categorySpecRepo := categorypo.NewCategorySpecRepository(global.DB)

	specRepo := specpo.NewSpecRepository(global.DB)

	categoryCache := categoryCache.NewCategoryCache(global.RedisClient)
	categorySvc := categoryapp.NewService(categoryRepo, specRepo, categorySpecRepo, categoryCache)

	productRepo := productpo.NewProductRepository(global.DB)
	productSvc := productapp.NewService(productRepo, categorySpecDirectory{svc: categorySvc})
	return handler.NewProductHandler(productSvc)
}

// registerProductAdminRoutes 商品管理端写接口:挂在 /api/catalog/admin/product/**,
// 由 admin.go 的 JWTAuth+AdminAuth 组统一收口(鉴权挂 group 一处收口,
// 不再沿用旧的"每条路由单独挂 JWT"散装写法)
func registerProductAdminRoutes(r *gin.RouterGroup) {
	h := newProductHandler()

	productApi := r.Group("/product")

	{
		productApi.POST("/create", h.Create)
		productApi.POST("/update", h.Update)
		productApi.DELETE("/delete/:id", h.Delete)
		productApi.POST("/inventory/add", h.AddInventory)
		productApi.POST("/sku/update", h.UpdateSKU)
		productApi.DELETE("/sku/delete/:productId/:skuCode", h.DeleteSKU)
	}
}

// registerProductRoutes 商品 C 端读接口:保持原路径(公开无鉴权,与现状一致);
// status 筛选强制只查上架等行为收口在教程阶段四
func registerProductRoutes(r *gin.RouterGroup) {
	h := newProductHandler()

	productApi := r.Group("/product")

	{
		productApi.GET("/list", h.List)
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
