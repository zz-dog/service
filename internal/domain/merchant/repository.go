package merchant

import "context"

// ApplicationRepository 申请单仓储
type ApplicationRepository interface {
	Save(ctx context.Context, a *MerchantApplication) error
	FindByID(ctx context.Context, id uint) (*MerchantApplication, error)
	// FindPendingByUser 找该用户"待审核"的申请单;没有返回 (nil, nil)——防重复提交的判定依据
	FindPendingByUser(ctx context.Context, userID uint) (*MerchantApplication, error)
	// ListByUser 某用户的申请历史,时间倒序——"我的申请"页
	ListByUser(ctx context.Context, userID uint) ([]MerchantApplication, error)
	// ListAll 全部申请单,时间倒序——管理端审核列表(教学版全量,分页进阶)
	ListAll(ctx context.Context) ([]MerchantApplication, error)
	// Approve 条件更新 待审核→已通过;返回本请求是否生效(并发审核只有一方 changed==true)
	Approve(ctx context.Context, id uint) (bool, error)
	// Reject 条件更新 待审核→已驳回(带理由);返回语义同 Approve
	Reject(ctx context.Context, id uint, reason string) (bool, error)
	// Withdraw 条件更新 待审核→已撤回;user_id 一起进 WHERE,归属与状态一次原子裁决
	Withdraw(ctx context.Context, id, userID uint) (bool, error)
	// BindMerchant 回填 merchant_id,审计关联;只在已通过状态做
	BindMerchant(ctx context.Context, id uint, merchantID uint) error
}

// MerchantRepository 商户仓储
type MerchantRepository interface {
	Save(ctx context.Context, m *Merchant) error
	FindByID(ctx context.Context, id uint) (*Merchant, error)
	// FindByUserID 一人一店:按归属用户查商户;没有返回 (nil, nil)
	FindByUserID(ctx context.Context, userID uint) (*Merchant, error)
	// UpdateStatus 条件更新 from→to;停业/恢复/封禁共用,状态机迁移由调用方(领域方法)裁决合法性
	UpdateStatus(ctx context.Context, id uint, from, to int) (bool, error)
	// UpdateInfo 更新店铺资料
	UpdateInfo(ctx context.Context, id uint, name, logo, desc string) error
}
