package model

import (
	"singo/event"
	"time"

	"gorm.io/gorm"
)

// PoolStatus 奖池状态
type PoolStatus string

const (
	PoolStatusCollecting PoolStatus = "collecting" // 收集中
	PoolStatusActive     PoolStatus = "active"     // 活跃中
	PoolStatusCompleted  PoolStatus = "completed"  // 已完成
)

// PrizePool 奖池模型
type PrizePool struct {
	gorm.Model
	Status                PoolStatus        `gorm:"type:varchar(20);not null"` // 奖池状态
	CurrentPlayers        int               `gorm:"default:0"`                 // 当前玩家数量
	PrizeAmount           float64           `gorm:"type:decimal(10,4)"`        // 奖池金额(SOL)
	BigPrizeWinner        string            `gorm:"type:varchar(44)"`          // 大奖获得者钱包地址
	CurrentBigPrizeHolder string            `gorm:"type:varchar(44)"`          // 当前可以看到大奖的用户地址
	CompletedAt           *time.Time        `gorm:"type:timestamp"`            // 完成时间
	Participants          []PoolParticipant `gorm:"foreignKey:PoolID"`         // 参与者
}

// CreatePool 创建奖池
func CreatePool() (PrizePool, error) {
	pool := PrizePool{
		Status:         PoolStatusCollecting,
		CurrentPlayers: 0,
		PrizeAmount:    0.1, // 初始奖池金额
	}
	result := DB.Create(&pool)
	return pool, result.Error
}

// GetAvailablePool 获取可用的奖池
func GetAvailablePool() (PrizePool, error) {
	var pool PrizePool
	result := DB.Where("status = ? AND current_players < 10", PoolStatusCollecting).First(&pool)
	return pool, result.Error
}

// AddParticipant 添加参与者
func (pool *PrizePool) AddParticipant(frogID uint, walletAddress string) error {
	if pool.CurrentPlayers >= 10 {
		return nil
	}

	participant := PoolParticipant{
		PoolID:        pool.ID,
		FrogID:        frogID,
		WalletAddress: walletAddress,
		SerialNumber:  pool.CurrentPlayers + 1,
		JoinedAt:      time.Now(),
	}

	tx := DB.Begin()
	if err := tx.Create(&participant).Error; err != nil {
		tx.Rollback()
		return err
	}

	pool.CurrentPlayers++
	if err := tx.Save(pool).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 获取所有参与者信息用于广播
	participants, err := GetParticipantsByPoolID(pool.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 准备参与者数据
	var participantsData []map[string]interface{}
	for _, p := range participants {
		// 获取青蛙的状态
		var frog Frog
		if err := DB.First(&frog, p.FrogID).Error; err != nil {
			continue
		}

		participantsData = append(participantsData, map[string]interface{}{
			"walletAddress":  p.WalletAddress,
			"serialNumber":   p.SerialNumber,
			"canSeeBigPrize": p.WalletAddress == pool.CurrentBigPrizeHolder,
			"isActive":       frog.IsActive,
		})
	}

	if pool.CurrentPlayers == 10 {
		pool.Status = PoolStatusActive
		if err := tx.Save(pool).Error; err != nil {
			tx.Rollback()
			return err
		}
		// 发布奖池激活事件
		event.Publish(event.PoolEvent{
			Type:   event.PoolBecameActive,
			PoolID: pool.ID,
		})
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// 发布奖池参与者变化事件
	event.Publish(event.PoolEvent{
		Type:         event.PoolParticipantsChanged,
		PoolID:       pool.ID,
		Participants: participantsData,
	})

	return nil
}

// CompletePool 完成奖池
func (pool *PrizePool) CompletePool(winnerAddress string) error {
	now := time.Now()
	pool.Status = PoolStatusCompleted
	pool.BigPrizeWinner = winnerAddress
	pool.CompletedAt = &now
	return DB.Save(pool).Error
}

// UpdateBigPrizeHolder 更新大奖持有者
func (pool *PrizePool) UpdateBigPrizeHolder(holderAddress string) error {
	pool.CurrentBigPrizeHolder = holderAddress
	return DB.Save(pool).Error
}
