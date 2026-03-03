package model

import (
	"fmt"
	"time"
)

type URL struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	ShortCode   string    `gorm:"uniqueIndex;size:10;not null" json:"short_code"`
	LongURL     string    `gorm:"size:2048;not null" json:"long_url"`
	UserID      int64     `gorm:"index;default:0" json:"user_id,omitempty"`
	ExpireAt    time.Time `gorm:"index" json:"expire_at"`
	AccessCount int64     `gorm:"default:0" json:"access_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 返回分表名
func (URL) TableName(index ...int) string {
	if len(index) > 0 {
		return fmt.Sprintf("url_mapping_%02d", index[0])
	}
	return "url_mapping"
}

func (u *URL) IsExpired() bool {
	return u.ExpireAt.Before(time.Now())
}
