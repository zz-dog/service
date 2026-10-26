package productapp

import (
	"context"

	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

// Review 管理员审核:approve=true 走通过,false 走驳回(理由必填)。
// 先加载聚合让领域裁决状态合法性,再条件更新落库;
// 条件更新败北 = 审核期间商品状态被商户动了(改内容回不了已迁移的状态),
// 报错让管理员刷新重审,看到的就是最新版本。
// 教学妥协:审核判的是"加载那一刻的快照",商户在队列里原地改材料是允许的;
// 严格方案是内容版本号乐观锁(教程进阶项),这里用状态条件更新近似。
func (s *Service) Review(ctx context.Context, productID uint, approve bool, reason string) error {
	p, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if approve {
		if err := p.Approve(); err != nil {
			return err
		}
		changed, err := s.repo.UpdateStatus(ctx, productID, domainproduct.StatusPendingReview, domainproduct.StatusOnShelf)
		if err != nil {
			return err
		}
		if !changed {
			return domainproduct.ErrInvalidStatusTransition
		}
		return nil
	}
	if err := p.Reject(reason); err != nil {
		return err
	}
	changed, err := s.repo.Reject(ctx, productID, reason)
	if err != nil {
		return err
	}
	if !changed {
		return domainproduct.ErrInvalidStatusTransition
	}
	return nil
}

// Takedown 管理员违规下架:先加载确认存在(404 语义),领域裁决(仅已上架可处置、理由必填),
// 再条件更新 1→5 落库。败北说明状态刚变,刷新即见最新。
func (s *Service) Takedown(ctx context.Context, productID uint, reason string) error {
	p, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if err := p.Takedown(reason); err != nil {
		return err
	}
	changed, err := s.repo.Takedown(ctx, productID, reason)
	if err != nil {
		return err
	}
	if !changed {
		return domainproduct.ErrInvalidStatusTransition
	}
	return nil
}

// PendingList 待审核列表(管理端审核队列):状态条件写死在应用层,不透传调用方。
func (s *Service) PendingList(ctx context.Context, q domainproduct.ListQuery) (ProductListResult, error) {
	pending := domainproduct.StatusPendingReview
	q.Status = &pending
	products, total, err := s.repo.List(ctx, q)
	if err != nil {
		return ProductListResult{}, err
	}
	return ProductListResult{List: toProductDTOs(products), Total: total}, nil
}
