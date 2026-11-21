package merchant

import "time"

// MemberRole 店铺成员角色。N:N 模型下,用户与商户的归属落在成员表,
// 角色决定"这个人对这家店能做什么"——权限是角色的属性,判断方法收在这里,
// 应用层只问"能不能",不自己展开 if-else(与 Payment.Pay 同款:规则住领域)。
type MemberRole int

const (
	RoleOwner MemberRole = 1 // 店主:全权(店铺资料/停业恢复/员工管理)
	RoleAdmin MemberRole = 2 // 管理员:经营全权(发布/修改/删除商品),动不了店和人事
	RoleStaff MemberRole = 3 // 员工:日常经营(上下架/补货/调价/媒体/客服 IM)
)

// MaxMembersPerShop 单店成员上限。唯一索引 (user_id, merchant_id) 防"重复邀请"是硬防线;
// 上限只是防滥用的友好检查,并发下略微超限可接受——计数和落库不是同一把锁,别装它是。
const MaxMembersPerShop = 20

// Valid 是否合法角色(1~3)。角色是"从审核通过/邀请入口合法诞生的值",
// 持久层读回来的脏值(0/负数/超范围)一律视为非成员,不能放权。
func (r MemberRole) Valid() bool { return r >= RoleOwner && r <= RoleStaff }

// IsMember 是否已是成员(任何合法角色)——RPC 授权返回 role=0 表示非成员。
func (r MemberRole) IsMember() bool { return r.Valid() }

// CanManageShop 能否管理店铺本身:改资料/停业/恢复。仅店主——
// 店是老板的营业执照,员工再能干也动不了主体资格。
func (r MemberRole) CanManageShop() bool { return r == RoleOwner }

// CanManageStaff 能否管理成员:邀请/移除/调整角色。仅店主——人事权在老板手里。
func (r MemberRole) CanManageStaff() bool { return r == RoleOwner }

// CanPublishProduct 能否发布/修改/删除商品:店主与管理员。
// 发布是合规责任(材料/描述),要担得起"审核被打回"的风险,店员不担。
func (r MemberRole) CanPublishProduct() bool { return r == RoleOwner || r == RoleAdmin }

// CanOperateProduct 能否日常经营:上下架/补货/调价/删规格/看商品/传媒体/客服 IM。
// 全部角色都可以——这是店员被雇来的目的。
func (r MemberRole) CanOperateProduct() bool { return r.Valid() }

// String 调试与日志友好输出。
func (r MemberRole) String() string {
	switch r {
	case RoleOwner:
		return "owner"
	case RoleAdmin:
		return "admin"
	case RoleStaff:
		return "staff"
	default:
		return "unknown"
	}
}

// Member 店铺成员:用户与商户 N:N 归属的载体。
// 一行 = "某用户以某角色归属某商户",(user_id, merchant_id) 唯一索引保证一人一店一条记录;
// 没有成员记录 = 与这家店毫无关系,一切操作拒绝。
// 商户表不再挂 user_id:owner 身份就是 role=owner 的那条成员记录,归属只有一个事实源。
type Member struct {
	ID         uint
	UserID     uint
	MerchantID uint
	Role       MemberRole
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewOwnerMember 开店时构造店主成员(出生即 owner)——
// 全系统唯一允许授予 owner 的入口,审核通过事务专用。
func NewOwnerMember(userID, merchantID uint) *Member {
	return &Member{UserID: userID, MerchantID: merchantID, Role: RoleOwner}
}

// NewMember 邀请成员:只发 admin/staff。owner 不走邀请(开店时授予),
// 这个约束在构造函数收口——调用方想传 RoleOwner 也传不进来,
// "每店有且只有一个 owner"因此不需要跨行事务去保证。
func NewMember(userID, merchantID uint, role MemberRole) (*Member, error) {
	if userID == 0 || merchantID == 0 {
		return nil, ErrInvalidMemberRole
	}
	if role != RoleAdmin && role != RoleStaff {
		return nil, ErrInvalidMemberRole
	}
	return &Member{UserID: userID, MerchantID: merchantID, Role: role}, nil
}
