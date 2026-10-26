package productapp

import (
	"context"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

// deriveShopID 用 JWT 的 userId 经 MerchantGateway 推导归属商户。
// merchantId=0 视为无店(ErrMerchantNotFound)。停业/封禁的店仍可编辑自己的商品——
// 停业挡的是"卖"(发布时已校验),不是"改";归属与经营状态是两件事,不混着校验。
func (s *Service) deriveShopID(ctx context.Context, userID uint) (uint, error) {
	m, err := s.merchantGw.GetMerchantByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if m.MerchantID == 0 {
		return 0, domainmerchant.ErrMerchantNotFound
	}
	return m.MerchantID, nil
}

// ensureOwner 归属校验:别人的商品与不存在的商品同错误,不泄露存在性
// (与 merchant 服务的 Withdraw 归属校验同款纪律)。
func ensureOwner(p *domainproduct.Product, merchantID uint) error {
	if p.MerchantID != merchantID {
		return domainproduct.ErrProductNotFound
	}
	return nil
}

// MerchantPublish 商户发布商品。
// merchantId 服务端推导,body 传什么都不采信——伪造 merchantId = 把商品挂进别人的店。
// 出生即待审核,审核通过才对 C 端可见;营业中才能发布(停业/封禁被拒)。
func (s *Service) MerchantPublish(ctx context.Context, userID uint, in CreateProductInput) (*ProductDTO, error) {
	m, err := s.merchantGw.GetMerchantByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if m.MerchantID == 0 {
		return nil, domainmerchant.ErrMerchantNotFound
	}
	if m.Status != int(domainmerchant.MStatusOperating) {
		return nil, domainmerchant.ErrMerchantNotOperating
	}
	p, err := s.compose(ctx, func(skus []domainproduct.SKU) (*domainproduct.Product, error) {
		return domainproduct.NewMerchantProduct(m.MerchantID, in.CategoryID, in.Name, in.Desc, skus)
	}, in.CategoryID, in.SKUs, in.Medias)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	dto := toProductDTO(p)
	return &dto, nil
}

// MerchantUpdate 商户修改商品:内容变更一律回待审核(已在待审核则原地更新材料)。
// 归属校验先行;违规下架(终态)的商品连内容都改不了——处罚期内冻结。
// 已知竞态:加载与 Save 之间他人改状态,Save 会带旧状态覆盖——内容版本号乐观锁是进阶项。
func (s *Service) MerchantUpdate(ctx context.Context, userID uint, in UpdateProductInput) (*ProductDTO, error) {
	shopID, err := s.deriveShopID(ctx, userID)
	if err != nil {
		return nil, err
	}
	p, err := s.repo.FindByID(ctx, in.ProductID)
	if err != nil {
		return nil, err
	}
	if err := ensureOwner(p, shopID); err != nil {
		return nil, err
	}
	if err := p.UpdateInfo(in.CategoryID, in.Name, in.Desc); err != nil {
		return nil, err
	}
	if err := p.ReplaceMedias(toMedias(in.Medias)); err != nil {
		return nil, err
	}
	if p.Status != domainproduct.StatusPendingReview {
		if err := p.SubmitForReview(); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	dto := toProductDTO(p)
	return &dto, nil
}

// MerchantDelete 商户删除商品:仅已下架/已驳回可删(待审核还在队列里,违规下架是处罚凭证);软删。
func (s *Service) MerchantDelete(ctx context.Context, userID, productID uint) error {
	shopID, err := s.deriveShopID(ctx, userID)
	if err != nil {
		return err
	}
	p, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if err := ensureOwner(p, shopID); err != nil {
		return err
	}
	if !p.CanDelete() {
		return domainproduct.ErrInvalidStatusTransition
	}
	return s.repo.Delete(ctx, productID)
}

// MerchantOnShelf 商户上架:已下架 → 已上架,免重审。
// 下架是商户的经营决策,不是合规否决——自己下的架自己回。
// 领域裁决迁移合法性,条件更新落库;败北 = 状态刚被人动了,重新加载即见最新。
func (s *Service) MerchantOnShelf(ctx context.Context, userID, productID uint) error {
	shopID, err := s.deriveShopID(ctx, userID)
	if err != nil {
		return err
	}
	p, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if err := ensureOwner(p, shopID); err != nil {
		return err
	}
	if err := p.OnShelf(); err != nil {
		return err
	}
	changed, err := s.repo.UpdateStatus(ctx, productID, domainproduct.StatusOffShelf, domainproduct.StatusOnShelf)
	if err != nil {
		return err
	}
	if !changed {
		return domainproduct.ErrInvalidStatusTransition
	}
	return nil
}

// MerchantOffShelf 商户下架:已上架 → 已下架。结构同上架,方向相反。
func (s *Service) MerchantOffShelf(ctx context.Context, userID, productID uint) error {
	shopID, err := s.deriveShopID(ctx, userID)
	if err != nil {
		return err
	}
	p, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if err := ensureOwner(p, shopID); err != nil {
		return err
	}
	if err := p.OffShelf(); err != nil {
		return err
	}
	changed, err := s.repo.UpdateStatus(ctx, productID, domainproduct.StatusOnShelf, domainproduct.StatusOffShelf)
	if err != nil {
		return err
	}
	if !changed {
		return domainproduct.ErrInvalidStatusTransition
	}
	return nil
}

// MerchantAddInventory 商户补货:归属校验后走原子加库存;经营数据,不回审核队列。
func (s *Service) MerchantAddInventory(ctx context.Context, userID, productID uint, skuCode string, qty int) error {
	shopID, err := s.deriveShopID(ctx, userID)
	if err != nil {
		return err
	}
	p, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if err := ensureOwner(p, shopID); err != nil {
		return err
	}
	return s.repo.RestockStock(ctx, productID, skuCode, qty)
}

// MerchantUpdateSKU 商户调价/换图:经营数据高频变更,不回审核队列——
// 审核资源花在合规维度(名称/描述/媒体/规格);"先低价过审再改价"靠价格监控治,不靠卡审核。
func (s *Service) MerchantUpdateSKU(ctx context.Context, userID, productID uint, skuCode string, price int64, picture string) error {
	shopID, err := s.deriveShopID(ctx, userID)
	if err != nil {
		return err
	}
	p, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if err := ensureOwner(p, shopID); err != nil {
		return err
	}
	if err := p.UpdateSKU(skuCode, price, picture); err != nil {
		return err
	}
	return s.repo.Save(ctx, p)
}

// MerchantDeleteSKU 商户删规格:仅已下架商品(与删除商品同门槛;
// 已下架商品回架前必然重审,删规格不需要再单独回队列)。
func (s *Service) MerchantDeleteSKU(ctx context.Context, userID, productID uint, skuCode string) error {
	shopID, err := s.deriveShopID(ctx, userID)
	if err != nil {
		return err
	}
	p, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if err := ensureOwner(p, shopID); err != nil {
		return err
	}
	if err := p.RemoveSKU(skuCode); err != nil {
		return err
	}
	return s.repo.Save(ctx, p)
}

// MyProducts 我的商品(全状态):归属即过滤条件,天然越权免疫——
// 没有"查别人商品"的入口,查询条件本身就是授权。
func (s *Service) MyProducts(ctx context.Context, userID uint, q domainproduct.ListQuery) (ProductListResult, error) {
	shopID, err := s.deriveShopID(ctx, userID)
	if err != nil {
		return ProductListResult{}, err
	}
	q.MerchantID = shopID
	products, total, err := s.repo.List(ctx, q)
	if err != nil {
		return ProductListResult{}, err
	}
	return ProductListResult{List: toProductDTOs(products), Total: total}, nil
}
