package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	categoryapp "github.com/wsc-zz/service/internal/application/category"
	productapp "github.com/wsc-zz/service/internal/application/product"
	categorypo "github.com/wsc-zz/service/internal/infrastructure/persistence/category"
	productpo "github.com/wsc-zz/service/internal/infrastructure/persistence/product"
	specpo "github.com/wsc-zz/service/internal/infrastructure/persistence/spec"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	"github.com/wsc-zz/service/internal/interfaces/rpc/merchantrpc"

	categoryCache "github.com/wsc-zz/service/internal/infrastructure/cache/redis"
)

// newProductHandler 商品应用服务的装配:依赖分类应用服务(规格目录)
// 和商户网关(商户身份推导),管理端/商户端/C 端三个注册函数共用一套
func newProductHandler() (*handler.ProductHandler, error) {
	categoryRepo := categorypo.NewCategoryRepository(global.DB)
	categorySpecRepo := categorypo.NewCategorySpecRepository(global.DB)

	specRepo := specpo.NewSpecRepository(global.DB)

	categoryCache := categoryCache.NewCategoryCache(global.RedisClient)
	categorySvc := categoryapp.NewService(categoryRepo, specRepo, categorySpecRepo, categoryCache)

	// 商户网关:商户端接口靠它把 token 里的 userID 推导成 merchantId,
	// 归属校验全走服务端,商户身份永不信请求参数
	merchantGw, err := merchantrpc.NewClient(global.Conf.Merchant.RpcAddr, 3*time.Second)
	if err != nil {
		return nil, err
	}

	productRepo := productpo.NewProductRepository(global.DB)
	productSvc := productapp.NewService(productRepo, categorySpecDirectory{svc: categorySvc}, merchantGw)
	return handler.NewProductHandler(productSvc), nil
}

// registerProductAdminRoutes 商品管理端接口:挂在 /api/catalog/admin/product/**,
// 由 router.go 的 JWTAuth+AdminAuth 组统一收口(鉴权挂 group 一处收口,
// 不再沿用旧的"每条路由单独挂 JWT"散装写法)
func registerProductAdminRoutes(r *gin.RouterGroup) {
	h, err := newProductHandler()
	if err != nil {
		panic(err) // 装配失败服务起不来,快速失败(与 DB 初始化失败同语义)
	}

	productApi := r.Group("/product")

	{
		productApi.POST("/create", h.Create)
		productApi.POST("/update", h.Update)
		productApi.DELETE("/delete/:id", h.Delete)
		productApi.POST("/inventory/add", h.AddInventory)
		productApi.POST("/sku/update", h.UpdateSKU)
		productApi.DELETE("/sku/delete/:productId/:skuCode", h.DeleteSKU)
		// 审核新增:队列/审核/违规下架——商户商品的合规闭环
		productApi.GET("/pending", h.AdminPendingList)
		productApi.POST("/:id/review", h.AdminReview)
		productApi.POST("/:id/takedown", h.AdminTakedown)
	}
}

// registerMerchantProductRoutes 商品商户端接口:挂在 /api/catalog/merchant/product/**。
// 商户商品和平台商品是同一张表、同一个聚合,只拆路由不拆服务——
// 归属校验/身份推导在应用层收口,路由层只负责"这一组都是登录商户在调"。
// 注意路由模型:/:id 段和 /my 是同级兄弟路径,gin 允许静态段与参数段共存
// (与商户服务的 /application/my + /application/:id/withdraw 同款证据)
func registerMerchantProductRoutes(r *gin.RouterGroup) {
	h, err := newProductHandler()
	if err != nil {
		panic(err)
	}

	productApi := r.Group("/product")

	{
		productApi.POST("", h.MerchantPublish)              // 发布(进待审核)
		productApi.GET("/my", h.MyProducts)                 // 我的商品(全状态)
		productApi.PUT("/:id", h.MerchantUpdate)            // 修改(内容变更回待审核)
		productApi.DELETE("/:id", h.MerchantDelete)         // 删除(仅下架/驳回可删)
		productApi.POST("/:id/on-shelf", h.MerchantOnShelf) // 上架(免重审)
		productApi.POST("/:id/off-shelf", h.MerchantOffShelf)
		productApi.POST("/:id/inventory", h.MerchantAddInventory) // 补货:经营数据,不回审核
		productApi.PUT("/:id/sku", h.MerchantUpdateSKU)           // 调价/换图:同上
		productApi.DELETE("/:id/sku/:skuCode", h.MerchantDeleteSKU)
	}
}

// registerProductRoutes 商品 C 端读接口:保持原路径(公开无鉴权,与现状一致);
// status 筛选已被参数白名单收掉——强制只查已上架,想看别的状态走商户端/管理端接口
func registerProductRoutes(r *gin.RouterGroup) {
	h, err := newProductHandler()
	if err != nil {
		panic(err)
	}

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
