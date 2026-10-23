package merchant

import "errors"

var (
	ErrApplicationNotFound     = errors.New("申请单不存在")
	ErrAlreadyPending          = errors.New("已有待审核的申请,请耐心等待") // 防重复提交
	ErrAlreadyMerchant         = errors.New("您已拥有店铺,无需重复入驻")
	ErrShopNameTaken           = errors.New("店铺名已被占用")
	ErrInvalidApplication      = errors.New("申请信息不完整")
	ErrInvalidRejectReason     = errors.New("驳回理由不能为空")
	ErrInvalidStatusTransition = errors.New("当前状态不允许该操作")
	ErrMerchantNotFound        = errors.New("店铺不存在") // "别人的店"和"不存在的店"同错误,不泄露存在性
	ErrNotMerchantOwner        = errors.New("无权操作该店铺")
	ErrMerchantNotOperating    = errors.New("店铺未营业") // 阶段四:catalog 侧据此拒绝发布
	ErrForbidden               = errors.New("需要管理员权限")
)
