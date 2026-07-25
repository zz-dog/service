package order

// 订单状态
const (
	StatusPending   = 1 // 待支付
	StatusPaid      = 2 // 已支付
	StatusShipped   = 3 // 已发货
	StatusCompleted = 4 // 已完成
	StatusCancelled = 5 // 已取消
)

// statusName 状态中文名，用于日志与展示
var statusName = map[int]string{
	StatusPending:   "待支付",
	StatusPaid:      "已支付",
	StatusShipped:   "已发货",
	StatusCompleted: "已完成",
	StatusCancelled: "已取消",
}

// StatusName 返回状态的中文名；未知状态返回"未知状态"
func StatusName(s int) string {
	if name, ok := statusName[s]; ok {
		return name
	}
	return "未知状态"
}
