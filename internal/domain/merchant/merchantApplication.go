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
	ReviewedAt   *time.Time // 审核动作发生的时间
}

func NewMerchantApplication(userID uint, shopName, contactName, contactPhone, licenseURL, idCardURL string) *MerchantApplication {
	return &MerchantApplication{
		UserID:       userID,
		ShopName:     shopName,
		ContactName:  contactName,
		ContactPhone: contactPhone,
		LicenseURL:   licenseURL,
		IDCardURL:    idCardURL,
		Status:       AppStatusPending,
	}
}

func (app *MerchantApplication) Reject(reason string) (bool, error) {
	if app.Status != AppStatusPending {
		return false, ErrInvalidStatusTransition
	}
	app.Status = AppStatusRejected
	return true, nil
}
func (app *MerchantApplication) Approve() (bool, error) {
	if app.Status != AppStatusPending {
		return false, ErrInvalidStatusTransition
	}
	app.Status = AppStatusApproved
	return true, nil
}

func (app *MerchantApplication) Withdraw() (bool, error) {
	if app.Status != AppStatusPending {
		return false, ErrInvalidStatusTransition
	}
	return true, nil
}
