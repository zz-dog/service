package imapp

import (
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"

	domainim "github.com/wsc-zz/service/internal/domain/im"
)

// Service im 应用服务:编排会话/消息仓储、事务与推送。
// REST 和 WS 两个入口共用这一套用例——连接层只做信封拆装,业务复杂性全在这里
type Service struct {
	convRepo domainim.ConversationRepository
	msgRepo  domainim.MessageRepository
	runner   TxRunner
	resolver MerchantResolver // B 端 handler 组装 Actor 时推导商户身份
	notifier Notifier
}

func NewService(convRepo domainim.ConversationRepository, msgRepo domainim.MessageRepository,
	runner TxRunner, resolver MerchantResolver, notifier Notifier) *Service {
	return &Service{convRepo: convRepo, msgRepo: msgRepo, runner: runner, resolver: resolver, notifier: notifier}
}

// ResolveMerchant 把 JWT 里的 userId 换成 merchantId(0 = 不是商户,调用方回 403)。
// 挂在 Service 上:REST 的 B 端 handler 和 WS 的握手 handler 共用同一个入口,
// "商户身份服务端推导"这条铁律只写一处
func (s *Service) ResolveMerchant(ctx context.Context, userID uint64) (uint64, error) {
	return s.resolver.MerchantIDByUserID(ctx, userID)
}

// SendText 发一条文本:建会话(懒创建)→ 插消息 → 刷会话 → 推对侧。
// 前三步在一个本地事务里(插消息和刷会话同生共死),推送在提交成功之后——
// 先推后存会出现"对方收到了、库里却没有"的灵异状态
func (s *Service) SendText(ctx context.Context, a Actor, peerID uint64, content string) (*domainim.Conversation, *domainim.Message, error) {
	// 归一化两端:谁发不重要,会话只认 (userID, merchantID) 这一对。
	// 卖家侧的商户身份来自 Actor(MerchantResolver 推导),peerID 永远是对端
	userID, merchantID, err := a.conversationPair(peerID)
	if err != nil {
		return nil, nil, err
	}
	senderID := a.UserID

	var conv *domainim.Conversation
	var msg *domainim.Message
	err = s.runner.RunIm(ctx, func(convRepo domainim.ConversationRepository, msgRepo domainim.MessageRepository) error {
		// ① 懒创建:先查(绝大多数请求命中),没有才建——
		// "买家点开商品页"就建会话,只会攒一堆空会话;会话由首条消息自然产生
		c, err := convRepo.FindByUserAndMerchant(ctx, userID, merchantID)
		switch {
		case err == nil:
			conv = c
		case errors.Is(err, domainim.ErrConversationNotFound):
			c = domainim.NewConversation(userID, merchantID)
			if err := convRepo.Create(ctx, c); err != nil {
				// ② 并发首条:两人(或同人两个标签页)同一瞬间都来建,
				// 唯一键让后来者撞索引。撞了 ≠ 失败:会话已存在,降级重查
				if !isDuplicateEntry(err) {
					return err
				}
				if c, err = convRepo.FindByUserAndMerchant(ctx, userID, merchantID); err != nil {
					return err
				}
				conv = c
			} else {
				conv = c
			}
		default:
			return err
		}
		// ③ 组装领域消息(长度校验在构造函数里),落库拿自增 ID
		m, err := domainim.NewTextMessage(conv.ID, a.Side, senderID, content)
		if err != nil {
			return err
		}
		id, err := msgRepo.Insert(ctx, m)
		if err != nil {
			return err
		}
		m.ID = id
		msg = m
		// ④ 刷会话:快照 + 对侧未读自增(SQL 原子)
		return convRepo.UpdateOnNewMessage(ctx, conv.ID, a.Side, content, m.CreatedAt)
	})
	if err != nil {
		return nil, nil, err
	}
	// ⑤ 事务已提交,才推对侧。Notifier 是端口,websocket 在外面包着;
	// 推送失败/对方离线无所谓——未读计数已落库,对方上线拉列表照样看到红点
	s.notifier.NotifyMessage(conv, msg)
	return conv, msg, nil
}

// ListConversations 会话列表:按 Actor 侧别走对应仓储,DTO 已经归一化成
// "我的视角"(peerId = 对面是谁,unread = 我几条没读)
func (s *Service) ListConversations(ctx context.Context, a Actor, limit int) ([]ConversationDTO, error) {
	limit = clampLimit(limit, 20, 50)
	var convs []*domainim.Conversation
	var err error
	if a.Side == domainim.SideMerchant {
		convs, err = s.convRepo.ListByMerchant(ctx, a.MerchantID, limit)
	} else {
		convs, err = s.convRepo.ListByUser(ctx, a.UserID, limit)
	}
	if err != nil {
		return nil, err
	}
	dtos := make([]ConversationDTO, 0, len(convs))
	for _, c := range convs {
		dtos = append(dtos, toConversationDTO(c, a))
	}
	return dtos, nil
}

// History 历史消息:先过成员校验再查——越权闸门在应用层,
// REST 和将来的 WS 共享同一条纪律。游标分页:beforeID=0 表示第一页
func (s *Service) History(ctx context.Context, a Actor, convID, beforeID uint64, limit int) ([]MessageDTO, error) {
	conv, err := s.convRepo.FindByID(ctx, convID)
	if err != nil {
		return nil, err
	}
	if !conv.BelongsTo(a.Side, a.SelfID()) {
		return nil, domainim.ErrNotParticipant
	}
	msgs, err := s.msgRepo.ListByConversation(ctx, convID, beforeID, clampLimit(limit, 20, 50))
	if err != nil {
		return nil, err
	}
	return toMessageDTOs(msgs), nil
}

// MarkRead 标记已读:成员校验同 History,然后持久层条件 UPDATE 置零。
// 重复调用只是重复置零,天然幂等
func (s *Service) MarkRead(ctx context.Context, a Actor, convID uint64) error {
	conv, err := s.convRepo.FindByID(ctx, convID)
	if err != nil {
		return err
	}
	if !conv.BelongsTo(a.Side, a.SelfID()) {
		return domainim.ErrNotParticipant
	}
	return s.convRepo.ClearUnread(ctx, convID, a.SelfID(), a.Side)
}

// conversationPair 归一化会话两端:我发还是对发不重要,会话只认 (userID, merchantID)。
// peerID 非法和 Actor 身份不全在这里拒绝
func (a Actor) conversationPair(peerID uint64) (userID, merchantID uint64, err error) {
	if peerID == 0 {
		return 0, 0, domainim.ErrInvalidSide
	}
	switch a.Side {
	case domainim.SideUser: // 买家发:我=UserID,对=peerID(商户ID)
		return a.UserID, peerID, nil
	case domainim.SideMerchant: // 卖家发:对=peerID(买家userId),我=a.MerchantID
		if a.MerchantID == 0 {
			return 0, 0, domainim.ErrInvalidSide
		}
		return peerID, a.MerchantID, nil
	default:
		return 0, 0, domainim.ErrInvalidSide
	}
}

// SelfID 我这一侧的身份 ID:买家 = userId,卖家 = merchantId。
// BelongsTo 成员校验、连接表 key 都用它
func (a Actor) SelfID() uint64 {
	if a.Side == domainim.SideMerchant {
		return a.MerchantID
	}
	return a.UserID
}

// clampLimit 分页参数边界:0 取默认,超出压到上限
func clampLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

// isDuplicateEntry 判 MySQL 1062(唯一键冲突)——懒创建并发的专属信号,
// 用 errors.As 拿驱动错误类型,不做字符串匹配
func isDuplicateEntry(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}
