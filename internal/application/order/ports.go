package orderapp

import (
	"context"
)

type InventoryService interface {
	DeductBatch(ctx context.Context, requestID string, item []InventoryItem) ([]DeductedItem, error)
	RestockBatch(ctx context.Context, requestID string, item []InventoryItem) error
}

type InventoryItem struct {
	ProductID uint
	SKUCode   string
	Quantity  int
}

type DeductedItem struct {
	ProductID   uint
	SKUCode     string
	ProductName string
	UnitPrice   int64 // 分
	Quantity    int
}
