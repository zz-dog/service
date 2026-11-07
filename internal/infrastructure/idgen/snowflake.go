package idgen

import (
	"strconv"

	"github.com/bwmarrin/snowflake"

	"github.com/wsc-zz/service/internal/domain/payment"
	domainsettlement "github.com/wsc-zz/service/internal/domain/settlement"
)

// SnowflakeGenerator 雪花算法支付单号生成器,实现 domain/payment.PayNoGenerator。
// 布局:41 位毫秒时间戳(约 69 年)+ 10 位机器号 + 12 位序列号。
type SnowflakeGenerator struct {
	node *snowflake.Node
}

// NewSnowflakeGenerator nodeID 范围 0~1023。
// 多实例部署时各实例的 nodeID 必须不同,否则会发出重复单号;
// 以后 order 服务若也接入雪花,编号错开。
func NewSnowflakeGenerator(nodeID int64) (*SnowflakeGenerator, error) {
	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		return nil, err
	}
	return &SnowflakeGenerator{node: node}, nil
}

// NextPayNo 生成 19 位十进制数字字符串,趋势递增。
// node.Generate 并发安全,全局共用一个 node 即可。
func (g *SnowflakeGenerator) NextPayNo() string {
	return strconv.FormatInt(g.node.Generate().Int64(), 10)
}

// NextSettleNo 结算单号,与 NextPayNo 共用同一颗雪花:同一个 node 生成的号全局唯一,
// 两类单号靠各自表的唯一索引当幂等锚点,互不冲突。教程原话:复用同一颗雪花再开一个方法,
// 单号格式自定、全局唯一即可。
func (g *SnowflakeGenerator) NextSettleNo() string {
	return strconv.FormatInt(g.node.Generate().Int64(), 10)
}

// 编译期保证实现了领域端口
var (
	_ payment.PayNoGenerator             = (*SnowflakeGenerator)(nil)
	_ domainsettlement.SettleNoGenerator = (*SnowflakeGenerator)(nil)
)
