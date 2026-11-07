package settlepo

import (
	"context"
	"errors"
	"time"

	domainsettlement "github.com/wsc-zz/service/internal/domain/settlement"
	"gorm.io/gorm"
)

// settlementRepository 是 domain/settlement.Repository 的 GORM 实现。
type settlementRepository struct {
	db *gorm.DB
}

func NewSettlementRepository(db *gorm.DB) *settlementRepository {
	return &settlementRepository{db: db}
}

// 编译期断言:接口签名漂移时在这里炸,而不是等装配那天
var _ domainsettlement.Repository = (*settlementRepository)(nil)

// Save 聚合 + 明细同一事务落库。po.Details 非空时 GORM 级联创建会自己开事务包住
// 两表插入(先插主表拿 ID,再插明细并回填 settlement_id),仿 order.Save 的 items 级联。
// 分账单一经创建就不再走 Save 改字段(状态流转全走条件更新),只实现插入路径。
func (r *settlementRepository) Save(ctx context.Context, s *domainsettlement.Settlement) error {
	po := toPO(s)
	err := r.db.WithContext(ctx).Create(&po).Error
	// pay_no / settle_no 唯一索引冲突翻译成领域错误(依赖 global 开 TranslateError):
	// 应用层幂等建单靠它区分"并发重放建重复了"(重查当成功)和真故障,不把 gorm 漏进应用层
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return domainsettlement.ErrDuplicateSettlement
	}
	if err != nil {
		return err
	}
	// 回填主键与时间戳到聚合:ID 是条件更新四件套(Start/MarkSettled/...)的定位键
	s.ID = po.ID
	s.CreatedAt = po.CreatedAt
	return nil
}

func (r *settlementRepository) FindBySettleNo(ctx context.Context, settleNo string) (*domainsettlement.Settlement, error) {
	var po SettlementPO
	err := r.db.WithContext(ctx).Preload("Details").First(&po, "settle_no = ?", settleNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainsettlement.ErrSettlementNotFound
		}
		return nil, err
	}
	return toDomain(&po), nil
}

// FindByPayNo 按支付单找分账单。查无此单返回 (nil, nil) 而不是错误——
// "没建过"是幂等建单的正常分支,不是异常,调用方据此走建单流程。
func (r *settlementRepository) FindByPayNo(ctx context.Context, payNo string) (*domainsettlement.Settlement, error) {
	var po SettlementPO
	err := r.db.WithContext(ctx).Preload("Details").First(&po, "pay_no = ?", payNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toDomain(&po), nil
}

func (r *settlementRepository) List(ctx context.Context, q domainsettlement.ListQuery) ([]*domainsettlement.Settlement, int64, error) {
	var (
		pos   []SettlementPO
		total int64
	)
	db := r.db.WithContext(ctx).Model(&SettlementPO{})
	if q.Status > 0 {
		db = db.Where("status = ?", int(domainsettlement.SettleStatus(q.Status)))
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (q.Page - 1) * q.PageSize
	if err := db.Preload("Details").Order("id DESC").Offset(offset).Limit(q.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	settlements := make([]*domainsettlement.Settlement, 0, len(pos))
	for i := range pos {
		settlements = append(settlements, toDomain(&pos[i]))
	}
	return settlements, total, nil
}

// Start 条件更新 待结算→结算中(1→2)。WHERE status=? 让状态机成为并发裁决者:
// job 和管理员"立即执行"同时抢,只有一人生效(RowsAffected==1),输家拿 false 当"别人在跑"。
// 生效与否看 RowsAffected,不看内存结构。
func (r *settlementRepository) Start(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Model(&SettlementPO{}).
		Where("id = ? AND status = ?", id, int(domainsettlement.SettleStatusPending)).
		Update("status", int(domainsettlement.SettleStatusExecuting))
	return result.RowsAffected == 1, result.Error
}

// MarkSettled 条件更新 结算中→结算成功(2→3),落 SettledAt。
// 渠道已把钱分出去,这里若 0 行(单子被并发改走)重试也不会重复分钱:
// out_request_no 是渠道幂等锚点,同一笔渠道侧只分一次。
func (r *settlementRepository) MarkSettled(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Model(&SettlementPO{}).
		Where("id = ? AND status = ?", id, int(domainsettlement.SettleStatusExecuting)).
		Updates(map[string]interface{}{
			"status":     int(domainsettlement.SettleStatusSuccess),
			"settled_at": time.Now(),
		})
	return result.RowsAffected == 1, result.Error
}

// FailRequeue 条件更新 结算中→待结算(2→1,未超限)或结算中→结算失败(2→4,超限)。
// 两个 UPDATE 各自看 RowsAffected:分支条件(retry_count + 1 与 maxRetry 比大小)写进 WHERE
// 让每条 UPDATE 自选,不在应用层先读后判——读、判、写三步之间单子可能被并发改走,
// 判断依据落在 SQL 里才是原子的。retry_count = retry_count + 1 写 SQL 表达式,
// 不拿应用层读到的旧值 +1 写回(并发下会丢加)。
func (r *settlementRepository) FailRequeue(ctx context.Context, id uint, reason string, maxRetry int) (bool, error) {
	// 先试超限分支:retry_count + 1 达到 maxRetry → 结算失败(终态)
	result := r.db.WithContext(ctx).Model(&SettlementPO{}).
		Where("id = ? AND status = ? AND retry_count + 1 >= ?", id, int(domainsettlement.SettleStatusExecuting), maxRetry).
		Updates(map[string]interface{}{
			"status":      int(domainsettlement.SettleStatusFailed),
			"retry_count": gorm.Expr("retry_count + 1"),
			"fail_reason": reason,
		})
	if result.Error != nil || result.RowsAffected == 1 {
		return result.RowsAffected == 1, result.Error
	}
	// 未超限:回到待结算,下一轮 job 再来
	result = r.db.WithContext(ctx).Model(&SettlementPO{}).
		Where("id = ? AND status = ? AND retry_count + 1 < ?", id, int(domainsettlement.SettleStatusExecuting), maxRetry).
		Updates(map[string]interface{}{
			"status":      int(domainsettlement.SettleStatusPending),
			"retry_count": gorm.Expr("retry_count + 1"),
			"fail_reason": reason,
		})
	return result.RowsAffected == 1, result.Error
}

// MarkRetry 条件更新 结算失败→待结算(4→1)。管理员手动重试,给终态单子第二次生命;
// 顺手清掉 fail_reason:单子重新上路,上一轮的失败原因不该带进新状态
// (成功后残留只会误导排查;若再失败 FailRequeue 会写上新的)。
func (r *settlementRepository) MarkRetry(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Model(&SettlementPO{}).
		Where("id = ? AND status = ?", id, int(domainsettlement.SettleStatusFailed)).
		Updates(map[string]interface{}{
			"status":      int(domainsettlement.SettleStatusPending),
			"fail_reason": "",
		})
	return result.RowsAffected == 1, result.Error
}

// FindPending job 扫描:待结算的单按 id 升序取 limit 张,老单先结。
func (r *settlementRepository) FindPending(ctx context.Context, limit int) ([]*domainsettlement.Settlement, error) {
	var pos []SettlementPO
	if err := r.db.WithContext(ctx).
		Where("status = ?", int(domainsettlement.SettleStatusPending)).
		Preload("Details").Order("id").Limit(limit).Find(&pos).Error; err != nil {
		return nil, err
	}
	settlements := make([]*domainsettlement.Settlement, 0, len(pos))
	for i := range pos {
		settlements = append(settlements, toDomain(&pos[i]))
	}
	return settlements, nil
}

func toDomain(po *SettlementPO) *domainsettlement.Settlement {
	details := make([]domainsettlement.SettlementDetail, 0, len(po.Details))
	for _, d := range po.Details {
		details = append(details, domainsettlement.SettlementDetail{
			MerchantID: d.MerchantID,
			Amount:     d.Amount,
		})
	}
	return &domainsettlement.Settlement{
		ID:               po.ID,
		SettleNo:         po.SettleNo,
		PayNo:            po.PayNo,
		OrderNo:          po.OrderNo,
		TotalAmount:      po.TotalAmount,
		SettleAmount:     po.SettleAmount,
		CommissionAmount: po.CommissionAmount,
		RateBps:          po.RateBps,
		Status:           domainsettlement.SettleStatus(po.Status),
		RetryCount:       po.RetryCount,
		FailReason:       po.FailReason,
		CreatedAt:        po.CreatedAt,
		SettledAt:        po.SettledAt,
		Details:          details,
	}
}

func toPO(s *domainsettlement.Settlement) SettlementPO {
	details := make([]SettlementDetailPO, 0, len(s.Details))
	for _, d := range s.Details {
		details = append(details, SettlementDetailPO{
			MerchantID: d.MerchantID,
			Amount:     d.Amount,
		})
	}
	return SettlementPO{
		SettleNo:         s.SettleNo,
		PayNo:            s.PayNo,
		OrderNo:          s.OrderNo,
		TotalAmount:      s.TotalAmount,
		SettleAmount:     s.SettleAmount,
		CommissionAmount: s.CommissionAmount,
		RateBps:          s.RateBps,
		Status:           int(s.Status),
		RetryCount:       s.RetryCount,
		FailReason:       s.FailReason,
		CreatedAt:        s.CreatedAt,
		SettledAt:        s.SettledAt,
		Details:          details,
	}
}
