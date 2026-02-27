package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
  BaseModel 所有模型的基础结构
  功能：提供统一的主键、时间戳和软删除支持
*/
type BaseModel struct {
	ID        string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

/*
  BeforeCreate GORM 钩子：创建前自动生成 UUID
*/
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}
