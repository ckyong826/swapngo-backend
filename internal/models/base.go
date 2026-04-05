package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Base struct {
	id        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	createdAt time.Time      `json:"created_at"`
	updatedAt time.Time      `json:"updated_at"`
	deletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (base *Base) BeforeCreate(tx *gorm.DB) (err error) {
	if base.id == uuid.Nil {
		base.id = uuid.New()
	}
	return
}