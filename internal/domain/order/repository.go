package order

import "context"

type OrderRepository interface {
	FindByID(ctx context.Context, id uint) (*Order, error)                                      //根据ID查询
	FindByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*Order, int64, error) //根据用户ID分页查询，返回列表与总数
	Save(ctx context.Context, order *Order) error                                               //新增/更新 订单
}
