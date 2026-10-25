package merchant

type ApplyStatus int

type MerchantStatus int

// 申请单状态
const (
	AppStatusPending  ApplyStatus = iota + 1 // 待审核(提交即待审核,教学版没有草稿态)
	AppStatusApproved                        // 已通过
	AppStatusRejected                        // 已驳回(RejectReason 必填)
	AppStatusWithdraw                        // 已撤回
)

const (
	MStatusOperating MerchantStatus = iota + 1 // 营业中
	MStatusSuspended                           // 停业中
	MStatusBanned                              // 已封禁(平台处罚,不可自行恢复)
)
