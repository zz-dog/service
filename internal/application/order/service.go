package orderapp

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wsc-zz/service/global"
	domainOrder "github.com/wsc-zz/service/internal/domain/order"
	"go.uber.org/zap"
)

type Service struct {
	repo domainOrder.OrderRepository

	inventory InventoryService
}

// NewService 构造应用服务，注入订单仓储。
func NewService(repo domainOrder.OrderRepository, inventory InventoryService) *Service {
	return &Service{repo: repo, inventory: inventory}
}

// Create 创建订单，成功返回订单视图。
func (s *Service) Create(ctx context.Context, in CreateOrderInput) (OrderDTO, error) {

	// 事务处理：扣库存 + 创建订单
	requestID := uuid.NewString()
	deducted, err := s.inventory.DeductBatch(ctx, requestID, toInventoryItem(in.Items))
	if err != nil {
		return OrderDTO{}, err
	}
	orderItems := deductedTOdomainOrderItem(deducted)
	o, err := domainOrder.NewOrder(in.UserID, orderItems, in.ConsigneeName, in.ConsigneePhone, in.ConsigneeAddress)
	if err == nil {
		o.RequestID = requestID
		err = s.repo.Save(ctx, o)
	}
	if err != nil {
		// 创建订单失败，恢复库存
		if errr := s.inventory.RestockBatch(ctx, requestID, toInventoryItem(in.Items)); errr != nil {
			global.Logger.Error("下单补偿回补失败,待对账处理",
				zap.String("request_id", requestID), zap.Error(errr))
		}
		return OrderDTO{}, err
	}

	return toOrderDTO(o), nil
}
func (s *Service) CancelOrder(ctx context.Context, in CancelOrderInput) error {
	o, err := s.repo.FindByID(ctx, in.OrderID)
	if err != nil {
		return err
	}
	if o.UserID != in.UserID {
		return domainOrder.ErrOrderNotFound
	}
	changed, err := s.repo.CancelPending(ctx, o.OrderID, time.Now())
	if err != nil || !changed {
		return err
	}

	if err := s.inventory.RestockBatch(ctx, o.RequestID, orderItemToInventoryItem(o.Items)); err != nil {
		global.Logger.Error("取消订单补偿回补失败,待对账处理",
			zap.String("request_id", o.RequestID), zap.Error(err))
		return err
	}
	return nil
}

// AutoCancelExpired 自动取消超过 timeout 仍未支付的订单。
func (s *Service) AutoCancelExpired(ctx context.Context, timeout time.Duration) error {
	orders, err := s.repo.FindPendingBefore(ctx, time.Now().Add(-timeout))
	if err != nil {
		return err
	}
	for _, order := range orders {
		changed, err := s.repo.CancelPending(ctx, order.OrderID, time.Now())
		if err != nil || !changed {
			global.Logger.Error("自动取消订单失败", zap.Uint("order_id", order.OrderID), zap.Error(err))
			continue
		}
		if err := s.inventory.RestockBatch(ctx, order.RequestID, orderItemToInventoryItem(order.Items)); err != nil {
			global.Logger.Error("取消订单补偿回补失败,待对账处理",
				zap.String("request_id", order.RequestID), zap.Error(err))
			return err
		}

	}
	return nil
}

func (s *Service) List(ctx context.Context, in QueryOrdersInput) (OrderListResult, error) {
	orders, total, err := s.repo.List(ctx, domainOrder.ListQuery{
		UserID:   in.UserID,
		Page:     in.Page,
		PageSize: in.PageSize,
		Status:   in.Status,
	})
	if err != nil {
		return OrderListResult{}, err
	}
	dtos := make([]OrderDTO, 0, len(orders))
	for _, o := range orders {
		dtos = append(dtos, toOrderDTO(o))
	}
	return OrderListResult{
		List:  dtos,
		Total: total,
	}, nil
}

func toOrderDTO(o *domainOrder.Order) OrderDTO {
	items := make([]OrderItemDTO, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, OrderItemDTO{
			ProductID:   it.ProductID,
			SKUCode:     it.SKUCode,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			Price:       it.Price,
			Subtotal:    it.Subtotal(),
		})
	}
	return OrderDTO{
		OrderID:          o.OrderID,
		UserID:           o.UserID,
		OrderNo:          o.OrderNo,
		Status:           o.Status,
		StatusName:       domainOrder.StatusName(o.Status),
		TotalAmount:      o.TotalAmount,
		ConsigneeName:    o.ConsigneeName,
		ConsigneePhone:   o.ConsigneePhone,
		ConsigneeAddress: o.ConsigneeAddress,
		LogisticsNo:      o.LogisticsNo,
		Items:            items,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
		PaidAt:           o.PaidAt,
		CancelledAt:      o.CancelledAt,
	}
}

func toInventoryItem(items []OrderItemInput) []InventoryItem {
	var result = make([]InventoryItem, 0, len(items))
	for _, item := range items {
		result = append(result, InventoryItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			SKUCode:   item.SKUCode,
		})
	}
	return result
}

func deductedTOdomainOrderItem(items []DeductedItem) []domainOrder.OrderItem {
	var result = make([]domainOrder.OrderItem, 0, len(items))
	for _, item := range items {
		result = append(result, domainOrder.OrderItem{
			ProductID:   item.ProductID,
			SKUCode:     item.SKUCode,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			Price:       item.UnitPrice,
		})
	}
	return result
}

func orderItemToInventoryItem(items []domainOrder.OrderItem) []InventoryItem {
	var result = make([]InventoryItem, 0, len(items))
	for _, item := range items {
		result = append(result, InventoryItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			SKUCode:   item.SKUCode,
		})
	}
	return result
}
