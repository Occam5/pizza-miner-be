package model

import (
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
)

const (
	MaxHungerLevel = 100 // 最大饥饿值
)

// Frog 青蛙模型
type Frog struct {
	gorm.Model
	UserID       uint      `gorm:"not null"`          // 关联用户ID
	User         User      `gorm:"foreignKey:UserID"` // 关联用户
	HungerLevel  int       `gorm:"default:100"`       // 饥饿值 0-100
	IsActive     bool      `gorm:"default:true"`      // 是否激活
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
func GetFrogByUserID(userID uint) (*Frog, error) {
	var frog Frog
	result := DB.Where("user_id = ? AND is_active = ?", userID, true).First(&frog)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &frog, nil
}

// UpdateHungerLevel 更新饥饿值
func (frog *Frog) UpdateHungerLevel(newLevel int) error {
	if newLevel < 0 {
		newLevel = 0
	} else if newLevel > MaxHungerLevel {
		newLevel = MaxHungerLevel
	}

	frog.HungerLevel = newLevel
	frog.LastFeedTime = time.Now()

	// 当饥饿值为0时，设置为非激活状态
	if newLevel == 0 {
		frog.IsActive = false
		log.Printf("Frog %d has been deactivated due to hunger level reaching 0", frog.ID)
	}

	return DB.Save(frog).Error
}

// IsRecordNotFoundError 检查是否是记录未找到错误
func IsRecordNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// GetCurrentActivePool 获取用户当前活跃的奖池
func GetCurrentActivePool(userID uint) (*PrizePool, error) {
	var pool PrizePool
	result := DB.Joins("JOIN pool_participants ON pool_participants.pool_id = prize_pools.id").
		Where("pool_participants.user_id = ? AND prize_pools.status != ?", userID, PoolStatusCompleted).
		Order("prize_pools.created_at DESC").
		First(&pool)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &pool, nil
}
