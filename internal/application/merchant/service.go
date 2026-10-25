package merchantapp

import (
	"context"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
)

// TxRunner 事务端口(应用层定义契约,infrastructure/persistence/transaction 提供)。
type TxRunner interface {
	// RunMerchant 在事务内执行 fn,fn 收到与事务绑定的申请单/商户仓储
	RunMerchant(ctx context.Context, fn func(domainmerchant.ApplicationRepository, domainmerchant.MerchantRepository) error) error
}

// Service 商户入驻应用服务:编排申请提交、审核、店铺管理用例。
// 普通读和单步写用普通仓储,审核通过才走 runner——事务是资源,按需用。
type Service struct {
	appRepo domainmerchant.ApplicationRepository
	mRepo   domainmerchant.MerchantRepository
	runner  TxRunner
}

func NewService(appRepo domainmerchant.ApplicationRepository, mRepo domainmerchant.MerchantRepository, runner TxRunner) *Service {
	return &Service{appRepo: appRepo, mRepo: mRepo, runner: runner}
}

// Submit 提交入驻申请。
// 幂等:同用户已有"待审核"申请 → ErrAlreadyPending(重复点击/网络重试安全);
// 已有店铺 → ErrAlreadyMerchant。历史被驳回的申请不拦截,允许重新提交(新建,旧单只读)。
// 两个前置检查只是友好报错,真正的防线是 merchants.user_id 唯一索引(并发兜底)。
func (s *Service) Submit(ctx context.Context, userID uint, in SubmitInput) (*ApplicationDTO, error) {
	pending, err := s.appRepo.FindPendingByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if pending != nil {
		return nil, domainmerchant.ErrAlreadyPending
	}
	m, err := s.mRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if m != nil {
		return nil, domainmerchant.ErrAlreadyMerchant
	}
	app, err := domainmerchant.NewMerchantApplication(userID, in.ShopName, in.ContactName, in.ContactPhone, in.LicenseURL, in.IDCardURL)
	if err != nil {
		return nil, err
	}
	if err := s.appRepo.Save(ctx, app); err != nil {
		return nil, err
	}
	dto := toApplicationDTO(app)
	return &dto, nil
}

// MyApplications 我的申请历史(时间倒序)。归属即过滤条件,天然越权免疫——
// 没有"查别人申请"的接口,这就是归属设计的成果。
func (s *Service) MyApplications(ctx context.Context, userID uint) ([]ApplicationDTO, error) {
	apps, err := s.appRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	dtos := make([]ApplicationDTO, 0, len(apps))
	for i := range apps {
		dtos = append(dtos, toApplicationDTO(&apps[i]))
	}
	return dtos, nil
}

// Withdraw 撤回自己的待审核申请。
// 归属校验在应用层(查出来比对 userID),并发安全交给仓储条件更新(user_id 一起进 WHERE)。
func (s *Service) Withdraw(ctx context.Context, userID, appID uint) error {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		return err
	}
	if app.UserID != userID {
		// 别人的申请单:和"不存在"同错误,不泄露存在性(与 address.getOwned 同款)
		return domainmerchant.ErrApplicationNotFound
	}
	if err := app.Withdraw(); err != nil {
		return err // 领域管规则:非待审核状态不能撤
	}
	changed, err := s.appRepo.Withdraw(ctx, appID, userID)
	if err != nil {
		return err
	}
	if !changed {
		return domainmerchant.ErrInvalidStatusTransition // 刚被人审过/撤过,条件更新没抢到
	}
	return nil
}
