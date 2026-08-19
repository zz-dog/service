# 软字典（Soft Dictionary）实现方案

## 目标
把商品 SKU 的规格从单一字符串 `Spec string`（"红色 / L码"）改造为**结构化 + 软字典**：
- 维度名（颜色/尺码）受字典管控；
- 维度值既可取字典标准值（红色），也可填自定义文本（深红），二选一；
- 全局可按标准值筛选/聚合，自定义值放行不阻塞商家。

## 已确认决策
1. **独立 `spec` 模块**：参照 category，作为参考数据模块，商品通过 ID 引用。
2. **读取时查字典解析**：`product_sku_attrs` 只存 ID + 自定义文本；读商品时应用层批量查 spec 字典还原可读名（"颜色=红色"）。

## 数据模型

### spec 模块（新增表）
- `spec_attrs(attr_id PK, name UNIQUE, sort)` —— 规格维度名
- `spec_attr_values(value_id PK, attr_id INDEX, name, sort)` —— 维度下的标准可选值

### product 模块（改动表）
- `product_skus`：**删除 `spec` 列**（规格下沉到子表）
- `product_sku_attrs(sku_code, attr_id, value_id, value_text)` —— 软字典核心
  - PK = `(sku_code, attr_id)`（一个 SKU 每个维度只取一个值）
  - `value_id` 默认 0 表示自定义；`value_text` 自定义时填，标准值时空
  - 约束：`value_id > 0 OR value_text != ''`（应用层校验，GORM 不便加 CHECK）

## 各层改动

### 1. spec 领域层 `internal/domain/spec/`
- `SpecAttr` 聚合根：`AttrID, Name, Sort, Values []SpecAttrValue`
  - `NewSpecAttr(name, sort)`、`Rename`、`AddValue(name)`（防重）、`RemoveValue(id)`
- `SpecAttrValue` 实体（聚合内）：`ValueID, AttrID, Name, Sort`
- `errors.go`：`ErrEmptyAttrName / ErrAttrNotFound / ErrDuplicateAttrValue / ErrAttrValueNotFound`
- `repository.go` 端口：`FindByID / FindByIDs / FindAll / Save(级联值) / Delete`
  - `FindByIDs` 供商品读取时批量解析用

### 2. spec 基础设施层 `internal/infrastructure/persistence/spec/`
- `SpecAttrPO`（has-many `[]SpecAttrValuePO`，`foreignKey:AttrID`）
- `SpecAttrValuePO`
- Repository 实现：`Save` 用 `db.Save` 级联 upsert；`FindByIDs` 用 `WHERE attr_id IN (?)` + Preload Values
- `toPO` / `toDomain` 双向映射

### 3. spec 应用层 + 接口层
- `application/spec/service.go` + `dto.go`：`CreateAttr / AddValue / RemoveValue / ListAttrs`
- `handler/spec.go` + `router/spec.go`：
  - `POST /spec/attr` 建维度
  - `POST /spec/attr/:id/value` 加标准值
  - `DELETE /spec/attr/:id/value/:vid` 删标准值
  - `GET /spec/attrs` 列全部（商家选规格用）

### 4. product 领域层改动 `internal/domain/product/`
- `sku.go`：`Spec string` → `Specs []SpecItem`
  - 新增值对象 `SpecItem{ AttrID uint; ValueID uint; ValueText string }`
  - `SpecItem.validate()`：`AttrID>0`；`ValueID>0` 与 `ValueText!=""` 二选一
  - `SKU.validate()`：增加"SKU 内 AttrID 不重复"校验
- `product.go`：`NewProduct`/`ReplaceSKUs` 已调用 `s.validate()`，自动覆盖；无需大改
- **注**：attr/value 是否真存在于字典的跨模块校验放应用层（领域层只做结构校验）

### 5. product 基础设施层改动 `internal/infrastructure/persistence/product/`
- `model.go`：
  - `SKUPO`：删 `Spec`，加 `Specs []SKUAttrPO`（`foreignKey:SKUCode`）
  - 新增 `SKUAttrPO{ SKUCode; AttrID; ValueID; ValueText }`，表名 `product_sku_attrs`
- `repository.go`：
  - `toPO/toSKUPO`、`toProduct/toSKUs` 适配 Specs
  - **修复 `FindByID`**：`Where("id = ?")` → `Where("product_id = ?")`，并 `Preload("SKUs.Specs")`
  - `Save`：保持 `Create` 级联写入（GORM 自动级联两层 has-many）；更新场景留作后续

### 6. product 应用层改动 `internal/application/product/`
- `Service` 注入 `domain.SpecAttrRepository`（构造函数加参数）
- `dto.go`：`SKUInput.Spec` → `Specs []SpecItemInput{AttrID,ValueID,ValueText}`；`SKUDTO` 同理
- `service.go`：
  - `Create`：用 spec repo 校验每个 `AttrID` 存在；若 `ValueID>0` 则校验该值属于该 attr；通过后建商品
  - `toProductDTO`：收集所有 `AttrID`/`ValueID`，调 `specRepo.FindByIDs` 批量解析，拼成 `颜色=红色` / `颜色=深红(自定义)` 文本或结构返回

### 7. product 接口层改动
- `handler/product.go`：`createProductReq.SKUs[].Spec` → `Specs`；**补全 `Create`**（当前是空壳：绑定后既没调 service 也没 return）；新增 `Get`（按 id 查，演示解析）
- `router/product.go`：加 `GET /product/:id`

### 8. main.go 装配
- 新增 spec repo / svc，注入 router
- `productSvc` 改为 `NewService(productRepo, specRepo)`
- `AutoMigrate` 补齐：`SpecAttrPO, SpecAttrValuePO, ProductPO, SKUPO, SKUAttrPO`（当前 product 表完全没迁移）
- 注册 spec 路由

## 顺带修复的阻塞 Bug（不改则软字典跑不通）
- `FindByID` 列名错（`id` → `product_id`）且未 Preload SKUs
- `AutoMigrate` 漏掉 product 全部表
- `product.handler.Create` 是空壳

## 实施顺序（每步可单独编译，逐步讲解）
1. **spec 领域层**：聚合根 + 值对象 + 端口 + 错误
2. **spec 基础设施层**：PO + repository 实现
3. **spec 应用层 + 接口层 + main 装配**：字典 CRUD 先跑通（可 curl 建维度/加值）
4. **product 领域层**：SKU.Specs 改造 + SpecItem 校验
5. **product 基础设施层**：PO 改造 + repository 适配 + 修 FindByID
6. **product 应用层**：注入 spec repo，Create 校验 + DTO 解析
7. **product 接口层**：补全 Create + 新增 Get
8. **main.go**：product 装配 + AutoMigrate 补全
9. **端到端验证**：建字典 → 建带软字典的商品 → 查商品看解析结果

## 验证用例
- 字典建 `颜色[红色,蓝色]`、`尺码[S,M,L]`
- 建商品 SKU1：颜色=红色(标准)、尺码=L(标准)；SKU2：颜色=深红(自定义)、尺码=M(标准)
- 查商品 → SKU1 显示 `颜色=红色 / 尺码=L`，SKU2 显示 `颜色=深红 / 尺码=M`
- 异常用例：传不存在的 `AttrID`、`ValueID` 与 attr 不匹配 → 报错
