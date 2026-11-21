package merchantapp

import (
	"context"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
)

// Review 审核:approve=true 走通过三连(事务),false 走驳回(理由必填)。
// 返回 ErrInvalidStatusTransition = 别人先审了(并发败北);返回 nil = 本请求生效。
func (s *Service) Review(ctx context.Context, reviewerID, appID uint, approve bool, reason string) error {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		return err
	}
	_ = reviewerID // 审计留痕(谁审的)是阶段五的进阶,教学版先不落库
	if approve {
		return s.reviewApprove(ctx, app)
	}
	// 规则先行:空理由的驳回在领域层就拒绝;真正落库是仓储的条件更新(带理由+审核时间)
	if err := app.Reject(reason); err != nil {
		return err
	}
	changed, err := s.appRepo.Reject(ctx, appID, reason)
	if err != nil {
		return err
	}
	if !changed {
		return domainmerchant.ErrInvalidStatusTransition // 并发败北:别人先审了
	}
	return nil
}

// reviewApprove 通过分支:同一事务内四步,全有或全无。
// ① 条件更新抢锁:两个管理员同时点"通过",只有一人生效;
//
//	没抢到的那个人,事务回滚,商户不会建两次——幂等锚点是申请单状态本身。
//
// ② 用申请单信息建商户(出生即营业中)。
// ③ 授予店主成员(role=owner):N:N 下归属的事实源是成员表,
//    店与店主同事务互相指认,不存在"有店无主"的孤儿状态。
// ④ 回填 merchant_id,凭证与结果从此互相指认。
func (s *Service) reviewApprove(ctx context.Context, app *domainmerchant.MerchantApplication) error {
	return s.runner.RunMerchant(ctx, func(appRepo domainmerchant.ApplicationRepository, mRepo domainmerchant.MerchantRepository, memberRepo domainmerchant.MemberRepository) error {
		changed, err := appRepo.Approve(ctx, app.ID)
		if err != nil {
			return err
		}
		if !changed {
			return domainmerchant.ErrInvalidStatusTransition // 并发败北:别人先审了,整个事务回滚
		}
		m, err := domainmerchant.NewMerchant(app.ShopName, "", "")
		if err != nil {
			return err
		}
		if err := mRepo.Save(ctx, m); err != nil {
			// 店铺名撞唯一索引会走到这里:整个事务回滚,申请单回到"待审核",
			// 管理员看到报错"店铺名已被占用",处理方式是驳回并让用户换名。
			// 想清楚:为什么申请单不会卡在"已通过但没商户"?——因为四步同事务
			return domainmerchant.ErrShopNameTaken
		}
		if err := memberRepo.Save(ctx, domainmerchant.NewOwnerMember(app.UserID, m.MerchantID)); err != nil {
			return err
		}
		return appRepo.BindMerchant(ctx, app.ID, m.MerchantID)
	})
}

// ListApplications 申请单列表(管理端,时间倒序)。教学版全量,分页进阶。
func (s *Service) ListApplications(ctx context.Context) ([]ApplicationDTO, error) {
	apps, err := s.appRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	dtos := make([]ApplicationDTO, 0, len(apps))
	for i := range apps {
		dtos = append(dtos, toApplicationDTO(&apps[i]))
	}
	return dtos, nil
}
