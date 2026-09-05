package orderapp

import (
	"context"

	domainorder "github.com/wsc-zz/service/internal/domain/order"
	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

type Service struct {
	repo        domainorder.OrderRepository
	productRepo domainproduct.ProductRepository
}

// NewService 构造应用服务，注入订单仓储。
func NewService(repo domainorder.OrderRepository, productRepo domainproduct.ProductRepository) *Service {
	return &Service{repo: repo, productRepo: productRepo}
}

// Create 创建订单，成功返回订单视图。
func (s *Service) Create(ctx context.Context, in CreateOrderInput) (*OrderDTO, error) {
	items := make([]domainorder.OrderItem, 0, len(in.Items))
	for _, it := range in.Items {
		product, err := s.productRepo.FindByProductAndSKUCode(ctx, it.ProductID, it.SKUCode)
		if err != nil {
			return nil, err
		}
		var sku *domainproduct.SKU
		for i := range product.SKUs {
			if product.SKUs[i].SKUCode == it.SKUCode {
				sku = &product.SKUs[i]
				break
			}
		}
		if sku == nil {
			return nil, domainproduct.ErrSKUNotFound
		}
		if err := s.productRepo.DeductStock(ctx, it.ProductID, it.SKUCode, it.Quantity); err != nil {
			return nil, err
		}
		items = append(items, domainorder.OrderItem{
			ProductID:   it.ProductID,
			SKUCode:     it.SKUCode,
			ProductName: product.Name,
			Quantity:    it.Quantity,
			Price:       sku.Price,
		})
	}
	o, err := domainorder.NewOrder(in.UserID, items, in.ConsigneeName, in.ConsigneePhone, in.ConsigneeAddress)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, o); err != nil {
		return nil, err
	}
	dto := toOrderDTO(o)
	return &dto, nil
}

// GetByID 查询订单详情：校验归属后返回视图。
func (s *Service) GetByID(ctx context.Context, in GetOrderInput) (*OrderDTO, error) {
	o, err := s.repo.FindByID(ctx, in.OrderID)
	if err != nil {
		return nil, err
	}
	if o.UserID != in.UserID {
		return nil, domainorder.ErrOrderNotFound
	}
	dto := toOrderDTO(o)
	return &dto, nil
}

func toOrderDTO(o *domainorder.Order) OrderDTO {
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
		StatusName:       domainorder.StatusName(o.Status),
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
