package merchant

import "time"

// Merchant 商户(店铺):审核通过后诞生的经营实体。
// 与 user 是 N:N(成员关系落在 merchant_members 表),谁是店主看 role=owner 的成员记录;
// 与 user 的关系是"归属于",不是"是"——封店不删号,各活各的。
type Merchant struct {
	MerchantID uint
	Name       string // 店铺名,全局唯一(用户按店名搜索、订单展示"来自 xx 店",重名会串)
	Logo       string
	Desc       string
	Status     MerchantStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewMerchant 审核通过时构造,出生即营业中。
// 不再收 userID:归属(N:N)由成员表管理,商户自身不认识任何用户。
func NewMerchant(name, logo, desc string) (*Merchant, error) {
	if name == "" {
		return nil, ErrInvalidMerchantInfo
	}
	return &Merchant{Name: name, Logo: logo, Desc: desc, Status: MStatusOperating}, nil
}

// Suspend 营业中 → 已停业
func (m *Merchant) Suspend() error {
	if m.Status != MStatusOperating {
		return ErrInvalidStatusTransition
	}
	m.Status = MStatusSuspended
	return nil
}

// Resume 已停业 → 营业中
func (m *Merchant) Resume() error {
	if m.Status != MStatusSuspended {
		return ErrInvalidStatusTransition
	}
	m.Status = MStatusOperating
	return nil
}

// Ban 营业中|已停业 → 已封禁(平台处罚,不可自行恢复;已封禁再 Ban 是非法迁移)
func (m *Merchant) Ban() error {
	if m.Status != MStatusOperating && m.Status != MStatusSuspended {
		return ErrInvalidStatusTransition
	}
	m.Status = MStatusBanned
	return nil
}

// IsOperating 是否营业中——catalog 判断"能不能发布商品"的依据(阶段四 gRPC 用)
func (m *Merchant) IsOperating() bool {
	return m.Status == MStatusOperating
}

// UpdateInfo 更新店铺资料(名字非空校验;改名次数限制是阶段五的进阶)
func (m *Merchant) UpdateInfo(name, logo, desc string) error {
	if name == "" {
		return ErrInvalidMerchantInfo
	}
	m.Name, m.Logo, m.Desc = name, logo, desc
	return nil
}
