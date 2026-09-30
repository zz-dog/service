package inventory

import (
	"context"

	"github.com/wsc-zz/service/internal/infrastructure/persistence/inventory"
	"github.com/wsc-zz/service/internal/interfaces/rpc/inventorypb"
	"gorm.io/gorm"

	domaininventory "github.com/wsc-zz/service/internal/domain/inventory"

	productpo "github.com/wsc-zz/service/internal/infrastructure/persistence/product"
)

type Server struct {
	inventorypb.UnimplementedInventoryServiceServer
	db     *gorm.DB
	ledger *inventory.LedgerRepository
}

func NewServer(db *gorm.DB) *Server {
	return &Server{
		db:     db,
		ledger: inventory.NewLedgerRepository(db),
	}
}

// ledgerRepo 用事务内 db 构造 ledger 仓储,保证流水 INSERT 与扣减 UPDATE 同事务
func (s *Server) ledgerRepo(tx *gorm.DB) *inventory.LedgerRepository {
	return inventory.NewLedgerRepository(tx)
}

func (s *Server) DeductBatch(ctx context.Context, req *inventorypb.DeductBatchRequest) (*inventorypb.DeductBatchResponse, error) {

	if req.GetRequestId() == "" {
		return nil, domaininventory.ErrInvalidRequestID
	}

	// 查询库存流水
	rows, err := s.ledger.FindByRequest(ctx, req.GetRequestId(), domaininventory.ActionDeduct)
	// 存在流水则返回
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		return rowsToDeductResponse(rows), nil
	}
	// 扣减库存
	deductRows, err := toDeductStock(ctx, req, s)
	if err != nil {
		return nil, err
	}
	return rowsToDeductResponse(deductRows), nil
}

func (s *Server) RestockBatch(ctx context.Context, req *inventorypb.RestockBatchRequest) (*inventorypb.RestockBatchResponse, error) {

	if req.GetRequestId() == "" {
		return nil, domaininventory.ErrInvalidRequestID
	}
	// 恢复库存流水
	rows, err := s.ledger.FindByRequest(ctx, req.GetRequestId(), domaininventory.ActionRevert)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		return &inventorypb.RestockBatchResponse{}, nil
	}
	//  恢复库存
	if _, err := toRestockStock(ctx, req, s); err != nil {
		return nil, err
	}

	return &inventorypb.RestockBatchResponse{}, nil
}

// rowsToDeductResponse 把流水行转成扣减响应:快照字段一一对应,
// rows 已按 item_index 排序,顺序即下单时的 items 顺序。
func rowsToDeductResponse(rows []inventory.LedgerPO) *inventorypb.DeductBatchResponse {
	items := make([]*inventorypb.DeductedItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &inventorypb.DeductedItem{
			ProductId:   uint64(row.ProductID),
			SkuCode:     row.SKUCode,
			ProductName: row.ProductName,
			UnitPrice:   row.UnitPrice,
			Quantity:    int32(row.Quantity),
		})
	}
	return &inventorypb.DeductBatchResponse{Items: items}
}

func toDeductStock(ctx context.Context, req *inventorypb.DeductBatchRequest, s *Server) ([]inventory.LedgerPO, error) {
	deductRows := make([]inventory.LedgerPO, 0, len(req.GetItems()))

	err := s.db.Transaction(func(tx *gorm.DB) error {
		prodectRepo := productpo.NewProductRepository(tx)

		for i, item := range req.GetItems() {
			product, err := prodectRepo.FindByProductAndSKUCode(ctx, uint(item.GetProductId()), item.GetSkuCode())
			if err != nil {
				return err
			}

			// 扣减库存
			if err := prodectRepo.DeductStock(ctx, uint(item.GetProductId()), item.GetSkuCode(), int(item.GetQuantity())); err != nil {
				return domaininventory.ErrInsufficientInventory
			}
			idx, ok := product.FindSKU(item.GetSkuCode())
			if ok == false {
				return domaininventory.ErrSKUNotFound
			}
			row := inventory.LedgerPO{
				RequestID:   req.GetRequestId(),
				Action:      domaininventory.ActionDeduct,
				ItemIndex:   i,
				SKUCode:     item.GetSkuCode(),
				ProductID:   uint(item.GetProductId()),
				ProductName: product.Name,
				UnitPrice:   product.SKUs[idx].Price,
				Quantity:    int(item.GetQuantity()),
			}
			if err := s.ledgerRepo(tx).Append(ctx, &row); err != nil {
				return err
			}

			deductRows = append(deductRows, row)
		}
		return nil
	})
	return deductRows, err
}

func toRestockStock(ctx context.Context, req *inventorypb.RestockBatchRequest, s *Server) ([]inventory.LedgerPO, error) {
	restockRows := make([]inventory.LedgerPO, 0, len(req.GetItems()))
	err := s.db.Transaction(func(tx *gorm.DB) error {
		prodectRepo := productpo.NewProductRepository(tx)
		for i, item := range req.GetItems() {
			product, err := prodectRepo.FindByProductAndSKUCode(ctx, uint(item.GetProductId()), item.GetSkuCode())
			if err != nil {
				return err
			}

			// 恢复库存
			if err := prodectRepo.RestockStock(ctx, uint(item.GetProductId()), item.GetSkuCode(), int(item.GetQuantity())); err != nil {
				return domaininventory.ErrInsufficientInventory
			}
			idx, ok := product.FindSKU(item.GetSkuCode())
			if ok == false {
				return domaininventory.ErrSKUNotFound
			}
			row := &inventory.LedgerPO{
				RequestID:   req.GetRequestId(),
				Action:      domaininventory.ActionRevert,
				ItemIndex:   i,
				SKUCode:     item.GetSkuCode(),
				ProductID:   uint(item.GetProductId()),
				ProductName: product.Name,
				Quantity:    int(item.GetQuantity()),
				UnitPrice:   product.SKUs[idx].Price,
			}
			if err := s.ledgerRepo(tx).Append(ctx, row); err != nil {
				return err
			}

			restockRows = append(restockRows, *row)
		}
		return nil
	})
	return restockRows, err
}
