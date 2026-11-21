package merchantpo

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
)

// MigrateLegacyOwnership 一人一店 → N:N 的存量迁移。
// 老 merchants 表挂过 user_id 列(GORM AutoMigrate 只加列不删列,老列不会自己消失),
// 迁移三步:
//  ① merchants 表还没有 user_id 列 → 直接返回,幂等可重入;
//  ② 有 → 把每行 (id, user_id) 复制成一条 owner 成员记录(冲突跳过,不覆盖已有角色);
//  ③ 确认搬完 → DROP user_id 列,归属从此只有 merchant_members 一个事实源。
//
// 为什么用裸 SQL 而不是 Migrator API:列在结构体里已经不存在了,
// HasColumn/DropColumn 靠结构体映射的 API 在这里语义模糊,information_schema 最直白。
func MigrateLegacyOwnership(db *gorm.DB) error {
	exists, err := legacyColumnExists(db)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	var rows []struct {
		ID     uint
		UserID uint
	}
	if err := db.Table("merchants").Select("id, user_id").Scan(&rows).Error; err != nil {
		return err
	}
	if len(rows) > 0 {
		members := make([]MemberPO, 0, len(rows))
		for _, row := range rows {
			members = append(members, MemberPO{
				UserID:     row.UserID,
				MerchantID: row.ID,
				Role:       domainmerchant.RoleOwner,
			})
		}
		// OnConflict DoNothing:上次迁移中断重跑时,已搬过的行跳过,不覆盖已有角色
		if err := db.Model(&MemberPO{}).
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&members).Error; err != nil {
			return err
		}
	}
	return db.Exec("ALTER TABLE merchants DROP COLUMN user_id").Error
}

// legacyColumnExists 检查 merchants 表是否还挂着老的 user_id 列。
func legacyColumnExists(db *gorm.DB) (bool, error) {
	var count int64
	err := db.Raw(
		"SELECT COUNT(*) FROM information_schema.COLUMNS "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'merchants' AND COLUMN_NAME = 'user_id'",
	).Scan(&count).Error
	return count > 0, err
}
