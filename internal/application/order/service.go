package orderapp

import (
	"context"
	"time"

	domainOrder "github.com/wsc-zz/service/internal/domain/order"
	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

type Service struct {
	repo        domainOrder.OrderRepository
	productRepo domainproduct.ProductRepository
	txRunner    TransactionRunner
}

type TransactionRunner interface {
	Run(ctx context.Context, fn func(domainOrder.OrderRepository, domainproduct.ProductRepository) error) error
}

// NewService 构造应用服务，注入订单仓储。
func NewService(repo domainOrder.OrderRepository, productRepo domainproduct.ProductRepository, txRunner TransactionRunner) *Service {
	return &Service{repo: repo, productRepo: productRepo, txRunner: txRunner}
}

// Create 创建订单，成功返回订单视图。
func (s *Service) Create(ctx context.Context, in CreateOrderInput) (*OrderDTO, error) {
	var result *OrderDTO
	err := s.txRunner.Run(ctx, func(orderRepo domainOrder.OrderRepository, productRepo domainproduct.ProductRepository) error {
		items := make([]domainOrder.OrderItem, 0, len(in.Items))
		for _, it := range in.Items {
			product, err := productRepo.FindByProductAndSKUCode(ctx, it.ProductID, it.SKUCode)
			if err != nil {
				return err
			}
			var sku *domainproduct.SKU
			for i := range product.SKUs {
				if product.SKUs[i].SKUCode == it.SKUCode {
					sku = &product.SKUs[i]
					break
				}
			}
			if sku == nil {
				return domainproduct.ErrSKUNotFound
			}
			if err := productRepo.DeductStock(ctx, it.ProductID, it.SKUCode, it.Quantity); err != nil {
				return err
			}
			items = append(items, domainOrder.OrderItem{
				ProductID:   it.ProductID,
				SKUCode:     it.SKUCode,
				ProductName: product.Name,
				Quantity:    it.Quantity,
				Price:       sku.Price,
			})
		}
		o, err := domainOrder.NewOrder(in.UserID, items, in.ConsigneeName, in.ConsigneePhone, in.ConsigneeAddress)
		if err != nil {
			return err
		}
		if err := orderRepo.Save(ctx, o); err != nil {
			return err
		}
		dto := toOrderDTO(o)
		result = &dto
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
func (s *Service) CancelOrder(ctx context.Context, in CancelOrderInput) error {
	return s.txRunner.Run(ctx, func(orderRepo domainOrder.OrderRepository, productRepo domainproduct.ProductRepository) error {
		o, err := orderRepo.FindByID(ctx, in.OrderID)
		if err != nil {
			return err
		}
		if o.UserID != in.UserID {
			return domainOrder.ErrOrderNotFound
		}
		if !o.CanCancel() {
			return domainOrder.ErrInvalidStatusTransition
		}
		changed, err := orderRepo.CancelPending(ctx, o.OrderID, time.Time{})
		if err != nil {
			return err
		}
		if !changed {
			return domainOrder.ErrInvalidStatusTransition
		}

		for _, it := range o.Items {
			// 恢复库存
			if err := productRepo.RestockStock(ctx, it.ProductID, it.SKUCode, it.Quantity); err != nil {
				return err
			}

		}
		return nil
	})

}

// AutoCancelExpired 自动取消超过 timeout 仍未支付的订单。
func (s *Service) AutoCancelExpired(ctx context.Context, timeout time.Duration) error {
	orders, err := s.repo.FindPendingBefore(ctx, time.Now().Add(-timeout))
	if err != nil {
		return err
	}
	for _, order := range orders {
		err := s.txRunner.Run(ctx, func(orderRepo domainOrder.OrderRepository, productRepo domainproduct.ProductRepository) error {
			current, err := orderRepo.FindByID(ctx, order.OrderID)
			if err != nil {
				return err
			}
			before := time.Now().Add(-timeout)
			changed, err := orderRepo.CancelPending(ctx, current.OrderID, before)
			if err != nil || !changed {
				return err
			}
			for _, item := range current.Items {
				if err := productRepo.RestockStock(ctx, item.ProductID, item.SKUCode, item.Quantity); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
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
