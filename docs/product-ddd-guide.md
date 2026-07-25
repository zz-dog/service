# Product 商品模块 DDD 实现指南

> 本文档带你实现一个**多 SKU 电商商品模块**（含分类管理、库存扣减、管理员鉴权），
> 沿用项目现有的 user / order 模块架构。按顺序完成 8 个步骤即可。
> 每步包含：文件清单、完整代码、设计点、自检问题。

---

## 0. 业务需求与架构总览

### 业务需求
- **多 SKU 商品**：`Product` 聚合根 + `SKU` 值对象集合（结构类比 `Order` + `OrderItem`）
- **分类管理**：`Category` 聚合根（独立），商品引用分类 ID
- **用例**：商品增删改、商品查询（列表+详情+按分类/名称筛选）、库存扣减、分类增删改查
- **权限**：浏览公开、管理要管理员 -> 扩展 JWT 中间件做角色判断

### 分层架构（与 user / order 模块对称）
```
domain/product/                          ← 第2步：领域层
application/product/                     ← 第3、5步：应用层
infrastructure/persistence/product/      ← 第4步：基础设施层
interfaces/http/handler/product.go       ← 第6步：接口层
interfaces/http/handler/category.go      ← 第6步
```

### 跨模块改动（第1步先做，为鉴权打基础）
- user 加 `Role` 字段
- JWT claims 携带角色
- 新增 `AdminAuth` 管理员中间件

### 重要约定
- **金额一律用"分"（int64）**，和 order 模块一致，避免浮点误差。
- **聚合根一致性**：`Product` 管理 `SKU` 列表，保存时一起落库（和 `Order` 管理 `OrderItem` 一样）。
- **库存扣减防超卖**：用 `WHERE stock >= qty` 条件更新，影响行数为 0 即库存不足。

---

## 第 1 步：跨模块准备（user 角色 + JWT + admin 中间件）

改动 5 个已有文件 + 新建 1 个文件。目的是给系统加"角色"概念。

### 1.1 修改 `domain/user/user.go` - 加角色

加角色常量、`Role` 字段、`IsAdmin()` 方法（标注 `// 新增`）：

```go
package user

import "time"

// 登录渠道
const (
	ChannelPassword = 1
	ChannelWechat   = 2
	ChannelAlipay   = 3
)

// 用户角色
const (
	RoleUser  = 1 // 普通用户
	RoleAdmin = 2 // 管理员
)

type User struct {
	UserID    uint
	CreatedAt time.Time
	UpdatedAt time.Time

	Username string
	Password string

	LoginChannel int
	UnionID      string
	OpenID       string
	AlipayUID    string

	Nickname string
	Phone    string
	Email    string
	Avatar   string
	Gender   int8
	Birthday *time.Time

	Status int8

	LastLoginIP string
	LastLoginAt *time.Time

	Role int8 // 新增：用户角色 1普通 2管理员
}

func NewUser(username, hashedPassword, phone, nickname string) *User {
	return &User{
		Username:     username,
		Password:     hashedPassword,
		Phone:        phone,
		Nickname:     nickname,
		LoginChannel: ChannelPassword,
		Status:       1,
		Role:         RoleUser, // 新增：默认普通用户
	}
}

func (u *User) VerifyPassword(plain string, hasher PasswordHasher) bool {
	return hasher.Compare(u.Password, plain) == nil
}

func (u *User) IsDisabled() bool {
	return u.Status == 0
}

// IsAdmin 是否是管理员。新增
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) RecordLogin(ip string, at time.Time) {
	u.LastLoginIP = ip
	u.LastLoginAt = &at
}
```

### 1.2 修改 `persistence/user/model.go` - PO 加角色字段

`UserPO` 加 `Role` 字段，`toDomain` / `toPO` 各加一行：

```go
// UserPO 结构体末尾加：
	Role int8 `gorm:"tinyint;not null;default:1;comment:用户角色 1普通 2管理员"` // 新增

// toDomain 末尾加：
		Role:         po.Role, // 新增

// toPO 末尾加：
		Role:        u.Role, // 新增
```

> `default:1` 保证老数据迁移后默认是普通用户。改完启动服务时 `AutoMigrate` 会自动加 `role` 列。

### 1.3 修改 `auth/token.go` - JWT claims 加角色

`UserClaims` 加 `Role`，`Issue` 签名加 `role` 参数：

```go
// UserClaims JWT 载荷
type UserClaims struct {
	Username string `json:"username"`
	UserID   string `json:"user_id"`
	Role     int8   `json:"role"` // 新增
	jwt.RegisteredClaims
}

// Issue 为指定用户签发 token。新增 role 参数
func (t *JWTTokenIssuer) Issue(userID uint, username string, role int8) (string, error) {
	secret := global.Conf.Jwt.Secret
	expire := time.Hour * time.Duration(global.Conf.Jwt.ExpireHour)

	claims := UserClaims{
		UserID:   strconv.Itoa(int(userID)),
		Username: username,
		Role:     role, // 新增
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
```

### 1.4 修改 `application/user/service.go` - 接口与调用同步

`Issue` 签名变了，这里必须同步改两处：

```go
// TokenIssuer 接口加 role 参数：
type TokenIssuer interface {
	Issue(userID uint, username string, role int8) (string, error) // 改：加 role
}

// Login 方法里调用处传 u.Role：
	token, err := s.tokenIssuer.Issue(u.UserID, u.Username, u.Role) // 改：加 u.Role
```

> ⚠️ 这是连锁改动：1.3 改了 `Issue` 签名，1.4 必须同步，否则编译不过。

### 1.5 修改 `middleware/jwt.go` - 写入 role

```go
		// 将用户信息存入上下文
		c.Set("userId", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role) // 新增
		c.Next()
```

### 1.6 新建 `middleware/admin.go` - 管理员鉴权

```go
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/wsc-zz/service/internal/domain/user"
	"github.com/wsc-zz/service/pkg/response"
)

// AdminAuth 校验当前登录用户是否为管理员。
// 必须在 JWTAuth 之后使用（依赖 JWTAuth 写入的 role）。
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, exists := c.Get("role")
		if !exists {
			response.Unauthorized(c, 401, "用户未登录")
			c.Abort()
			return
		}
		role, ok := v.(int8)
		if !ok {
			response.Unauthorized(c, 401, "用户信息异常")
			c.Abort()
			return
		}
		if role != user.RoleAdmin {
			response.Forbidden(c, 403, "需要管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}
```

**设计点**：
- `AdminAuth` 必须套在 `JWTAuth` 之后，依赖它写入的 `c.Get("role")`。
- `role` 类型断言是 `int8`（和 `UserClaims.Role` 一致），全链路保持 `int8`。
- 非管理员返回 403（登录了但没权限），比 401（没登录）语义准确。

### ✅ 第 1 步自检
- [ ] `go build ./...` 编译通过（重点检查 `Issue` 签名连锁是否都改了）
- 怎么造管理员：注册用户后，手动去数据库把该用户 `role` 改成 `2`，重新登录拿新 token。

---

## 第 2 步：商品领域层 `internal/domain/product/`

### 2.1 `errors.go` - 领域错误

```go
package product

import "errors"

var (
	ErrProductNotFound    = errors.New("商品不存在")
	ErrSKUNotFound        = errors.New("商品规格不存在")
	ErrCategoryNotFound   = errors.New("分类不存在")
	ErrInsufficientStock  = errors.New("库存不足")
	ErrProductOffShelf    = errors.New("商品已下架")
	ErrEmptySKUs          = errors.New("商品至少需要一个规格")
	ErrDuplicateSKUCode   = errors.New("商品规格编码重复")
	ErrInvalidPrice       = errors.New("价格必须大于0")
	ErrInvalidStock       = errors.New("库存不能为负数")
	ErrEmptyProductName   = errors.New("商品名称不能为空")
	ErrEmptyCategoryName  = errors.New("分类名称不能为空")
	ErrCategoryHasProduct = errors.New("分类下还有商品，不能删除")
)
```

### 2.2 `sku.go` - SKU 值对象

```go
package product

// SKU 是商品规格（值对象）。
// 一个商品有多个 SKU（如"红色 L 码"、"蓝色 M 码"），每个 SKU 独立价格和库存。
// 没有独立身份，作为 Product 聚合根的一部分存在。
type SKU struct {
	SKUCode string // 规格编码，商品内唯一，如 "RED-L"
	Spec    string // 规格描述，如 "红色 / L码"
	Price   int64  // 单价，单位：分
	Stock   int    // 库存
}

// validate 校验 SKU 字段
func (s SKU) validate() error {
	if s.SKUCode == "" {
		return ErrDuplicateSKUCode
	}
	if s.Price <= 0 {
		return ErrInvalidPrice
	}
	if s.Stock < 0 {
		return ErrInvalidStock
	}
	return nil
}
```

**设计点**：`SKU` 和 order 的 `OrderItem` 一样是值对象，由 `Product` 管理。`SKUCode` 在商品内唯一（用于下单时定位具体规格）。

### 2.3 `product.go` - 商品聚合根

```go
package product

import (
	"time"
)

// 商品状态
const (
	StatusOnShelf  = 1 // 上架
	StatusOffShelf = 0 // 下架
)

// Product 是商品聚合根，管理 SKU 列表。
type Product struct {
	ProductID  uint
	CategoryID uint
	Name       string
	Desc       string
	Status     int // 上下架状态
	SKUs       []SKU

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewProduct 创建商品（默认上架）。
// 充血构造函数：校验名称、SKU 列表、SKU 编码不重复。
func NewProduct(categoryID uint, name, desc string, skus []SKU) (*Product, error) {
	if name == "" {
		return nil, ErrEmptyProductName
	}
	if len(skus) == 0 {
		return nil, ErrEmptySKUs
	}
	seen := make(map[string]bool, len(skus))
	for _, s := range skus {
		if err := s.validate(); err != nil {
			return nil, err
		}
		if seen[s.SKUCode] {
			return nil, ErrDuplicateSKUCode
		}
		seen[s.SKUCode] = true
	}
	return &Product{
		CategoryID: categoryID,
		Name:       name,
		Desc:       desc,
		Status:     StatusOnShelf,
		SKUs:       skus,
	}, nil
}

// UpdateInfo 修改商品基础信息（名称、描述、分类）。
func (p *Product) UpdateInfo(categoryID uint, name, desc string) error {
	if name == "" {
		return ErrEmptyProductName
	}
	p.CategoryID = categoryID
	p.Name = name
	p.Desc = desc
	return nil
}

// ReplaceSKUs 替换全部 SKU（管理端重新设置规格）。
// 校验编码不重复。
func (p *Product) ReplaceSKUs(skus []SKU) error {
	if len(skus) == 0 {
		return ErrEmptySKUs
	}
	seen := make(map[string]bool, len(skus))
	for _, s := range skus {
		if err := s.validate(); err != nil {
			return nil
		}
		if seen[s.SKUCode] {
			return ErrDuplicateSKUCode
		}
		seen[s.SKUCode] = true
	}
	p.SKUs = skus
	return nil
}

// OnShelf 上架
func (p *Product) OnShelf() { p.Status = StatusOnShelf }

// OffShelf 下架
func (p *Product) OffShelf() { p.Status = StatusOffShelf }

// IsOnShelf 是否上架
func (p *Product) IsOnShelf() bool { return p.Status == StatusOnShelf }

// findSKU 按 code 找 SKU
func (p *Product) findSKU(code string) (int, bool) {
	for i, s := range p.SKUs {
		if s.SKUCode == code {
			return i, true
		}
	}
	return 0, false
}

// DeductStock 扣减某 SKU 的库存（下单时调用）。
// 在聚合根内校验：库存不足返回 ErrInsufficientStock，下架商品返回 ErrProductOffShelf。
// 注意：内存层校验后，仓储层还要用条件更新防并发超卖。
func (p *Product) DeductStock(skuCode string, qty int) error {
	if !p.IsOnShelf() {
		return ErrProductOffShelf
	}
	if qty <= 0 {
		return ErrInvalidStock
	}
	idx, ok := p.findSKU(skuCode)
	if !ok {
		return ErrSKUNotFound
	}
	if p.SKUs[idx].Stock < qty {
		return ErrInsufficientStock
	}
	p.SKUs[idx].Stock -= qty
	return nil
}

// Restock 增加某 SKU 库存（管理端补货）
func (p *Product) Restock(skuCode string, qty int) error {
	if qty <= 0 {
		return ErrInvalidStock
	}
	idx, ok := p.findSKU(skuCode)
	if !ok {
		return ErrSKUNotFound
	}
	p.SKUs[idx].Stock += qty
	return nil
}
```

**设计点**：
1. `NewProduct` 校验名称、SKU 非空、SKU 编码不重复。
2. `DeductStock` 在聚合根内做库存校验（内存层），但**并发安全要靠仓储层**的条件更新（第 4 步）。
3. `ReplaceSKUs` 提供整体替换 SKU 的能力（管理端改规格）。

### 2.4 `category.go` - 分类聚合根

```go
package category

import "time"

// Category 是商品分类聚合根。
// 独立于 Product，Product 通过 CategoryID 引用。
type Category struct {
	CategoryID uint
	Name       string
	ParentID   uint // 父分类ID，0 表示顶级分类
	Sort       int  // 排序权重，越小越靠前

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewCategory 创建分类
func NewCategory(name string, parentID uint, sort int) (*Category, error) {
	if name == "" {
		return nil, ErrEmptyCategoryName
	}
	return &Category{
		Name:     name,
		ParentID: parentID,
		Sort:     sort,
	}, nil
}

// Rename 重命名分类
func (c *Category) Rename(name string) error {
	if name == "" {
		return ErrEmptyCategoryName
	}
	c.Name = name
	return nil
}
```

> ⚠️ 注意：`category.go` 我放进了 `package category`，即**独立的子包** `domain/category/`，而不是 `domain/product/category.go`。
> 原因：分类是一个独立的聚合根，有自己的生命周期和仓储，不该塞进 product 包。
> 所以这个文件实际路径是 **`internal/domain/category/category.go`**，包名 `category`。
> 对应的 `ErrEmptyCategoryName` 要放在 `internal/domain/category/errors.go` 里（见 2.5）。

### 2.5 `internal/domain/category/errors.go` - 分类领域错误

```go
package category

import "errors"

var (
	ErrEmptyCategoryName  = errors.New("分类名称不能为空")
	ErrCategoryNotFound   = errors.New("分类不存在")
	ErrCategoryHasProduct = errors.New("分类下还有商品，不能删除")
)
```

> 注意 product 包里也有 `ErrCategoryNotFound`/`ErrCategoryHasProduct` 的定义（2.1），
> 那是给 product 应用层用的别名；category 包自己定义一份供 category 应用层用。
> 如果你嫌重复，可以只在 category 包定义，product 包里删掉这两个，用的时候 import category 包。

### 2.6 `internal/domain/product/repository.go` - 商品仓储端口

```go
package product

import "context"

// ProductRepository 商品聚合根仓储端口
type ProductRepository interface {
	// FindByID 按主键查询商品（含 SKU）；未找到返回 ErrProductNotFound
	FindByID(ctx context.Context, id uint) (*Product, error)
	// FindBySKUCode 按 SKU 编码查询商品（用于下单扣库存），返回商品与 SKU 索引
	FindBySKUCode(ctx context.Context, skuCode string) (*Product, error)
	// Search 分页查询商品，支持按分类/名称筛选，返回列表与总数
	Search(ctx context.Context, categoryID uint, keyword string, page, pageSize int) ([]*Product, int64, error)
	// Save 新增或更新商品（含 SKU）
	Save(ctx context.Context, p *Product) error
	// DeductStock 原子扣减某 SKU 库存，并发安全；库存不足返回 ErrInsufficientStock
	DeductStock(ctx context.Context, productID uint, skuCode string, qty int) error
	// CountByCategory 统计某分类下的商品数量（删除分类前校验）
	CountByCategory(ctx context.Context, categoryID uint) (int64, error)
}
```

**设计点**：
- `DeductStock` 单独作为仓储方法，用 `UPDATE ... SET stock=stock-? WHERE product_id=? AND sku_code=? AND stock>=?` 原子操作防超卖。
- `FindBySKUCode` 用于下单时根据订单里的 SKU 编码找到商品。
- `CountByCategory` 给删除分类时做安全校验。

### 2.7 `internal/domain/category/repository.go` - 分类仓储端口

```go
package category

import "context"

// CategoryRepository 分类聚合根仓储端口
type CategoryRepository interface {
	FindByID(ctx context.Context, id uint) (*Category, error)
	// FindAll 查询全部分类（分类数量通常不多，不分页），可按 sort 排序
	FindAll(ctx context.Context) ([]*Category, error)
	Save(ctx context.Context, c *Category) error
	Delete(ctx context.Context, id uint) error
}
```

### ✅ 第 2 步自检
- [ ] `go build ./internal/domain/product/ ./internal/domain/category/` 编译通过
- 自检问题：为什么 `DeductStock` 既在聚合根里有方法，又在仓储里单独有一个？（聚合根做内存校验，仓储做并发安全的原子更新）

---

## 第 3 步：商品应用层 `internal/application/product/`

### 3.1 `dto.go`

```go
package productapp

import "time"

// ---- 输入 DTO ----

type SKUInput struct {
	SKUCode string `json:"skuCode"`
	Spec    string `json:"spec"`
	Price   int64  `json:"price"`  // 分
	Stock   int    `json:"stock"`
}

type CreateProductInput struct {
	CategoryID uint       `json:"categoryId"`
	Name       string     `json:"name"`
	Desc       string     `json:"desc"`
	SKUs       []SKUInput `json:"skus"`
}

type UpdateProductInput struct {
	ProductID  uint       `json:"-"`
	CategoryID uint       `json:"categoryId"`
	Name       string     `json:"name"`
	Desc       string     `json:"desc"`
	SKUs       []SKUInput `json:"skus"` // 整体替换 SKU
}

type DeductStockInput struct {
	SKUCode string `json:"skuCode"`
	Qty     int    `json:"qty"`
}

type SearchProductInput struct {
	CategoryID uint   `json:"-"` // 从 query 取
	Keyword    string `json:"-"`
	Page       int
	PageSize   int
}

// ---- 输出 DTO ----

type SKUDTO struct {
	SKUCode string `json:"skuCode"`
	Spec    string `json:"spec"`
	Price   int64  `json:"price"`
	Stock   int    `json:"stock"`
}

type ProductDTO struct {
	ProductID  uint       `json:"productId"`
	CategoryID uint       `json:"categoryId"`
	Name       string     `json:"name"`
	Desc       string     `json:"desc"`
	Status     int        `json:"status"`
	SKUs       []SKUDTO   `json:"skus"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type ProductListResult struct {
	List     []ProductDTO `json:"list"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}
```

### 3.2 `service.go`

```go
package productapp

import (
	"context"
	"errors"

	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

type Service struct {
	repo domainproduct.ProductRepository
}

func NewService(repo domainproduct.ProductRepository) *Service {
	return &Service{repo: repo}
}

// Create 创建商品
func (s *Service) Create(ctx context.Context, in CreateProductInput) (*ProductDTO, error) {
	skus := toDomainSKUs(in.SKUs)
	p, err := domainproduct.NewProduct(in.CategoryID, in.Name, in.Desc, skus)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	dto := toProductDTO(p)
	return &dto, nil
}

// Update 更新商品基础信息并整体替换 SKU
func (s *Service) Update(ctx context.Context, in UpdateProductInput) (*ProductDTO, error) {
	p, err := s.repo.FindByID(ctx, in.ProductID)
	if err != nil {
		return nil, err
	}
	if err := p.UpdateInfo(in.CategoryID, in.Name, in.Desc); err != nil {
		return nil, err
	}
	if err := p.ReplaceSKUs(toDomainSKUs(in.SKUs)); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	dto := toProductDTO(p)
	return &dto, nil
}

// OnShelf 上架
func (s *Service) OnShelf(ctx context.Context, productID uint) error {
	p, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	p.OnShelf()
	return s.repo.Save(ctx, p)
}

// OffShelf 下架
func (s *Service) OffShelf(ctx context.Context, productID uint) error {
	p, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	p.OffShelf()
	return s.repo.Save(ctx, p)
}

// GetByID 查询商品详情
func (s *Service) GetByID(ctx context.Context, productID uint) (*ProductDTO, error) {
	p, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	dto := toProductDTO(p)
	return &dto, nil
}

// Search 分页查询商品
func (s *Service) Search(ctx context.Context, in SearchProductInput) (*ProductListResult, error) {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PageSize < 1 || in.PageSize > 100 {
		in.PageSize = 10
	}
	list, total, err := s.repo.Search(ctx, in.CategoryID, in.Keyword, in.Page, in.PageSize)
	if err != nil {
		return nil, err
	}
	dtos := make([]ProductDTO, 0, len(list))
	for _, p := range list {
		dtos = append(dtos, toProductDTO(p))
	}
	return &ProductListResult{
		List:     dtos,
		Total:    total,
		Page:     in.Page,
		PageSize: in.PageSize,
	}, nil
}

// DeductStock 扣减库存（供 order 模块下单时调用）。
// 优先用仓储的原子扣减，保证并发安全。
func (s *Service) DeductStock(ctx context.Context, productID uint, in DeductStockInput) error {
	// 先查出商品做内存校验（下架、SKU 是否存在、库存是否足够）
	p, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if err := p.DeductStock(in.SKUCode, in.Qty); err != nil {
		return err
	}
	// 再用仓储的原子操作真正扣减（防并发超卖）
	if err := s.repo.DeductStock(ctx, productID, in.SKUCode, in.Qty); err != nil {
		// 原子扣减失败说明并发下库存不足，统一返回库存不足
		if !errors.Is(err, domainproduct.ErrInsufficientStock) {
			return err
		}
		return domainproduct.ErrInsufficientStock
	}
	return nil
}

// CountByCategory 查某分类下商品数（删除分类前校验用）
func (s *Service) CountByCategory(ctx context.Context, categoryID uint) (int64, error) {
	return s.repo.CountByCategory(ctx, categoryID)
}

// ---- 转换函数 ----

func toDomainSKUs(in []SKUInput) []domainproduct.SKU {
	skus := make([]domainproduct.SKU, 0, len(in))
	for _, s := range in {
		skus = append(skus, domainproduct.SKU{
			SKUCode: s.SKUCode,
			Spec:    s.Spec,
			Price:   s.Price,
			Stock:   s.Stock,
		})
	}
	return skus
}

func toProductDTO(p *domainproduct.Product) ProductDTO {
	skus := make([]SKUDTO, 0, len(p.SKUs))
	for _, s := range p.SKUs {
		skus = append(skus, SKUDTO{
			SKUCode: s.SKUCode,
			Spec:    s.Spec,
			Price:   s.Price,
			Stock:   s.Stock,
		})
	}
	return ProductDTO{
		ProductID:  p.ProductID,
		CategoryID: p.CategoryID,
		Name:       p.Name,
		Desc:       p.Desc,
		Status:     p.Status,
		SKUs:       skus,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}
```

**设计点**：`DeductStock` 做了两层校验：聚合根内存校验（快速失败，给出明确错误）+ 仓储原子更新（真正防超卖）。内存校验不是必须的，但能让"下架商品""SKU 不存在"这类错误提前暴露，不用走到 SQL。

### ✅ 第 3 步自检
- [ ] `go build ./internal/application/product/` 编译通过

---

## 第 4 步：商品基础设施层 `internal/infrastructure/persistence/product/`

### 4.1 `model.go`

```go
package productpo

import (
	"time"

	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

// ProductPO 商品持久化对象，对应 products 表
type ProductPO struct {
	ProductID  uint   `gorm:"primaryKey;autoIncrement;comment:商品主键"`
	CategoryID uint   `gorm:"index;comment:分类ID"`
	Name       string `gorm:"size:128;comment:商品名称"`
	Desc       string `gorm:"size:512;comment:商品描述"`
	Status     int    `gorm:"tinyint;not null;default:1;comment:状态 1上架 0下架"`

	CreatedAt time.Time
	UpdatedAt time.Time

	// 一对多：商品包含的 SKU
	SKUs []SKUPO `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE"`
}

// SKUPO 商品规格持久化对象，对应 product_skus 表
type SKUPO struct {
	SKUID      uint   `gorm:"primaryKey;autoIncrement;comment:规格主键"`
	ProductID  uint   `gorm:"index;uniqueIndex:uk_product_sku;comment:所属商品ID"`
	SKUCode    string `gorm:"size:64;uniqueIndex:uk_product_sku;comment:规格编码(商品内唯一)"`
	Spec       string `gorm:"size:128;comment:规格描述"`
	Price      int64  `gorm:"comment:单价(分)"`
	Stock      int    `gorm:"not null;default:0;comment:库存"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ProductPO) TableName() string { return "products" }
func (SKUPO) TableName() string    { return "product_skus" }

// toDomain 商品 PO -> 领域聚合根（含 SKU）
func toDomain(po *ProductPO) *domainproduct.Product {
	skus := make([]domainproduct.SKU, 0, len(po.SKUs))
	for _, s := range po.SKUs {
		skus = append(skus, domainproduct.SKU{
			SKUCode: s.SKUCode,
			Spec:    s.Spec,
			Price:   s.Price,
			Stock:   s.Stock,
		})
	}
	return &domainproduct.Product{
		ProductID:  po.ProductID,
		CategoryID: po.CategoryID,
		Name:       po.Name,
		Desc:       po.Desc,
		Status:     po.Status,
		SKUs:       skus,
		CreatedAt:  po.CreatedAt,
		UpdatedAt:  po.UpdatedAt,
	}
}

// toPO 领域聚合根 -> 商品 PO（主表，不含 SKU，更新时用）
func toPO(p *domainproduct.Product) *ProductPO {
	return &ProductPO{
		ProductID:  p.ProductID,
		CategoryID: p.CategoryID,
		Name:       p.Name,
		Desc:       p.Desc,
		Status:     p.Status,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

// toSKUsPO 领域 SKU 列表 -> PO 列表（新增/替换时用）
func toSKUsPO(skus []domainproduct.SKU) []SKUPO {
	result := make([]SKUPO, 0, len(skus))
	for _, s := range skus {
		result = append(result, SKUPO{
			SKUCode: s.SKUCode,
			Spec:    s.Spec,
			Price:   s.Price,
			Stock:   s.Stock,
		})
	}
	return result
}
```

**设计点**：
- `uniqueIndex:uk_product_sku` 联合唯一索引（ProductID + SKUCode），保证同一商品下 SKU 编码不重复。
- `toPO` 不含 SKUs（和 order 一样的套路，更新时只存主表）。

### 4.2 `repository.go`

```go
package productpo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

var _ domainproduct.ProductRepository = (*ProductRepository)(nil)

func (r *ProductRepository) FindByID(ctx context.Context, id uint) (*domainproduct.Product, error) {
	var po ProductPO
	err := r.db.WithContext(ctx).Preload("SKUs").First(&po, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainproduct.ErrProductNotFound
		}
		return nil, err
	}
	return toDomain(&po), nil
}

func (r *ProductRepository) FindBySKUCode(ctx context.Context, skuCode string) (*domainproduct.Product, error) {
	var sku SKUPO
	err := r.db.WithContext(ctx).Where("sku_code = ?", skuCode).First(&sku).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainproduct.ErrSKUNotFound
		}
		return nil, err
	}
	return r.FindByID(ctx, sku.ProductID)
}

func (r *ProductRepository) Search(ctx context.Context, categoryID uint, keyword string, page, pageSize int) ([]*domainproduct.Product, int64, error) {
	var (
		pos   []ProductPO
		total int64
	)
	db := r.db.WithContext(ctx).Model(&ProductPO{})
	if categoryID > 0 {
		db = db.Where("category_id = ?", categoryID)
	}
	if keyword != "" {
		db = db.Where("name LIKE ?", "%"+keyword+"%")
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := db.Preload("SKUs").Order("product_id DESC").Limit(pageSize).Offset(offset).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	list := make([]*domainproduct.Product, 0, len(pos))
	for i := range pos {
		list = append(list, toDomain(&pos[i]))
	}
	return list, total, nil
}

func (r *ProductRepository) Save(ctx context.Context, p *domainproduct.Product) error {
	po := toPO(p)
	if p.ProductID == 0 {
		// 新增：级联创建商品 + SKU
		po.SKUs = toSKUsPO(p.SKUs)
		if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
			return err
		}
		p.ProductID = po.ProductID
		p.CreatedAt = po.CreatedAt
		p.UpdatedAt = po.UpdatedAt
		return nil
	}
	// 更新：事务里更新主表 + 替换 SKU（先删后插）
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(po).Error; err != nil {
			return err
		}
		// 替换 SKU：先删旧的，再插新的
		if err := tx.Where("product_id = ?", p.ProductID).Delete(&SKUPO{}).Error; err != nil {
			return err
		}
		newSKUs := toSKUsPO(p.SKUs)
		for i := range newSKUs {
			newSKUs[i].ProductID = p.ProductID
		}
		if len(newSKUs) > 0 {
			if err := tx.Create(&newSKUs).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeductStock 原子扣减库存，防并发超卖。
// 用 WHERE stock >= qty 条件更新，影响行数为 0 即库存不足。
func (r *ProductRepository) DeductStock(ctx context.Context, productID uint, skuCode string, qty int) error {
	result := r.db.WithContext(ctx).
		Model(&SKUPO{}).
		Where("product_id = ? AND sku_code = ? AND stock >= ?", productID, skuCode, qty).
		Update("stock", gorm.Expr("stock - ?", qty))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainproduct.ErrInsufficientStock
	}
	return nil
}

func (r *ProductRepository) CountByCategory(ctx context.Context, categoryID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&ProductPO{}).Where("category_id = ?", categoryID).Count(&count).Error
	return count, err
}
```

**设计点**：
1. **`DeductStock` 防超卖核心**：`WHERE stock >= qty` + `UPDATE stock = stock - qty`，单条 SQL 原子完成。`RowsAffected == 0` 说明库存不足（或 SKU 不存在），返回 `ErrInsufficientStock`。这是防并发超卖的标准做法。
2. **更新时替换 SKU**：用事务"先删后插"，因为 SKU 列表是整体替换（管理端重新设规格）。和 order 不同--order 的明细创建后不可变，商品的 SKU 可以改。
3. **`Save` 用事务**：保证"更新主表 + 替换 SKU"要么全成功要么全失败。

### ✅ 第 4 步自检
- [ ] `go build ./internal/infrastructure/persistence/product/` 编译通过
- 自检问题：为什么 `DeductStock` 用 `WHERE stock >= qty` 而不是先查再减？（防并发：两个请求同时查到 stock=1，各自判断够，都去减就超卖了）

---

## 第 5 步：分类应用层 + 基础设施层

### 5.1 `internal/domain/category/category.go` 和 `errors.go`、`repository.go`

见第 2 步的 2.4、2.5、2.7（已经写过了）。

### 5.2 `internal/infrastructure/persistence/category/model.go`

```go
package categorypo

import (
	"time"

	domaincategory "github.com/wsc-zz/service/internal/domain/category"
)

type CategoryPO struct {
	CategoryID uint   `gorm:"primaryKey;autoIncrement;comment:分类主键"`
	Name       string `gorm:"size:64;comment:分类名称"`
	ParentID   uint   `gorm:"index;default:0;comment:父分类ID，0为顶级"`
	Sort       int    `gorm:"default:0;comment:排序权重"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (CategoryPO) TableName() string { return "categories" }

func toDomain(po *CategoryPO) *domaincategory.Category {
	return &domaincategory.Category{
		CategoryID: po.CategoryID,
		Name:       po.Name,
		ParentID:   po.ParentID,
		Sort:       po.Sort,
		CreatedAt:  po.CreatedAt,
		UpdatedAt:  po.UpdatedAt,
	}
}

func toPO(c *domaincategory.Category) *CategoryPO {
	return &CategoryPO{
		CategoryID: c.CategoryID,
		Name:       c.Name,
		ParentID:   c.ParentID,
		Sort:       c.Sort,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}
```

### 5.3 `internal/infrastructure/persistence/category/repository.go`

```go
package categorypo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	domaincategory "github.com/wsc-zz/service/internal/domain/category"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

var _ domaincategory.CategoryRepository = (*CategoryRepository)(nil)

func (r *CategoryRepository) FindByID(ctx context.Context, id uint) (*domaincategory.Category, error) {
	var po CategoryPO
	err := r.db.WithContext(ctx).First(&po, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaincategory.ErrCategoryNotFound
		}
		return nil, err
	}
	return toDomain(&po), nil
}

func (r *CategoryRepository) FindAll(ctx context.Context) ([]*domaincategory.Category, error) {
	var pos []CategoryPO
	if err := r.db.WithContext(ctx).Order("sort ASC, category_id ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	list := make([]*domaincategory.Category, 0, len(pos))
	for i := range pos {
		list = append(list, toDomain(&pos[i]))
	}
	return list, nil
}

func (r *CategoryRepository) Save(ctx context.Context, c *domaincategory.Category) error {
	po := toPO(c)
	if c.CategoryID == 0 {
		if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
			return err
		}
		c.CategoryID = po.CategoryID
		c.CreatedAt = po.CreatedAt
		c.UpdatedAt = po.UpdatedAt
		return nil
	}
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *CategoryRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&CategoryPO{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domaincategory.ErrCategoryNotFound
	}
	return nil
}
```

### 5.4 `internal/application/category/dto.go`

```go
package categoryapp

import "time"

type CreateCategoryInput struct {
	Name     string `json:"name"`
	ParentID uint   `json:"parentId"`
	Sort     int    `json:"sort"`
}

type UpdateCategoryInput struct {
	Name string `json:"name"`
	Sort int    `json:"sort"`
}

type CategoryDTO struct {
	CategoryID uint      `json:"categoryId"`
	Name       string    `json:"name"`
	ParentID   uint      `json:"parentId"`
	Sort       int       `json:"sort"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
```

### 5.5 `internal/application/category/service.go`

```go
package categoryapp

import (
	"context"

	domaincategory "github.com/wsc-zz/service/internal/domain/category"
	domainproduct "github.com/wsc-zz/service/internal/domain/product"
)

type Service struct {
	repo        domaincategory.CategoryRepository
	productRepo domainproduct.ProductRepository // 删除分类前检查是否有关联商品
}

func NewService(repo domaincategory.CategoryRepository, productRepo domainproduct.ProductRepository) *Service {
	return &Service{repo: repo, productRepo: productRepo}
}

func (s *Service) Create(ctx context.Context, in CreateCategoryInput) (*CategoryDTO, error) {
	c, err := domaincategory.NewCategory(in.Name, in.ParentID, in.Sort)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, c); err != nil {
		return nil, err
	}
	dto := toCategoryDTO(c)
	return &dto, nil
}

func (s *Service) Update(ctx context.Context, id uint, in UpdateCategoryInput) (*CategoryDTO, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := c.Rename(in.Name); err != nil {
		return nil, err
	}
	c.Sort = in.Sort
	if err := s.repo.Save(ctx, c); err != nil {
		return nil, err
	}
	dto := toCategoryDTO(c)
	return &dto, nil
}

func (s *Service) Delete(ctx context.Context, id uint) error {
	// 安全校验：分类下有商品则禁止删除
	count, err := s.productRepo.CountByCategory(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return domaincategory.ErrCategoryHasProduct
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) FindAll(ctx context.Context) ([]CategoryDTO, error) {
	list, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	dtos := make([]CategoryDTO, 0, len(list))
	for _, c := range list {
		dtos = append(dtos, toCategoryDTO(c))
	}
	return dtos, nil
}

func toCategoryDTO(c *domaincategory.Category) CategoryDTO {
	return CategoryDTO{
		CategoryID: c.CategoryID,
		Name:       c.Name,
		ParentID:   c.ParentID,
		Sort:       c.Sort,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}
```

**设计点**：`categoryapp.Service` 依赖 `ProductRepository`（跨聚合），目的是删除分类前检查"是否还有商品引用"。这是跨聚合协作的合法场景（一个用例需要两个聚合的信息）。

### ✅ 第 5 步自检
- [ ] `go build ./internal/infrastructure/persistence/category/ ./internal/application/category/` 编译通过

---

## 第 6 步：接口层 handler + router

### 6.1 新建 `handler/product.go`

```go
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	categoryapp "github.com/wsc-zz/service/internal/application/category"
	productapp "github.com/wsc-zz/service/internal/application/product"
	domainproduct "github.com/wsc-zz/service/internal/domain/product"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

type ProductHandler struct {
	productSvc  *productapp.Service
	categorySvc *categoryapp.Service
}

func NewProductHandler(productSvc *productapp.Service, categorySvc *categoryapp.Service) *ProductHandler {
	return &ProductHandler{productSvc: productSvc, categorySvc: categorySvc}
}

// ---- 商品请求结构体 ----

type skuRequest struct {
	SKUCode string `json:"skuCode" binding:"required"`
	Spec    string `json:"spec" binding:"required"`
	Price   int64  `json:"price" binding:"required,min=1"`
	Stock   int    `json:"stock" binding:"min=0"`
}

type createProductRequest struct {
	CategoryID uint         `json:"categoryId" binding:"required"`
	Name       string       `json:"name" binding:"required,min=1,max=128"`
	Desc       string       `json:"desc"`
	SKUs       []skuRequest `json:"skus" binding:"required,min=1,dive"`
}

type updateProductRequest struct {
	CategoryID uint         `json:"categoryId" binding:"required"`
	Name       string       `json:"name" binding:"required,min=1,max=128"`
	Desc       string       `json:"desc"`
	SKUs       []skuRequest `json:"skus" binding:"required,min=1,dive"`
}

type searchProductRequest struct {
	CategoryID uint   `form:"categoryId"`
	Keyword    string `form:"keyword"`
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// ---- 商品 handler ----

// Create 创建商品（管理员）
func (h *ProductHandler) Create(c *gin.Context) {
	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	result, err := h.productSvc.Create(c.Request.Context(), productapp.CreateProductInput{
		CategoryID: req.CategoryID,
		Name:       req.Name,
		Desc:       req.Desc,
		SKUs:       toSKUInputs(req.SKUs),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "createProduct", result)
}

// Update 更新商品（管理员）
func (h *ProductHandler) Update(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "商品ID格式错误")
		return
	}
	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	result, err := h.productSvc.Update(c.Request.Context(), productapp.UpdateProductInput{
		ProductID:  uint(productID),
		CategoryID: req.CategoryID,
		Name:       req.Name,
		Desc:       req.Desc,
		SKUs:       toSKUInputs(req.SKUs),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "updateProduct", result)
}

// OnShelf 上架（管理员）
func (h *ProductHandler) OnShelf(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "商品ID格式错误")
		return
	}
	if err := h.productSvc.OnShelf(c.Request.Context(), uint(productID)); err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "onShelf", nil)
}

// OffShelf 下架（管理员）
func (h *ProductHandler) OffShelf(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "商品ID格式错误")
		return
	}
	if err := h.productSvc.OffShelf(c.Request.Context(), uint(productID)); err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "offShelf", nil)
}

// GetByID 商品详情（公开）
func (h *ProductHandler) GetByID(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "商品ID格式错误")
		return
	}
	result, err := h.productSvc.GetByID(c.Request.Context(), uint(productID))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "getProduct", result)
}

// Search 商品列表（公开）
func (h *ProductHandler) Search(c *gin.Context) {
	var req searchProductRequest
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
	result, err := h.productSvc.Search(c.Request.Context(), productapp.SearchProductInput{
		CategoryID: req.CategoryID,
		Keyword:    req.Keyword,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.SuccessMsg(c, "listProducts", result)
}

// ---- 分类 handler ----

type createCategoryRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=64"`
	ParentID uint   `json:"parentId"`
	Sort     int    `json:"sort"`
}

type updateCategoryRequest struct {
	Name string `json:"name" binding:"required,min=1,max=64"`
	Sort int    `json:"sort"`
}

func (h *ProductHandler) CreateCategory(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	result, err := h.categorySvc.Create(c.Request.Context(), categoryapp.CreateCategoryInput{
		Name:     req.Name,
		ParentID: req.ParentID,
		Sort:     req.Sort,
	})
	if err != nil {
		h.writeCategoryError(c, err)
		return
	}
	response.SuccessMsg(c, "createCategory", result)
}

func (h *ProductHandler) UpdateCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "分类ID格式错误")
		return
	}
	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, http.StatusBadRequest, validator.ErrorMsg(err))
		return
	}
	result, err := h.categorySvc.Update(c.Request.Context(), uint(id), categoryapp.UpdateCategoryInput{
		Name: req.Name,
		Sort: req.Sort,
	})
	if err != nil {
		h.writeCategoryError(c, err)
		return
	}
	response.SuccessMsg(c, "updateCategory", result)
}

func (h *ProductHandler) DeleteCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, http.StatusBadRequest, "分类ID格式错误")
		return
	}
	if err := h.categorySvc.Delete(c.Request.Context(), uint(id)); err != nil {
		h.writeCategoryError(c, err)
		return
	}
	response.SuccessMsg(c, "deleteCategory", nil)
}

func (h *ProductHandler) ListCategories(c *gin.Context) {
	list, err := h.categorySvc.FindAll(c.Request.Context())
	if err != nil {
		h.writeCategoryError(c, err)
		return
	}
	response.SuccessMsg(c, "listCategories", list)
}

// ---- 错误映射 ----

func (h *ProductHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainproduct.ErrProductNotFound),
		errors.Is(err, domainproduct.ErrSKUNotFound):
		response.NotFound(c, http.StatusNotFound, err.Error())
	case errors.Is(err, domainproduct.ErrInsufficientStock),
		errors.Is(err, domainproduct.ErrProductOffShelf),
		errors.Is(err, domainproduct.ErrEmptySKUs),
		errors.Is(err, domainproduct.ErrDuplicateSKUCode),
		errors.Is(err, domainproduct.ErrInvalidPrice),
		errors.Is(err, domainproduct.ErrInvalidStock),
		errors.Is(err, domainproduct.ErrEmptyProductName):
		response.BadRequest(c, http.StatusBadRequest, err.Error())
	default:
		response.ServerError(c, http.StatusInternalServerError, err.Error())
	}
}

func (h *ProductHandler) writeCategoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainproduct.ErrCategoryNotFound):
		response.NotFound(c, http.StatusNotFound, err.Error())
	case errors.Is(err, domainproduct.ErrCategoryHasProduct),
		errors.Is(err, domainproduct.ErrEmptyCategoryName):
		response.BadRequest(c, http.StatusBadRequest, err.Error())
	default:
		response.ServerError(c, http.StatusInternalServerError, err.Error())
	}
}

func toSKUInputs(req []skuRequest) []productapp.SKUInput {
	skus := make([]productapp.SKUInput, 0, len(req))
	for _, s := range req {
		skus = append(skus, productapp.SKUInput{
			SKUCode: s.SKUCode,
			Spec:    s.Spec,
			Price:   s.Price,
			Stock:   s.Stock,
		})
	}
	return skus
}
```

> 注意：这里把分类的 handler 也放进了 `ProductHandler`（同一个文件），因为商品和分类紧密相关。如果你喜欢分离，可以单独建 `handler/category.go` + `CategoryHandler`，但需要调整构造函数。

### 6.2 修改 `router/router.go`

`InitRouter` 签名加 `productSvc`、`categorySvc`，加商品/分类路由（公开 vs 管理员分组）：

```go
package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/wsc-zz/service/internal/application/category"
	"github.com/wsc-zz/service/internal/application/order"
	"github.com/wsc-zz/service/internal/application/product"
	"github.com/wsc-zz/service/internal/application/user"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	"github.com/wsc-zz/service/internal/interfaces/http/middleware"
)

func InitRouter(
	userSvc *userapp.Service,
	orderSvc *orderapp.Service,
	productSvc *productapp.Service,
	categorySvc *categoryapp.Service,
) *gin.Engine {
	r := gin.Default()

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
	productH := handler.NewProductHandler(productSvc, categorySvc)

	apiGroup := r.Group("/api")
	{
		// 用户
		apiGroup.POST("/user/register", h.Register)
		apiGroup.POST("/user/login", h.Login)

		// 商品浏览：公开（无需登录）
		apiGroup.GET("/product", productH.Search)
		apiGroup.GET("/product/:id", productH.GetByID)
		// 分类浏览：公开
		apiGroup.GET("/category", productH.ListCategories)

		// 商品管理：需要管理员
		adminProduct := apiGroup.Group("/product", middleware.JWTAuth(), middleware.AdminAuth())
		{
			adminProduct.POST("", productH.Create)
			adminProduct.PUT("/:id", productH.Update)
			adminProduct.POST("/:id/onshelf", productH.OnShelf)
			adminProduct.POST("/:id/offshelf", productH.OffShelf)
		}
		// 分类管理：需要管理员
		adminCategory := apiGroup.Group("/category", middleware.JWTAuth(), middleware.AdminAuth())
		{
			adminCategory.POST("", productH.CreateCategory)
			adminCategory.PUT("/:id", productH.UpdateCategory)
			adminCategory.DELETE("/:id", productH.DeleteCategory)
		}

		// 订单：需要登录
		orderGroup := apiGroup.Group("/order", middleware.JWTAuth())
		{
			orderGroup.POST("", orderH.Create)
			orderGroup.GET("/:id", orderH.GetByID)
		}
	}
	return r
}
```

> ⚠️ `InitRouter` 签名又变了，`main.go` 调用处要同步（第 8 步）。

### ✅ 第 6 步自检
- [ ] `go build ./internal/interfaces/http/...` 编译通过

---

## 第 7 步：order 下单集成库存扣减（跨模块协作）

下单时，对订单里每个商品项，调用 `productSvc.DeductStock` 扣减库存。这是 order 依赖 product 的跨模块协作。

### 7.1 修改 `application/order/service.go` 的 `Service` 与 `Create`

```go
type Service struct {
	repo       domainorder.OrderRepository
	productSvc domainproduct.StockDeductor // 新增：依赖商品库存扣减端口
}

// NewService 加 productSvc 参数
func NewService(repo domainorder.OrderRepository, productSvc domainproduct.StockDeductor) *Service {
	return &Service{repo: repo, productSvc: productSvc}
}

// Create 创建订单：先扣库存，再存订单
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
	// 扣减库存：对每个商品项调用商品服务
	for _, it := range items {
		if err := s.productSvc.DeductStock(ctx, it.ProductID, domainproduct.DeductStockInput{
			SKUCode: it.SKUCode, // 需要在 OrderItem 里加 SKUCode 字段，见 7.2
			Qty:     it.Quantity,
		}); err != nil {
			return nil, err // 库存不足等错误直接返回，订单不创建
		}
	}
	if err := s.repo.Save(ctx, o); err != nil {
		return nil, err
	}
	dto := toOrderDTO(o)
	return &dto, nil
}
```

### 7.2 `domain/order/item.go` 加 `SKUCode` 字段

订单明细需要知道买的是哪个 SKU，才能扣对应库存：

```go
type OrderItem struct {
	ProductID   uint
	SKUCode     string  // 新增：买的哪个规格
	ProductName string
	Quantity    int
	Price       int64
}
```

> 改完 `OrderItem`，order 的 PO（`OrderItemPO`）、转换函数、DTO、handler 请求体都要同步加 `SKUCode`。这是一串连锁改动，和第 1 步改 `Role` 一样，记得都改到。

### 7.3 在 `domain/product/` 定义 `StockDeductor` 端口

为了让 order 应用层**不直接依赖 product 应用层的具体类型**（解耦），在 product 领域层定义一个端口：

```go
// internal/domain/product/stock_deductor.go
package product

import "context"

// DeductStockInput 库存扣减输入（放在 product 领域层，供 order 引用）
type DeductStockInput struct {
	SKUCode string
	Qty     int
}

// StockDeductor 库存扣减端口：order 模块依赖这个接口，不依赖 productapp.Service 具体类型。
type StockDeductor interface {
	DeductStock(ctx context.Context, productID uint, in DeductStockInput) error
}
```

然后让 `productapp.Service` 满足这个接口（加编译期断言）：

```go
// internal/application/product/service.go 里加：
var _ domainproduct.StockDeductor = (*Service)(nil)
```

`productapp.Service` 已经有 `DeductStock` 方法了，签名匹配，自动实现接口。

**设计点（重要）**：
- order 依赖 `domainproduct.StockDeductor` **接口**，而不是 `productapp.Service` **具体类型**。这样 order 和 product 应用层解耦，符合依赖倒置。
- 接口定义在 product 领域层（被依赖方定义接口），order 应用层 import 它。这是"端口"模式。
- `main.go` 装配时把 `productSvc`（满足 `StockDeductor`）注入给 `orderapp.NewService`。

> 💡 这是 plan 里说的"方案 A 的解耦版"：order 依赖 product 的接口而非具体类型。比直接依赖 `*productapp.Service` 更干净，但不用引入事件机制。教学推荐这种。

### ✅ 第 7 步自检
- [ ] `go build ./...` 编译通过（注意 OrderItem 加 SKUCode 的连锁改动是否都改到）

---

## 第 8 步：组合根 main.go + 编译验证

### 8.1 修改 `cmd/main.go`

```go
package main

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
	"github.com/wsc-zz/service/internal/application/category"
	"github.com/wsc-zz/service/internal/application/order"
	"github.com/wsc-zz/service/internal/application/product"
	"github.com/wsc-zz/service/internal/application/user"
	"github.com/wsc-zz/service/internal/infrastructure/auth"
	"github.com/wsc-zz/service/internal/infrastructure/persistence/category"
	categorypo "github.com/wsc-zz/service/internal/infrastructure/persistence/category"
	"github.com/wsc-zz/service/internal/infrastructure/persistence/order"
	"github.com/wsc-zz/service/internal/infrastructure/persistence/product"
	"github.com/wsc-zz/service/internal/infrastructure/persistence/user"
	"github.com/wsc-zz/service/internal/infrastructure/security"
	"github.com/wsc-zz/service/internal/interfaces/http/router"
)

func main() {
	global.InitViper()
	global.InitZap()
	global.InitMysql()

	if err := global.DB.AutoMigrate(
		&userpo.UserPO{},
		&orderpo.OrderPO{}, &orderpo.OrderItemPO{},
		&productpo.ProductPO{}, &productpo.SKUPO{},
		&categorypo.CategoryPO{},
	); err != nil {
		global.Logger.Error("数据表迁移失败", zap.Error(err))
		panic("数据表迁移失败:" + err.Error())
	}
	global.Logger.Info("数据表迁移成功")

	// 组合根
	userRepo := userpo.NewUserRepository(global.DB)
	hasher := security.NewBcryptHasher()
	tokenIssuer := auth.NewJWTTokenIssuer()
	userSvc := userapp.NewService(userRepo, hasher, tokenIssuer)

	categoryRepo := categorypo.NewCategoryRepository(global.DB)
	productRepo := productpo.NewProductRepository(global.DB)
	productSvc := productapp.NewService(productRepo)
	categorySvc := categoryapp.NewService(categoryRepo, productRepo)

	// order 注入 productSvc 作为 StockDeductor
	orderRepo := orderpo.NewOrderRepository(global.DB)
	orderSvc := orderapp.NewService(orderRepo, productSvc)

	r := router.InitRouter(userSvc, orderSvc, productSvc, categorySvc)
	if err := r.Run(":" + fmt.Sprint(global.Conf.Service.Port)); err != nil {
		panic(err)
	}
}
```

> ⚠️ 注意 import：`persistence/category` 和 `application/category` 包名都是 `category`，会和别的冲突。上面的写法有坑（`category` 重复）。实际请用别名：
> ```go
> categoryapp "github.com/wsc-zz/service/internal/application/category"
> categorypo "github.com/wsc-zz/service/internal/infrastructure/persistence/category"
> ```
> 然后用 `categoryapp.NewService` / `categorypo.NewCategoryRepository`。
> （上面 main.go 的 import 块我故意写了重复名让你看到这个坑，请用别名修正。）

> 但 `persistence/order`、`persistence/product`、`persistence/user` 的包名是 `orderpo`/`productpo`/`userpo`（和目录名不同），所以不冲突。只有 category 的包名建议也改成 `categorypo` 保持一致。

### 8.2 编译验证
```bash
go build ./...
```

### 8.3 接口测试流程（curl）

1. 注册用户 -> 手动把 role 改成 2 -> 登录拿 admin token
2. `POST /api/category` 建分类（管理员）
3. `POST /api/product` 建商品（管理员，带 SKU）
4. `GET /api/product` 浏览商品（公开）
5. 注册普通用户 -> 登录拿 user token
6. `POST /api/order` 下单（自动扣库存）
7. `GET /api/product/:id` 看库存是否减少
8. 用普通用户 token 调 `POST /api/product` 应返回 403

---

## 附录

### A. 文件清单（共 16 个新文件 + 5 个修改）

| # | 文件 | 类型 | 步骤 |
|---|------|------|------|
| 1 | `domain/user/user.go` | 修改 | 1 |
| 2 | `persistence/user/model.go` | 修改 | 1 |
| 3 | `infrastructure/auth/token.go` | 修改 | 1 |
| 4 | `application/user/service.go` | 修改 | 1 |
| 5 | `middleware/jwt.go` | 修改 | 1 |
| 6 | `middleware/admin.go` | 新建 | 1 |
| 7 | `domain/product/errors.go` | 新建 | 2 |
| 8 | `domain/product/sku.go` | 新建 | 2 |
| 9 | `domain/product/product.go` | 新建 | 2 |
| 10 | `domain/product/repository.go` | 新建 | 2 |
| 11 | `domain/product/stock_deductor.go` | 新建 | 7 |
| 12 | `domain/category/category.go` | 新建 | 2 |
| 13 | `domain/category/errors.go` | 新建 | 2 |
| 14 | `domain/category/repository.go` | 新建 | 2 |
| 15 | `application/product/dto.go` | 新建 | 3 |
| 16 | `application/product/service.go` | 新建 | 3 |
| 17 | `persistence/product/model.go` | 新建 | 4 |
| 18 | `persistence/product/repository.go` | 新建 | 4 |
| 19 | `persistence/category/model.go` | 新建 | 5 |
| 20 | `persistence/category/repository.go` | 新建 | 5 |
| 21 | `application/category/dto.go` | 新建 | 5 |
| 22 | `application/category/service.go` | 新建 | 5 |
| 23 | `interfaces/http/handler/product.go` | 新建 | 6 |
| 24 | `interfaces/http/router/router.go` | 修改 | 6 |
| 25 | `application/order/service.go` | 修改 | 7 |
| 26 | `domain/order/item.go` | 修改 | 7 |
| 27 | `cmd/main.go` | 修改 | 8 |

### B. 防超卖原理
```
并发请求A、B 同时下单，库存=1：
  ❌ 先查再减：A查到1够、B查到1够 -> 各减1 -> 库存变-1，超卖
  ✅ 条件更新：UPDATE stock=stock-1 WHERE stock>=1
              A 命中1行(库存变0)，B 命中0行(库存0不满足>=1) -> B失败，不超卖
```
关键：把"判断+扣减"放进一条 SQL，靠数据库行锁保证原子性。

### C. 跨模块协作架构
```
order 应用层  --依赖接口-->  domain/product.StockDeductor  <--实现--  product 应用层
                                                                       ↓
                                                                product 仓储（真正扣库存）
```
order 不认识 productapp.Service 具体类型，只认识接口。main.go 把 productSvc 注入给 order。这就是依赖倒置在跨模块协作上的应用。

---

祝编码顺利！卡住时回来对照，或直接问。
