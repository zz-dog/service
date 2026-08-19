package specpo

import "time"

type SpecPO struct {
	SpecID    uint          `gorm:"primaryKey;autoIncrement;comment:规格ID"`
	Name      string        `gorm:"size:128;uniqueIndex;comment:规格名称"`
	Sort      int           `gorm:"comment:排序"`
	Values    []SpecValuePO `gorm:"foreignKey:SpecID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;comment:规格值"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SpecValuePO struct {
	ValueID uint   `gorm:"primaryKey;autoIncrement;comment:规格值ID"`
	SpecID  uint   `gorm:"index;comment:规格ID"`
	Name    string `gorm:"size:128;comment:规格值名称"`
	Sort    int    `gorm:"comment:排序"`
}

func (SpecPO) TableName() string {
	return "spec"
}

func (SpecValuePO) TableName() string {
	return "spec_value"
}
