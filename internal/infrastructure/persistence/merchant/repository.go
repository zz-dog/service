package merchantpo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
)

// MerchantApplicationRepository 申请单仓储。
// 状态迁移一律"条件更新":UPDATE ... WHERE status=旧状态,让状态机成为并发裁决者——
// "查出来改完存回去"在两个管理员同时点按钮时是双生效,和支付回调同一招(阶段二核心)。
type MerchantApplicationRepository struct {
	db *gorm.DB
}

func NewMerchantApplicationRepository(db *gorm.DB) *MerchantApplicationRepository {
	return &MerchantApplicationRepository{db: db}
}

// Save 新建申请单(提交入驻)。领域对象先转 PO 再落库——两套结构,不许直接喂 GORM。
func (r MerchantApplicationRepository) Save(ctx context.Context, a *domainmerchant.MerchantApplication) error {
	return r.db.WithContext(ctx).Create(topo(a)).Error
}

func (r MerchantApplicationRepository) FindByID(ctx context.Context, id uint) (*domainmerchant.MerchantApplication, error) {
	var po MerchantApplicationPO
	err := r.db.WithContext(ctx).First(&po, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainmerchant.ErrApplicationNotFound
		}
		return nil, err
	}
	return todomain(&po), nil
}

// FindPendingByUser 找该用户"待审核"的申请单;没有返回 (nil, nil)——防重复提交的判定依据。
// 注意两点:① 内联参数 First(&po, userID) 走主键语义,按非主键查必须显式 Where;
// ② 过滤条件必须带 status=待审核,否则用户的历史单(已驳回)会把他永久卡在"重复提交"上。
func (r MerchantApplicationRepository) FindPendingByUser(ctx context.Context, userID uint) (*domainmerchant.MerchantApplication, error) {
	var po MerchantApplicationPO
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, int(domainmerchant.AppStatusPending)).
		First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return todomain(&po), nil
}

// ListByUser 某用户的申请历史,时间倒序——"我的申请"页。
// 归属即过滤条件,漏掉 user_id 就是全表泄露(别人身份证、营业执照全看见)。
func (r MerchantApplicationRepository) ListByUser(ctx context.Context, userID uint) ([]domainmerchant.MerchantApplication, error) {
	var pos []MerchantApplicationPO
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&pos).Error
	if err != nil {
		return nil, err
	}
	return todomains(pos), nil
}

// ListAll 全部申请单,时间倒序——管理端审核列表(教学版全量,分页进阶)
func (r MerchantApplicationRepository) ListAll(ctx context.Context) ([]domainmerchant.MerchantApplication, error) {
	var pos []MerchantApplicationPO
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&pos).Error
	if err != nil {
		return nil, err
	}
	return todomains(pos), nil
}

// Approve 条件更新 待审核→已通过,同时落审核时间:两个管理员同时点"通过",只有一人生效。
// 返回本请求是否生效;没生效不算错误,由调用方决定幂等还是回滚(和 MarkPaid 同骨架)。
func (r MerchantApplicationRepository) Approve(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Model(&MerchantApplicationPO{}).
		Where("id = ? AND status = ?", id, int(domainmerchant.AppStatusPending)).
		Updates(map[string]interface{}{
			"status":      int(domainmerchant.AppStatusApproved),
			"reviewed_at": time.Now(),
		})
	return result.RowsAffected == 1, result.Error
}

// Reject 条件更新 待审核→已驳回,理由与审核时间一并落库。返回语义同 Approve。
func (r MerchantApplicationRepository) Reject(ctx context.Context, id uint, reason string) (bool, error) {
	result := r.db.WithContext(ctx).Model(&MerchantApplicationPO{}).
		Where("id = ? AND status = ?", id, int(domainmerchant.AppStatusPending)).
		Updates(map[string]interface{}{
			"status":        int(domainmerchant.AppStatusRejected),
			"reject_reason": reason,
			"reviewed_at":   time.Now(),
		})
	return result.RowsAffected == 1, result.Error
}

// Withdraw 条件更新 待审核→已撤回。user_id 一起进 WHERE:归属校验和状态迁移一次原子完成。
func (r MerchantApplicationRepository) Withdraw(ctx context.Context, id, userID uint) (bool, error) {
	result := r.db.WithContext(ctx).Model(&MerchantApplicationPO{}).
		Where("id = ? AND user_id = ? AND status = ?", id, userID, int(domainmerchant.AppStatusPending)).
		Update("status", int(domainmerchant.AppStatusWithdraw))
	return result.RowsAffected == 1, result.Error
}

// BindMerchant 回填 merchant_id,审计关联;只在已通过状态做(条件更新防手抖改到别的状态)。
func (r MerchantApplicationRepository) BindMerchant(ctx context.Context, id, merchantID uint) error {
	return r.db.WithContext(ctx).Model(&MerchantApplicationPO{}).
		Where("id = ? AND status = ?", id, int(domainmerchant.AppStatusApproved)).
		Update("merchant_id", merchantID).Error
}
