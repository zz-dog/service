package inventory

type Action string

const (
	ActionDeduct Action = "deduct"
	ActionRevert Action = "restock"
)
