package model

import (
	"time"

	"gorm.io/gorm"
)

// Frog 青蛙模型
type Frog struct {
	gorm.Model
	UserID       uint      `gorm:"not null"`          // 关联用户ID
	User         User      `gorm:"foreignKey:UserID"` // 关联用户
	HungerLevel  int       `gorm:"default:100"`       // 饥饿值 0-100
	IsActive     bool      `gorm:"default:false"`     // 是否激活
	LastFeedTime time.Time `gorm:"type:timestamp"`    // 上次投喂时间
}

// CreateFrog 创建青蛙
func CreateFrog(userID uint) (Frog, error) {
	frog := Frog{
		UserID:       userID,
		HungerLevel:  100,
		IsActive:     true,
		LastFeedTime: time.Now(),
	}
	result := DB.Create(&frog)
	return frog, result.Error
}

// GetFrogByUserID 通过用户ID获取青蛙
func GetFrogByUserID(userID uint) (Frog, error) {
	var frog Frog
	result := DB.Where("user_id = ?", userID).First(&frog)
	return frog, result.Error
}

// UpdateHungerLevel 更新饥饿值
func (frog *Frog) UpdateHungerLevel(value int) error {
	frog.HungerLevel = value
	if frog.HungerLevel > 100 {
		frog.HungerLevel = 100
	} else if frog.HungerLevel < 0 {
		frog.HungerLevel = 0
	}
	frog.LastFeedTime = time.Now()
	return DB.Save(frog).Error
}
