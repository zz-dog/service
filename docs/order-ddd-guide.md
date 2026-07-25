# Order 模块 DDD 实现指南

> 本文档带你从零实现一个电商订单模块，沿用项目现有的 user 模块架构。
> 按顺序完成 6 个步骤即可。每步都包含：文件清单、完整代码、设计点、自检问题。

---

## 0. 业务需求与架构总览

### 业务需求
- **电商购物订单**：聚合根 `Order` + 值对象 `OrderItem`（一对多商品明细）、收货地址、总金额
- **完整状态机**：
  ```
  待支付(1) ──支付──> 已支付(2) ──发货──> 已发货(3) ──完成──> 已完成(4)
     │                   │
     └──取消──> 已取消(5) <──取消──┘
  ```
  - 待支付、已支付 可取消；已发货、已完成 不可取消
- **四个用例**：创建订单、查询订单（列表+详情）、模拟支付、取消订单

### 分层架构（与 user 模块完全对称）
```
internal/domain/order/                      ← 第1步：领域层（核心，不依赖任何人）
internal/application/order/                 ← 第2步：应用层（用例编排）
internal/infrastructure/persistence/order/  ← 第3步：基础设施层（GORM 实现）
internal/interfaces/http/handler/order.go   ← 第4步：接口层（HTTP 入口）
cmd/main.go（修改）                          ← 第5步：组合根装配
```

依赖方向：`接口层 -> 应用层 -> 领域层 <- 基础设施层`（领域层定义端口，基础设施层实现端口，这就是"依赖倒置"）。

### 重要约定：金额一律用"分"（int64）
金融领域铁律：**永远不要用 float64 存钱**。`1.99 元` 存成 `199`（分），加减乘除永远精确。
全栈统一：领域实体、PO、DTO、请求体 都用 `int64` 存分，前端拿到后除以 100 显示。

---

## 第 1 步：领域层 `internal/domain/order/`

领域层是核心，纯净、不依赖任何外部库（不带 gorm/json 标签）。包含 5 个文件。

### 1.1 `status.go` — 状态常量

```go
package order

// 订单状态
const (
	StatusPending   = 1 // 待支付
	StatusPaid      = 2 // 已支付
	StatusShipped   = 3 // 已发货
	StatusCompleted = 4 // 已完成
	StatusCancelled = 5 // 已取消
)

// statusName 状态中文名，用于日志与展示
var statusName = map[int]string{
	StatusPending:   "待支付",
	StatusPaid:      "已支付",
	StatusShipped:   "已发货",
	StatusCompleted: "已完成",
	StatusCancelled: "已取消",
}

// StatusName 返回状态的中文名；未知状态返回"未知状态"
func StatusName(s int) string {
	if name, ok := statusName[s]; ok {
		return name
	}
	return "未知状态"
}
```

### 1.2 `errors.go` — 领域错误

```go
package order

import "errors"

// 领域错误：用哨兵错误表达业务语义。
// 仓储实现把底层错误（如 gorm.ErrRecordNotFound）翻译成这些领域错误，
// 上层据此映射 HTTP 响应，无需感知任何持久化细节。
var (
	ErrOrderNotFound           = errors.New("订单不存在")
	ErrInvalidStatusTransition = errors.New("订单当前状态不允许该操作")
	ErrOrderAlreadyCancelled   = errors.New("订单已取消，不可重复操作")
	ErrEmptyOrderItems         = errors.New("订单商品明细不能为空")
	ErrInvalidQuantity         = errors.New("商品数量必须大于0")
	ErrInvalidPrice            = errors.New("商品单价必须大于0")
	ErrConsigneeIncomplete     = errors.New("收货信息不完整")
	ErrEmptyLogisticsNo        = errors.New("物流单号不能为空")
)
```

### 1.3 `item.go` — 订单明细（值对象）

```go
package order

// OrderItem 是订单明细（值对象）。
// 它没有独立身份，作为 Order 聚合根的一部分存在，由 Order 负责一致性。
// 金额一律用 int64 存"分"，避免浮点误差。
type OrderItem struct {
	ProductID   uint
	ProductName string
	Quantity    int
	Price       int64 // 单价，单位：分
}

// Subtotal 计算小计金额（单价 * 数量），单位：分
func (i OrderItem) Subtotal() int64 {
	return i.Price * int64(i.Quantity)
}

// validate 校验明细字段是否合法
func (i OrderItem) validate() error {
	if i.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	if i.Price <= 0 {
		return ErrInvalidPrice
	}
	return nil
}
```

**设计点**：
- `OrderItem` 是**值对象**（无 ID，依附于 `Order`），对比 `Order` 是**实体**（有 `OrderID`，可独立查询）。
- `validate` 小写开头（私有），只给 `Order` 内部调用。
- `Subtotal()` 把"算小计"收进值对象，属于充血模型。

### 1.4 `order.go` — 聚合根（核心）

```go
package order

import (
	"fmt"
	"time"
)

// Order 是订单聚合根。
// 聚合内含 OrderItem 列表，由 Order 统一管理一致性（保存时一并落库）。
// 纯净结构：不带任何 GORM / JSON 标签，不依赖外部库。
type Order struct {
	OrderID     uint
	UserID      uint
	OrderNo     string
	Status      int
	TotalAmount int64 // 总金额，单位：分

	// 收货信息
	ConsigneeName    string
	ConsigneePhone   string
	ConsigneeAddress string

	// 物流
	LogisticsNo string

	// 订单明细（聚合内的值对象集合）
	Items []OrderItem

	CreatedAt   time.Time
	UpdatedAt   time.Time
	PaidAt      *time.Time
	CancelledAt *time.Time
}

// NewOrder 创建一个待支付订单。
// 作为充血模型的构造函数，集中封装不变量：校验明细、计算总额、生成订单号、初始状态。
func NewOrder(userID uint, items []OrderItem, consigneeName, consigneePhone, consigneeAddress string) (*Order, error) {
	if len(items) == 0 {
		return nil, ErrEmptyOrderItems
	}
	var total int64
	for _, it := range items {
		if err := it.validate(); err != nil {
			return nil, err
		}
		total += it.Subtotal()
	}
	if consigneeName == "" || consigneePhone == "" || consigneeAddress == "" {
		return nil, ErrConsigneeIncomplete
	}

	return &Order{
		UserID:           userID,
		OrderNo:          generateOrderNo(),
		Status:           StatusPending,
		TotalAmount:      total,
		ConsigneeName:    consigneeName,
		ConsigneePhone:   consigneePhone,
		ConsigneeAddress: consigneeAddress,
		Items:            items,
	}, nil
}

// generateOrderNo 生成订单号：yyyyMMddHHmmss + 4 位纳秒后缀。
// 简单方案，足以演示；生产环境可换成雪花算法等，通过端口注入。
func generateOrderNo() string {
	now := time.Now()
	return fmt.Sprintf("%s%04d", now.Format("20060102150405"), now.UnixNano()%10000)
}

// Pay 待支付 -> 已支付
func (o *Order) Pay() error {
	if o.Status != StatusPending {
		return ErrInvalidStatusTransition
	}
	o.Status = StatusPaid
	now := time.Now()
	o.PaidAt = &now
	return nil
}

// Ship 已支付 -> 已发货，并记录物流单号
func (o *Order) Ship(logisticsNo string) error {
	if o.Status != StatusPaid {
		return ErrInvalidStatusTransition
	}
	if logisticsNo == "" {
		return ErrEmptyLogisticsNo
	}
	o.Status = StatusShipped
	o.LogisticsNo = logisticsNo
	return nil
}

// Complete 已发货 -> 已完成
func (o *Order) Complete() error {
	if o.Status != StatusShipped {
		return ErrInvalidStatusTransition
	}
	o.Status = StatusCompleted
	return nil
}

// Cancel 取消订单：待支付或已支付可取消（已支付视为退款），
// 已发货/已完成不可取消，已取消不可重复取消。
func (o *Order) Cancel() error {
	if o.Status == StatusCancelled {
		return ErrOrderAlreadyCancelled
	}
	if o.Status != StatusPending && o.Status != StatusPaid {
		return ErrInvalidStatusTransition
	}
	o.Status = StatusCancelled
	now := time.Now()
	o.CancelledAt = &now
	return nil
}

// CanCancel 是否可取消（供展示层判断按钮可用性）
func (o *Order) CanCancel() bool {
	return o.Status == StatusPending || o.Status == StatusPaid
}
```

**设计点**：
1. **聚合根一致性**：`Order` 管理 `OrderItem`，外部只能通过 `Order` 的方法改明细，保存时整个聚合一起存。
2. **状态机封装**：`Pay()` 第一行就校验状态。应用层只管调用 `order.Pay()`，不判断状态，错了领域层返回 `ErrInvalidStatusTransition`。状态规则不被散落到各处。
3. **`NewOrder` 集中校验**：明细非空、数量单价合法、收货完整、总额计算，全在构造函数里。保证构造出的 `Order` 一定合法。
4. **订单号生成**在领域层，用 `time.Now()`（领域层可用标准库，只是不依赖 gorm/jwt/gin）。

### 1.5 `repository.go` — 仓储端口

```go
package order

import "context"

// OrderRepository 是订单聚合根的仓储接口（端口）。
// 领域层定义契约，持久化实现由基础设施层（infrastructure/persistence/order）提供。
type OrderRepository interface {
	// FindByID 按主键查询订单（含明细）；未找到返回 ErrOrderNotFound
	FindByID(ctx context.Context, id uint) (*Order, error)
	// FindByUserID 分页查询某用户的订单（含明细），返回订单列表与总数
	FindByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*Order, int64, error)
	// Save 新增或更新订单：主键为 0 时新增（连同明细一起落库），否则更新
	Save(ctx context.Context, o *Order) error
}
```

**设计点**：`FindByUserID` 返回**两个值**（列表 + 总数），用于前端分页显示"共 X 条"。`Save` 负责整个聚合落库，具体怎么用事务实现是第 3 步的事。

### ✅ 第 1 步自检
- [ ] `go build ./internal/domain/order/` 编译通过
- 自检问题：
  - 为什么 `OrderItem` 没有 ID，而 `Order` 有？（值对象 vs 实体）
  - 为什么 `Pay()` 里判断状态，而不是在 service 里判断？（状态机封装）
  - 为什么金额用 `int64` 存"分"？（避免浮点误差）

---

## 第 2 步：应用层 `internal/application/order/`

应用层只编排用例，不写业务规则。依赖领域接口，不碰 GORM。包含 2 个文件。包名用 `orderapp`（与 user 模块的 `userapp` 对称）。

### 2.1 `dto.go` — 输入输出 DTO

```go
package orderapp

import "time"

// ---- 输入 DTO：用例契约，不带 gin binding 标签，保持应用层与 Web 框架解耦 ----

// OrderItemInput 创建订单时的商品明细输入
type OrderItemInput struct {
	ProductID   uint
	ProductName string
	Quantity    int
	Price       int64 // 单位：分
}

// CreateOrderInput 创建订单用例输入
type CreateOrderInput struct {
	UserID           uint
	Items            []OrderItemInput
	ConsigneeName    string
	ConsigneePhone   string
	ConsigneeAddress string
}

// PayOrderInput 模拟支付用例输入
type PayOrderInput struct {
	OrderID uint
	UserID  uint // 用于校验订单归属
}

// CancelOrderInput 取消订单用例输入
type CancelOrderInput struct {
	OrderID uint
	UserID  uint
}

// GetOrderInput 查询订单详情用例输入
type GetOrderInput struct {
	OrderID uint
	UserID  uint
}

// QueryOrdersInput 分页查询订单用例输入
type QueryOrdersInput struct {
	UserID   uint
	Page     int
	PageSize int
}

// ---- 输出 DTO ----

// OrderItemDTO 订单明细视图
type OrderItemDTO struct {
	ProductID   uint   `json:"productId"`
	ProductName string `json:"productName"`
	Quantity    int    `json:"quantity"`
	Price       int64  `json:"price"`    // 单位：分
	Subtotal    int64  `json:"subtotal"` // 小计，单位：分
}

// OrderDTO 订单视图
type OrderDTO struct {
	OrderID          uint           `json:"orderId"`
	UserID           uint           `json:"userId"`
	OrderNo          string         `json:"orderNo"`
	Status           int            `json:"status"`
	StatusName       string         `json:"statusName"`
	TotalAmount      int64          `json:"totalAmount"` // 单位：分
	ConsigneeName    string         `json:"consigneeName"`
	ConsigneePhone   string         `json:"consigneePhone"`
	ConsigneeAddress string         `json:"consigneeAddress"`
	LogisticsNo      string         `json:"logisticsNo"`
	Items            []OrderItemDTO `json:"items"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	PaidAt           *time.Time     `json:"paidAt,omitempty"`
	CancelledAt      *time.Time     `json:"cancelledAt,omitempty"`
}

// OrderListResult 订单分页结果
type OrderListResult struct {
	List     []OrderDTO `json:"list"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
}
```

### 2.2 `service.go` — 应用服务

```go
package orderapp

import (
	"context"

	domainorder "github.com/wsc-zz/service/internal/domain/order"
)

// Service 是订单应用服务，编排创建/支付/取消/查询等用例。
// 它只依赖领域仓储接口，不接触 GORM 等具体实现。
type Service struct {
	repo domainorder.OrderRepository
}

// NewService 构造应用服务，注入订单仓储。
func NewService(repo domainorder.OrderRepository) *Service {
	return &Service{repo: repo}
}

// Create 创建订单，成功返回订单视图。
func (s *Service) Create(ctx context.Context, in CreateOrderInput) (*OrderDTO, error) {
	items := make([]domainorder.OrderItem, 0, len(in.Items))
	for _, it := range in.Items {
		items = append(items, domainorder.OrderItem{
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			Price:       it.Price,
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

// Pay 模拟支付：查出订单 -> 校验归属 -> 调用聚合根 Pay -> 保存。
func (s *Service) Pay(ctx context.Context, in PayOrderInput) (*OrderDTO, error) {
	o, err := s.repo.FindByID(ctx, in.OrderID)
	if err != nil {
		return nil, err
	}
	// 订单不属于当前用户，统一当作"不存在"返回，避免泄露订单存在性
	if o.UserID != in.UserID {
		return nil, domainorder.ErrOrderNotFound
	}
	if err := o.Pay(); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, o); err != nil {
		return nil, err
	}
	dto := toOrderDTO(o)
	return &dto, nil
}

// Cancel 取消订单：查出订单 -> 校验归属 -> 调用聚合根 Cancel -> 保存。
func (s *Service) Cancel(ctx context.Context, in CancelOrderInput) (*OrderDTO, error) {
	o, err := s.repo.FindByID(ctx, in.OrderID)
	if err != nil {
		return nil, err
	}
	if o.UserID != in.UserID {
		return nil, domainorder.ErrOrderNotFound
	}
	if err := o.Cancel(); err != nil {
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

// ListByUser 分页查询某用户的订单列表。
func (s *Service) ListByUser(ctx context.Context, in QueryOrdersInput) (*OrderListResult, error) {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PageSize < 1 || in.PageSize > 100 {
		in.PageSize = 10
	}
	orders, total, err := s.repo.FindByUserID(ctx, in.UserID, in.Page, in.PageSize)
	if err != nil {
		return nil, err
	}
	list := make([]OrderDTO, 0, len(orders))
	for _, o := range orders {
		list = append(list, toOrderDTO(o))
	}
	return &OrderListResult{
		List:     list,
		Total:    total,
		Page:     in.Page,
		PageSize: in.PageSize,
	}, nil
}

// toOrderDTO 将领域聚合根转为对外输出 DTO。
func toOrderDTO(o *domainorder.Order) OrderDTO {
	items := make([]OrderItemDTO, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, OrderItemDTO{
			ProductID:   it.ProductID,
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
```

**设计点**：
1. 每个用例都是固定套路：**查询 -> 校验归属 -> 调聚合根方法 -> 保存 -> 转 DTO**。
2. **归属校验**：`o.UserID != in.UserID` 时返回 `ErrOrderNotFound`（不当成"无权限"），避免泄露订单是否存在。
3. 应用层**不判断状态**（`o.Pay()` 内部判断），只编排流程。
4. `toOrderDTO` 不暴露领域实体内部结构，且 `OrderItemDTO` 带上计算好的 `Subtotal` 方便前端。

### ✅ 第 2 步自检
- [ ] `go build ./internal/application/order/` 编译通过
- 自检问题：为什么 `PayOrderInput` 里要带 `UserID`？（归属校验，防越权操作别人的订单）

---

## 第 3 步：基础设施层 `internal/infrastructure/persistence/order/`

这一层用 GORM 实现领域层定义的仓储端口。包名 `orderpo`。包含 2 个文件。

### 3.1 `model.go` — PO 与转换函数

```go
package orderpo

import (
	"time"

	"gorm.io/gorm"

	domainorder "github.com/wsc-zz/service/internal/domain/order"
)

// OrderPO 订单持久化对象，对应 orders 表。
type OrderPO struct {
	OrderID     uint   `gorm:"primaryKey;autoIncrement;comment:订单主键"`
	OrderNo     string `gorm:"size:32;uniqueIndex;comment:订单号"`
	UserID      uint   `gorm:"index;comment:下单用户ID"`
	Status      int    `gorm:"tinyint;not null;default:1;comment:订单状态 1待支付 2已支付 3已发货 4已完成 5已取消"`
	TotalAmount int64  `gorm:"comment:总金额(分)"`

	ConsigneeName    string `gorm:"size:32;comment:收货人姓名"`
	ConsigneePhone   string `gorm:"size:20;comment:收货人电话"`
	ConsigneeAddress string `gorm:"size:255;comment:收货地址"`

	LogisticsNo string `gorm:"size:64;default:'';comment:物流单号"`

	CreatedAt   time.Time
	UpdatedAt   time.Time
	PaidAt      *time.Time `gorm:"comment:支付时间"`
	CancelledAt *time.Time `gorm:"comment:取消时间"`

	// 一对多：订单包含的明细行。GORM 通过外键 OrderID 关联。
	Items []OrderItemPO `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
}

// OrderItemPO 订单明细持久化对象，对应 order_items 表。
type OrderItemPO struct {
	ItemID      uint   `gorm:"primaryKey;autoIncrement;comment:明细主键"`
	OrderID     uint   `gorm:"index;comment:所属订单ID"`
	ProductID   uint   `gorm:"comment:商品ID"`
	ProductName string `gorm:"size:128;comment:商品名称"`
	Quantity    int    `gorm:"comment:购买数量"`
	Price       int64  `gorm:"comment:单价(分)"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 自定义表名（GORM 默认会把 OrderPO 推断成 order_p_o_s）
func (OrderPO) TableName() string     { return "orders" }
func (OrderItemPO) TableName() string { return "order_items" }

// toDomain 将订单 PO 转为领域聚合根（含明细）
func toDomain(po *OrderPO) *domainorder.Order {
	items := make([]domainorder.OrderItem, 0, len(po.Items))
	for _, ip := range po.Items {
		items = append(items, domainorder.OrderItem{
			ProductID:   ip.ProductID,
			ProductName: ip.ProductName,
			Quantity:    ip.Quantity,
			Price:       ip.Price,
		})
	}
	return &domainorder.Order{
		OrderID:          po.OrderID,
		UserID:           po.UserID,
		OrderNo:          po.OrderNo,
		Status:           po.Status,
		TotalAmount:      po.TotalAmount,
		ConsigneeName:    po.ConsigneeName,
		ConsigneePhone:   po.ConsigneePhone,
		ConsigneeAddress: po.ConsigneeAddress,
		LogisticsNo:      po.LogisticsNo,
		Items:            items,
		CreatedAt:        po.CreatedAt,
		UpdatedAt:        po.UpdatedAt,
		PaidAt:           po.PaidAt,
		CancelledAt:      po.CancelledAt,
	}
}

// toPO 将领域聚合根转为订单 PO（主表，不含明细）。
// 注意：故意不含 Items，更新时只存主表，避免 GORM 误删明细。
func toPO(o *domainorder.Order) *OrderPO {
	return &OrderPO{
		OrderID:          o.OrderID,
		OrderNo:          o.OrderNo,
		UserID:           o.UserID,
		Status:           o.Status,
		TotalAmount:      o.TotalAmount,
		ConsigneeName:    o.ConsigneeName,
		ConsigneePhone:   o.ConsigneePhone,
		ConsigneeAddress: o.ConsigneeAddress,
		LogisticsNo:      o.LogisticsNo,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
		PaidAt:           o.PaidAt,
		CancelledAt:      o.CancelledAt,
	}
}

// toItemsPO 将领域明细列表转为 PO 列表（新增时用，OrderID 由 GORM 级联创建回填）
func toItemsPO(items []domainorder.OrderItem) []OrderItemPO {
	result := make([]OrderItemPO, 0, len(items))
	for _, it := range items {
		result = append(result, OrderItemPO{
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			Price:       it.Price,
		})
	}
	return result
}
```

**设计点**：
1. **PO 与领域实体分离**：`OrderPO` 带 GORM 标签，`domain.Order` 纯净。中间靠 `toDomain`/`toPO` 转换，和 user 模块一样的模式。
2. **`toPO` 故意不含 Items**：这是关键。GORM 的 `Save` 对 has-many 关联会用当前 slice **替换**关联，若 po.Items 为空会**删光所有明细**！所以更新时 `toPO` 不带 Items，只存主表。
3. **`TableName`**：GORM 默认把 `OrderPO` 推断成 `order_p_o_s`（按大写分词），所以显式指定 `orders` / `order_items`。
4. **一对多关联**：`Items []OrderItemPO` + `foreignKey:OrderID` 声明关联，查询时用 `Preload("Items")` 一次性带出明细。

### 3.2 `repository.go` — GORM 仓储实现

```go
package orderpo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	domainorder "github.com/wsc-zz/service/internal/domain/order"
)

// OrderRepository 是 domain/order.OrderRepository 的 GORM 实现。
type OrderRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造仓储实现，注入 *gorm.DB。
func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// 编译期断言：确保实现满足领域接口
var _ domainorder.OrderRepository = (*OrderRepository)(nil)

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
	db := r.db.WithContext(ctx).Model(&OrderPO{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
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
```

**设计点**：
1. **错误翻译**：`gorm.ErrRecordNotFound` -> `domainorder.ErrOrderNotFound`，上层只见领域错误。
2. **新增 vs 更新**：`OrderID == 0` 走新增（级联创建明细 + 回填主键），否则走更新（`Save` 只存主表，因为 `toPO` 没带 Items）。
3. **分页查询**：先 `Count` 总数，再 `Limit/Offset` 取列表，`Order("order_id DESC")` 让最新订单排前面。
4. **`Preload("Items")`**：避免 N+1 查询，一次性把明细带出来。

> 💡 关于"明细创建后不可变"：订单一旦创建，商品明细就不应该再改（改金额/数量等于换订单）。状态变化（支付、取消）只动主表字段，所以更新时不碰 Items 是合理的。如果想支持改明细，需要更复杂的差异同步逻辑，超出当前范围。

### ✅ 第 3 步自检
- [ ] `go build ./internal/infrastructure/persistence/order/` 编译通过
- 自检问题：为什么更新订单时不能把 `po.Items` 一起 `Save`？（GORM 会用空 slice 替换关联，删光明细）

---

## 第 4 步：接口层 `internal/interfaces/http/`

### 4.1 新建 `handler/order.go`

```go
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	orderapp "github.com/wsc-zz/service/internal/application/order"
	domainorder "github.com/wsc-zz/service/internal/domain/order"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

// OrderHandler 是订单相关的 HTTP 处理器。
type OrderHandler struct {
	orderSvc *orderapp.Service
}

func NewOrderHandler(orderSvc *orderapp.Service) *OrderHandler {
	return &OrderHandler{orderSvc: orderSvc}
}

// ---- 请求结构体（带 gin binding 标签，仅接口层感知 Web 框架）----

type orderItemRequest struct {
	ProductID   uint   `json:"productId" binding:"required"`
	ProductName string `json:"productName" binding:"required"`
	Quantity    int    `json:"quantity" binding:"required,min=1"`
	Price       int64  `json:"price" binding:"required,min=1"` // 单位：分
}

type createOrderRequest struct {
	Items            []orderItemRequest `json:"items" binding:"required,min=1,dive"`
	ConsigneeName    string             `json:"consigneeName" binding:"required"`
	ConsigneePhone   string             `json:"consigneePhone" binding:"required"`
	ConsigneeAddress string             `json:"consigneeAddress" binding:"required"`
}

type listOrdersRequest struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// Create 创建订单
func (h *OrderHandler) Create(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	items := make([]orderapp.OrderItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, orderapp.OrderItemInput{
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			Price:       it.Price,
		})
	}
	result, err := h.orderSvc.Create(c.Request.Context(), orderapp.CreateOrderInput{
		UserID:           userID,
		Items:            items,
		ConsigneeName:    req.ConsigneeName,
		ConsigneePhone:   req.ConsigneePhone,
		ConsigneeAddress: req.ConsigneeAddress,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "createOrder", result)
}

// Pay 模拟支付
func (h *OrderHandler) Pay(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "订单ID格式错误")
		return
	}
	result, err := h.orderSvc.Pay(c.Request.Context(), orderapp.PayOrderInput{
		OrderID: uint(orderID),
		UserID:  userID,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "payOrder", result)
}

// Cancel 取消订单
func (h *OrderHandler) Cancel(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "订单ID格式错误")
		return
	}
	result, err := h.orderSvc.Cancel(c.Request.Context(), orderapp.CancelOrderInput{
		OrderID: uint(orderID),
		UserID:  userID,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "cancelOrder", result)
}

// GetByID 查询订单详情
func (h *OrderHandler) GetByID(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "订单ID格式错误")
		return
	}
	result, err := h.orderSvc.GetByID(c.Request.Context(), orderapp.GetOrderInput{
		OrderID: uint(orderID),
		UserID:  userID,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "getOrder", result)
}

// List 查询订单列表（分页）
func (h *OrderHandler) List(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	var req listOrdersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}
	result, err := h.orderSvc.ListByUser(c.Request.Context(), orderapp.QueryOrdersInput{
		UserID:   userID,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "listOrders", result)
}

// writeError 将领域/应用错误映射为对应的 HTTP 响应。
func (h *OrderHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainorder.ErrOrderNotFound):
		response.NotFound(c, http.StatusNotFound, err.Error())
	case errors.Is(err, domainorder.ErrInvalidStatusTransition),
		errors.Is(err, domainorder.ErrOrderAlreadyCancelled),
		errors.Is(err, domainorder.ErrEmptyOrderItems),
		errors.Is(err, domainorder.ErrInvalidQuantity),
		errors.Is(err, domainorder.ErrInvalidPrice),
		errors.Is(err, domainorder.ErrConsigneeIncomplete):
		response.BadRequest(c, http.StatusBadRequest, err.Error())
	default:
		response.ServerError(c, http.StatusInternalServerError, err.Error())
	}
}

// getCurrentUserID 从 gin 上下文取 JWT 中间件写入的 userID。
// JWT 中间件写入的是 string（见 auth.UserClaims.UserID），这里转成 uint。
// 提示：后续其他 handler 也会用到，可提取到 handler 包的公共文件 base.go。
func getCurrentUserID(c *gin.Context) (uint, bool) {
	v, exists := c.Get("userId")
	if !exists {
		return 0, false
	}
	s, ok := v.(string)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}
```

**设计点**：
1. **UserID 来自 JWT，不来自请求体**：`getCurrentUserID(c)` 从 `c.Get("userId")` 取（JWT 中间件写入），前端无法伪造。这是防越权的关键。
2. **`binding:"required,min=1,dive"`**：`dive` 让 gin 进入 slice 内部校验每个 `orderItemRequest`。
3. **`writeError` 错误映射**：领域错误 -> HTTP 状态码。`NotFound` -> 404，参数/状态类错误 -> 400，未知 -> 500。
4. handler 职责很薄：解析参数 -> 调应用服务 -> 转响应，不含业务逻辑。

### 4.2 修改 `router/router.go`

把 `InitRouter` 改成接收两个服务，并增加订单路由组（套 JWT 中间件）。完整文件如下：

```go
package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/wsc-zz/service/internal/application/order"
	"github.com/wsc-zz/service/internal/application/user"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	"github.com/wsc-zz/service/internal/interfaces/http/middleware"
)

// InitRouter 初始化路由，注入用户与订单应用服务。
func InitRouter(userSvc *userapp.Service, orderSvc *orderapp.Service) *gin.Engine {
	r := gin.Default()

	// 允许本地开发前端跨域访问
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8081", "http://127.0.0.1:8081"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	h := handler.NewHandler(userSvc)
	orderH := handler.NewOrderHandler(orderSvc)

	apiGroup := r.Group("/api")
	{
		// 用户：注册、登录（无需登录）
		apiGroup.POST("/user/register", h.Register)
		apiGroup.POST("/user/login", h.Login)

		// 订单：全部需要登录（JWT 中间件校验 token 并写入 userId）
		orderGroup := apiGroup.Group("/order", middleware.JWTAuth())
		{
			orderGroup.POST("", orderH.Create)            // 创建订单
			orderGroup.GET("", orderH.List)               // 订单列表（分页）
			orderGroup.GET("/:id", orderH.GetByID)        // 订单详情
			orderGroup.POST("/:id/pay", orderH.Pay)       // 模拟支付
			orderGroup.POST("/:id/cancel", orderH.Cancel) // 取消订单
		}
	}
	return r
}
```

> ⚠️ 注意：`InitRouter` 的签名变了（多了 `orderSvc` 参数），所以 `main.go` 调用处也要改（第 5 步）。

### ✅ 第 4 步自检
- [ ] `go build ./internal/interfaces/http/...` 编译通过
- 自检问题：为什么 `Create` 的 `userID` 从 `c.Get("userId")` 取，而不是从请求体取？（防伪造越权）

---

## 第 5 步：组合根 `cmd/main.go`（修改）

`main.go` 是唯一同时感知所有层的地方。需要：① import order 相关包；② AutoMigrate 加两张表；③ 装配 order 仓储和服务；④ 传给 router。完整文件如下：

```go
package main

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
	"github.com/wsc-zz/service/internal/application/order"
	"github.com/wsc-zz/service/internal/application/user"
	"github.com/wsc-zz/service/internal/infrastructure/auth"
	"github.com/wsc-zz/service/internal/infrastructure/persistence/order"
	"github.com/wsc-zz/service/internal/infrastructure/persistence/user"
	"github.com/wsc-zz/service/internal/infrastructure/security"
	"github.com/wsc-zz/service/internal/interfaces/http/router"
)

func main() {
	// 1. 初始化基础设施：配置、日志、数据库
	global.InitViper()
	global.InitZap()
	global.InitMysql()

	// 2. 自动迁移持久化对象，确保表已创建/更新
	if err := global.DB.AutoMigrate(&userpo.UserPO{}, &orderpo.OrderPO{}, &orderpo.OrderItemPO{}); err != nil {
		global.Logger.Error("数据表迁移失败", zap.Error(err))
		panic("数据表迁移失败:" + err.Error())
	}
	global.Logger.Info("数据表迁移成功")

	// 3. 组合根：依赖注入装配（唯一感知所有层的地方）
	userRepo := userpo.NewUserRepository(global.DB)
	hasher := security.NewBcryptHasher()
	tokenIssuer := auth.NewJWTTokenIssuer()
	userSvc := userapp.NewService(userRepo, hasher, tokenIssuer)

	orderRepo := orderpo.NewOrderRepository(global.DB)
	orderSvc := orderapp.NewService(orderRepo)

	// 4. 启动 HTTP 服务
	r := router.InitRouter(userSvc, orderSvc)
	if err := r.Run(":" + fmt.Sprint(global.Conf.Service.Port)); err != nil {
		panic(err)
	}
}
```

> 💡 这里 `user` 和 `order` 两个 application 包都叫 `userapp`/`orderapp`，import 路径不同，包名不同，所以不会冲突。persistence 同理（`userpo`/`orderpo`）。

### ✅ 第 5 步自检
- [ ] `go build ./...` 整个项目编译通过

---

## 第 6 步：编译验证与接口测试

### 6.1 编译
```bash
go build ./...
```
若报错，常见原因：
- import 路径写错（检查 `github.com/wsc-zz/service/...` 是否完整）
- 包名不符（应用层是 `orderapp`，持久层是 `orderpo`，领域层是 `order`）
- `InitRouter` 签名改了但 `main.go` 没同步

### 6.2 启动服务
```bash
go run cmd/main.go
```
看到 "数据表迁移成功" 即 OK，`orders` 和 `order_items` 两张表已自动建好。

### 6.3 用 curl 走一遍完整流程

**① 注册一个用户**
```bash
curl -X POST http://localhost:端口/api/user/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testorder","password":"123456","phone":"13800000001","nickname":"测试"}'
```

**② 登录拿 token**（把返回的 `token` 复制下来）
```bash
curl -X POST http://localhost:端口/api/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testorder","password":"123456"}'
```

> 下面所有请求都带上 `Authorization: Bearer <token>`

**③ 创建订单**（金额用分：199 = 1.99 元）
```bash
curl -X POST http://localhost:端口/api/order \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "items":[
      {"productId":1,"productName":"商品A","quantity":2,"price":199},
      {"productId":2,"productName":"商品B","quantity":1,"price":500}
    ],
    "consigneeName":"张三",
    "consigneePhone":"13800000001",
    "consigneeAddress":"北京市朝阳区xx路1号"
  }'
```
记下返回的 `orderId`（假设是 1）。

**④ 查询订单列表**
```bash
curl "http://localhost:端口/api/order?page=1&pageSize=10" \
  -H "Authorization: Bearer <token>"
```

**⑤ 查询订单详情**
```bash
curl http://localhost:端口/api/order/1 \
  -H "Authorization: Bearer <token>"
```

**⑥ 模拟支付**（状态：待支付 -> 已支付）
```bash
curl -X POST http://localhost:端口/api/order/1/pay \
  -H "Authorization: Bearer <token>"
```

**⑦ 取消订单**（需要新建一个待支付订单来测，因为已支付的也能取消）
```bash
curl -X POST http://localhost:端口/api/order/2/cancel \
  -H "Authorization: Bearer <token>"
```

**⑧ 验证状态机**（对已完成的订单再次支付，应返回 400 "订单当前状态不允许该操作"）
```bash
curl -X POST http://localhost:端口/api/order/1/pay \
  -H "Authorization: Bearer <token>"
```

---

## 附录 A：一次"创建订单"请求的完整数据流

```
① POST /api/order (带 JWT)
   ↓
② middleware.JWTAuth 校验 token，把 userId 写入 gin.Context
   ↓
③ router -> OrderHandler.Create(c)                 [接口层]
   - getCurrentUserID 从 context 取 userID（不信任请求体）
   - ShouldBindJSON 校验参数
   ↓
④ orderSvc.Create(ctx, CreateOrderInput)          [应用层]
   ↓
⑤ domainorder.NewOrder(...)                       [领域层]
   - 校验明细、计算总额、生成订单号、初始状态=待支付
   ↓
⑥ repo.Save(ctx, order)                           [应用层调端口]
   ↓ (实际跑的是)
⑦ OrderRepository.Save                            [基础设施层]
   - OrderID==0 -> Create(po) 级联创建订单+明细
   - 回填主键到领域实体
   ↓
⑧ toOrderDTO(order) -> response.SuccessMsg        [应用层转 DTO + 接口层返回 JSON]
```

## 附录 B：状态机速查表

| 当前状态 | 可执行操作 |
|---------|-----------|
| 待支付(1) | Pay→已支付, Cancel→已取消 |
| 已支付(2) | Ship→已发货, Cancel→已取消 |
| 已发货(3) | Complete→已完成 |
| 已完成(4) | （终态） |
| 已取消(5) | （终态） |

非法流转（如对已取消订单 Pay）返回 `ErrInvalidStatusTransition` -> HTTP 400。

## 附录 C：文件清单（共 8 个新文件 + 2 个修改）

| # | 文件 | 类型 | 步骤 |
|---|------|------|------|
| 1 | `internal/domain/order/status.go` | 新建 | 1 |
| 2 | `internal/domain/order/errors.go` | 新建 | 1 |
| 3 | `internal/domain/order/item.go` | 新建 | 1 |
| 4 | `internal/domain/order/order.go` | 新建 | 1 |
| 5 | `internal/domain/order/repository.go` | 新建 | 1 |
| 6 | `internal/application/order/dto.go` | 新建 | 2 |
| 7 | `internal/application/order/service.go` | 新建 | 2 |
| 8 | `internal/infrastructure/persistence/order/model.go` | 新建 | 3 |
| 9 | `internal/infrastructure/persistence/order/repository.go` | 新建 | 3 |
| 10 | `internal/interfaces/http/handler/order.go` | 新建 | 4 |
| 11 | `internal/interfaces/http/router/router.go` | 修改 | 4 |
| 12 | `cmd/main.go` | 修改 | 5 |

---

祝编码顺利！卡住时回来对照文档检查，或直接问。
