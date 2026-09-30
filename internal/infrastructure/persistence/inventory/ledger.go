package inventory

import (
	"context"
	"errors"

	domianinventory "github.com/wsc-zz/service/internal/domain/inventory"
	"gorm.io/gorm"
)

func (LedgerPO) TableName() string { return "inventory_transactions" }

type LedgerRepository struct{ db *gorm.DB }

func NewLedgerRepository(db *gorm.DB) *LedgerRepository { return &LedgerRepository{db: db} }

// FindByRequest 查某次请求的全部流水(重放检查用)
func (r *LedgerRepository) FindByRequest(ctx context.Context, requestID string, action domianinventory.Action) ([]LedgerPO, error) {
	var rows []LedgerPO
	err := r.db.WithContext(ctx).
		Where("request_id = ? AND action = ?", requestID, action).
		Order("item_index").Find(&rows).Error
	return rows, err
}

// Append 追加一行流水;唯一索引冲突说明同一请求重复提交,返回 ErrDuplicateRequest
func (r *LedgerRepository) Append(ctx context.Context, row *LedgerPO) error {
	err := r.db.WithContext(ctx).Create(row).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) { // 唯一索引冲突
		return domianinventory.ErrDuplicateRequest
	}
	return err
}
