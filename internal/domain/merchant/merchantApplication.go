package merchant

import "time"

// MerchantApplication 入驻申请:一次入驻尝试的完整凭证。
// 只增不改(状态迁移除外),是审计对象——驳回重提永远新建,不在这里改材料。
type MerchantApplication struct {
	ID           uint
	UserID       uint        // 申请人,"我的申请"归属与越权判断用
	ShopName     string      // 申请时填的店铺名(凭证;当前店名看商户表)
	ContactName  string      // 申请人
	ContactPhone string      // 申请人
	LicenseURL   string      // 营业执照(OSS URL)
	IDCardURL    string      // 法人身份证(OSS URL)
	Status       ApplyStatus // 申请状态
	RejectReason string      // 驳回理由,必填
	MerchantID   *uint       // 通过后回填,指向产出的商户(审计关联;待审核时为 nil)
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ReviewedAt   *time.Time // 审核动作发生的时间
}

// NewMerchantApplication 充血构造函数:校验必填字段非空,状态直接是待审核。
// 为什么没有草稿态:教学版提交即审;真实系统要"存草稿"再加一个状态 + 定时清理,先不做。
func NewMerchantApplication(userID uint, shopName, contactName, contactPhone, licenseURL, idCardURL string) (*MerchantApplication, error) {
	if userID == 0 || shopName == "" || contactName == "" || contactPhone == "" || licenseURL == "" || idCardURL == "" {
		return nil, ErrInvalidApplication
	}
	return &MerchantApplication{
		UserID:       userID,
		ShopName:     shopName,
		ContactName:  contactName,
		ContactPhone: contactPhone,
		LicenseURL:   licenseURL,
		IDCardURL:    idCardURL,
		Status:       AppStatusPending,
	}, nil
}

// Approve 待审核 → 已通过。非法迁移返回 ErrInvalidStatusTransition。
// 只改内存状态,落库的条件更新在仓储——和 Payment.Pay() 同款双层结构:
// 领域管规则,仓储管并发。
func (app *MerchantApplication) Approve() error {
	if app.Status != AppStatusPending {
		return ErrInvalidStatusTransition
	}
	app.Status = AppStatusApproved
	return nil
}

// Reject 待审核 → 已驳回,理由必填(空理由的驳回是官僚主义,拒绝它)。
// 理由同时记进内存状态:驳回单是凭证,"为什么被拒"是凭证的一部分。
func (app *MerchantApplication) Reject(reason string) error {
	if reason == "" {
		return ErrInvalidRejectReason
	}
	if app.Status != AppStatusPending {
		return ErrInvalidStatusTransition
	}
	app.Status = AppStatusRejected
	app.RejectReason = reason
	return nil
}

// Withdraw 待审核 → 已撤回。
// 为什么"已通过"不能撤:商户已经诞生,退出要走停业/封禁,那是商户状态链的事;
// 在申请单上开"撤回已通过"的口子,商户和申请单就会各说各话。
func (app *MerchantApplication) Withdraw() error {
	if app.Status != AppStatusPending {
		return ErrInvalidStatusTransition
	}
	app.Status = AppStatusWithdraw
	return nil
}
