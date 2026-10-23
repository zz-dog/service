package merchant

import "time"

type Merchant struct {
	MerchantID uint
	UserID     uint
	Name       string // 店铺名,全局唯一(想清楚:为什么店铺名要全局唯一?——用户按店名搜索、订单展示"来自 xx 店",重名会串)
	Logo       string // logo
	Desc       string // 描述
	Status     MerchantStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewMerchant(userID uint, name, logo, desc string) *Merchant {
	return &Merchant{UserID: userID, Name: name, Logo: logo, Desc: desc, Status: MStatusSuspended}
}

// 营业中 → 已停业
func (m *Merchant) Suspend() (bool, error) {
	if m.Status == MStatusOperating {
		m.Status = MStatusSuspended
		return true, nil
	}
	return false, ErrInvalidStatusTransition
}

//已停业 → 营业中
func (m *Merchant) Resume() (bool, error) {
	if m.Status == MStatusSuspended {
		m.Status = MStatusBanned
		return true, nil
	}
	return false, ErrInvalidStatusTransition
}

// 营业中|已停业 → 已封禁
func (m *Merchant) Ban() (bool, error) {
	if m.Status == MStatusOperating || m.Status == MStatusSuspended {
		m.Status = MStatusBanned
		return true, nil
	}
	return false, ErrInvalidStatusTransition
}
