package merchant

import "errors"

var (
	ErrApplicationNotFound  = errors.New("申请单不存在")
	ErrAlreadyPending       = errors.New("已有待审核的申请,请耐心等待") // 防重复提交申请单
	ErrShopNameTaken        = errors.New("店铺名已被占用")
	ErrInvalidApplication   = errors.New("申请信息不完整")
	ErrInvalidRejectReason  = errors.New("驳回理由不能为空")
	ErrInvalidMerchantInfo  = errors.New("店铺信息不完整")
	ErrInvalidStatusTransition = errors.New("当前状态不允许该操作")
	ErrMerchantNotFound       = errors.New("店铺不存在") // "非成员访问别人的店"和"店不存在"同错误,不泄露存在性
	ErrMerchantNotOperating   = errors.New("店铺未营业") // 阶段四:catalog 侧据此拒绝发布
	ErrForbidden              = errors.New("需要管理员权限")

	// ---- N:N 成员关系 ----
	ErrMemberNotFound      = errors.New("该用户不是店铺成员")  // 移除/改角色时目标不在店里(owner 调用,可明确告知)
	ErrAlreadyMember       = errors.New("该用户已是店铺成员")   // 邀请撞唯一索引前的友好报错
	ErrCannotModifyOwner   = errors.New("不能移除或调整店主")  // 每店必须永远有主;owner 转让是进阶
	ErrInvalidMemberRole   = errors.New("成员角色不合法")      // 邀请只发 admin/staff
	ErrMemberLimitReached  = errors.New("店铺成员数已达上限")   // MaxMembersPerShop
	ErrPermissionDenied    = errors.New("无权执行该操作")      // 是成员但角色不够(403,可明确告知)
)
