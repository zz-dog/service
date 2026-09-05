package orderpo

import (
	"context"
	"errors"
	"time"

	domainorder "github.com/wsc-zz/service/internal/domain/order"
	"gorm.io/gorm"
)

// OrderRepository 是 domain/order.OrderRepository 的 GORM 实现。
type OrderRepository struct {
	db *gorm.DB
}

// NewOrderRepository 构造仓储实现，注入 *gorm.DB。
func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) FindByID(ctx context.Context, id uint) (*domainorder.Order, error) {
	var po OrderPO
	err := r.db.WithContext(ctx).Preload("Items").First(&po, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainorder.ErrOrderNotFound
		}
		return nil, err
	}
	return toDomain(&po), nil
}
func (r *OrderRepository) FindByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*domainorder.Order, int64, error) {
	var (
		pos   []OrderPO
		total int64
	)
	// 获取总记录数
	db := r.db.WithContext(ctx).Model(&OrderPO{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	// 分页查询
	offset := (page - 1) * pageSize
	if err := db.Preload("Items").Order("order_id DESC").Limit(pageSize).Offset(offset).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	orders := make([]*domainorder.Order, 0, len(pos))
	for i := range pos {
		orders = append(orders, toDomain(&pos[i]))
	}
	return orders, total, nil
}

func (r *OrderRepository) FindPendingBefore(ctx context.Context, before time.Time) ([]*domainorder.Order, error) {
	var pos []OrderPO
	if err := r.db.WithContext(ctx).
		Where("status = ? AND created_at <= ?", domainorder.StatusPending, before).
		Preload("Items").Find(&pos).Error; err != nil {
		return nil, err
	}
	orders := make([]*domainorder.Order, 0, len(pos))
	for i := range pos {
		orders = append(orders, toDomain(&pos[i]))
	}
	return orders, nil
}

func (r *OrderRepository) CancelPending(ctx context.Context, id uint, before time.Time) (bool, error) {
	query := r.db.WithContext(ctx).Model(&OrderPO{}).
		Where("order_id = ? AND status = ?", id, domainorder.StatusPending)
	if !before.IsZero() {
		query = query.Where("created_at <= ?", before)
	}
	result := query.Updates(map[string]interface{}{
		"status":       domainorder.StatusCancelled,
		"cancelled_at": time.Now(),
	})
	return result.RowsAffected == 1, result.Error
}

func (r *OrderRepository) Save(ctx context.Context, o *domainorder.Order) error {
	po := toPO(o)
	if o.OrderID == 0 {
		// 新增：把明细填进 po.Items，GORM 会级联创建（先插订单拿到 ID，再插明细并回填外键）
		po.Items = toItemsPO(o.Items)
		if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
			return err
		}
		// 回填自增主键与时间戳到领域实体
		o.OrderID = po.OrderID
		o.CreatedAt = po.CreatedAt
		o.UpdatedAt = po.UpdatedAt
		return nil
	}
	// 更新：只存主表（明细创建后不可变，不重新写明细）
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *OrderRepository) List(ctx context.Context, q domainorder.ListQuery) ([]*domainorder.Order, int, error) {
	var (
		pos   []OrderPO
		total int64
	)
	db := r.db.WithContext(ctx).Model(&OrderPO{})
	if q.UserID > 0 {
		db = db.Where("user_id = ?", q.UserID)
	}
	if q.Status > 0 {
		db = db.Where("status = ?", q.Status)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (q.Page - 1) * q.PageSize
	if err := db.Preload("Items").Order("order_id DESC").
		Offset(offset).Limit(q.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	orders := make([]*domainorder.Order, 0, len(pos))
	for i := range pos {
		orders = append(orders, toDomain(&pos[i]))
	}
	return orders, int(total), nil
}
