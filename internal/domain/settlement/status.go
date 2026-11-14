package settlement

type SettleStatus int

const (
	SettleStatusPending   SettleStatus = iota + 1 // 待结算
	SettleStatusExecuting                         // 结算中
	SettleStatusSuccess                           // 结算成功
	SettleStatusFailed                            // 结算失败
)

// StatusName 状态中文名,管理端列表展示用
func StatusName(s SettleStatus) string {
	switch s {
	case SettleStatusPending:
		return "待结算"
	case SettleStatusExecuting:
		return "结算中"
	case SettleStatusSuccess:
		return "已结算"
	case SettleStatusFailed:
		return "结算失败"
	default:
		return "未知状态"
	}
}
