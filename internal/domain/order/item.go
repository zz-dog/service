package order

type OrderItem struct {
	ProductID   uint   // 商品ID
	ProductName string // 商品名称
	Quantity    int    // 商品数量
	Price       int64  // 单价，单位：分
}

func (o *OrderItem) Subtotal() int64 {
	return o.Price * int64(o.Quantity)
}

func (o *OrderItem) validate() error {

	if o.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	if o.Price < 0 {
		return ErrInvalidPrice
	}
	return nil
}
