package im

import (
	"context"
	"time"

	domainim "github.com/wsc-zz/service/internal/domain/im"
	"gorm.io/gorm"
)

type conversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) domainim.ConversationRepository {
	return &conversationRepository{db: db}
}

// FindByUserAndMerchant 按两端身份查会话;"没有会话"是正常业务态,
// gorm.ErrRecordNotFound 翻译成领域错误,不让持久层词汇泄漏进应用层。
func (r *conversationRepository) FindByUserAndMerchant(ctx context.Context, userID, merchantID uint64) (*domainim.Conversation, error) {
	var po ConversationPO
	err := r.db.WithContext(ctx).First(&po, "user_id = ? AND merchant_id = ?", userID, merchantID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainim.ErrConversationNotFound
		}
		return nil, err
	}
	return toConversation(po), nil
}

func (r *conversationRepository) Create(ctx context.Context, c *domainim.Conversation) error {
	po := toConversationPO(c)
	if err := r.db.WithContext(ctx).Create(&po).Error; err != nil {
		return err
	}
	c.ID = po.ID // 回填自增 ID,后续 UpdateOnNewMessage / 推送都要用
	return nil
}

// UpdateOnNewMessage 一条 UPDATE 同时完成"刷新快照 + 对侧未读自增":
// 行锁保证原子,不做 SELECT 出来加一再写回的读改写——
// 和支付回调条件更新同一哲学:并发正确性是数据库的责任,不是代码的责任。
func (r *conversationRepository) UpdateOnNewMessage(ctx context.Context, convID uint64, senderSide domainim.Side, content string, at time.Time) error {
	col := "merchant_unread" // 买家发的 → 卖家未读涨
	if senderSide == domainim.SideMerchant {
		col = "user_unread"
	}
	return r.db.WithContext(ctx).Model(&ConversationPO{}).
		Where("id = ?", convID).
		Updates(map[string]any{
			"last_msg_content": content,
			"last_msg_at":      at,
			col:                gorm.Expr(col + " + 1"), // SQL 层自增,不是内存算好再写回
		}).Error
}

func (r *conversationRepository) ListByUser(ctx context.Context, userID uint64, limit int) ([]*domainim.Conversation, error) {
	return r.listBySide(ctx, "user_id", userID, limit)
}

func (r *conversationRepository) ListByMerchant(ctx context.Context, merchantID uint64, limit int) ([]*domainim.Conversation, error) {
	return r.listBySide(ctx, "merchant_id", merchantID, limit)
}

// listBySide 会话列表:C/B 两端同构,只有过滤列不同——
// 排序走 (身份列, last_msg_at) 复合索引,最近聊过的排前面。
func (r *conversationRepository) listBySide(ctx context.Context, idCol string, id uint64, limit int) ([]*domainim.Conversation, error) {
	var pos []ConversationPO
	err := r.db.WithContext(ctx).
		Where(idCol+" = ?", id).
		Order("last_msg_at DESC").
		Limit(limit).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}
	convs := make([]*domainim.Conversation, 0, len(pos))
	for _, po := range pos {
		convs = append(convs, toConversation(po))
	}
	return convs, nil
}

func (r *conversationRepository) FindByID(ctx context.Context, convID uint64) (*domainim.Conversation, error) {
	var po ConversationPO
	err := r.db.WithContext(ctx).First(&po, "id = ?", convID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainim.ErrConversationNotFound
		}
		return nil, err
	}
	return toConversation(po), nil
}

// ClearUnread 我侧未读置零。身份列进 WHERE:越权校验和幂等一次解决——
// 不是本人时影响行数为 0;重复点已读只是重复置零,天然无害。
func (r *conversationRepository) ClearUnread(ctx context.Context, convID, selfID uint64, side domainim.Side) error {
	col, idCol := "user_unread", "user_id"
	if side == domainim.SideMerchant {
		col, idCol = "merchant_unread", "merchant_id"
	}
	return r.db.WithContext(ctx).Model(&ConversationPO{}).
		Where("id = ? AND "+idCol+" = ?", convID, selfID).
		Update(col, 0).Error
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) domainim.MessageRepository {
	return &messageRepository{db: db}
}

// Insert 追加一条消息,返回自增 ID(chat.ack 回执要用)。
// 消息表不可变:只有 INSERT 和 SELECT,没有任何 UPDATE 路径。
func (r *messageRepository) Insert(ctx context.Context, m *domainim.Message) (uint64, error) {
	po := toMessagePO(m)
	if err := r.db.WithContext(ctx).Create(&po).Error; err != nil {
		return 0, err
	}
	return po.ID, nil
}

// ListByConversation 游标分页:翻页凭据是"上一页最后一条的 ID",不是页码。
// OFFSET 的两个毛病它都没有:深翻页不用白扫前 N 页;翻页间隙来了新消息,
// 页也不漂移(ID 单调,窗口钉死)。倒序取 limit 条,取完反转成时间正序返回。
func (r *messageRepository) ListByConversation(ctx context.Context, convID uint64, beforeID uint64, limit int) ([]*domainim.Message, error) {
	q := r.db.WithContext(ctx).Where("conversation_id = ?", convID)
	if beforeID > 0 { // 0 = 第一页,不加条件
		q = q.Where("id < ?", beforeID)
	}
	var pos []MessagePO
	if err := q.Order("id DESC").Limit(limit).Find(&pos).Error; err != nil {
		return nil, err
	}
	msgs := make([]*domainim.Message, 0, len(pos))
	for _, po := range pos {
		msgs = append(msgs, toMessage(po))
	}
	// 反转:聊天窗口按时间正序渲染
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}
