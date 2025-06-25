package model

import (
	"time"

	"gorm.io/gorm"
)

// PoolParticipant 奖池参与者模型
type PoolParticipant struct {
	gorm.Model
	PoolID        uint      `gorm:"not null"`          // 关联奖池ID
	Pool          PrizePool `gorm:"foreignKey:PoolID"` // 关联奖池
	FrogID        uint      `gorm:"not null"`          // 关联青蛙ID
	Frog          Frog      `gorm:"foreignKey:FrogID"` // 关联青蛙
	WalletAddress string    `gorm:"type:varchar(44)"`  // 用户钱包地址
	SerialNumber  int       `gorm:"not null"`          // 在奖池中的序号 1-10
	JoinedAt      time.Time `gorm:"type:timestamp"`    // 加入时间
}

// GetParticipantsByPoolID 获取奖池的所有参与者
func GetParticipantsByPoolID(poolID uint) ([]PoolParticipant, error) {
	var participants []PoolParticipant
	result := DB.Where("pool_id = ?", poolID).Order("serial_number").Find(&participants)
	return participants, result.Error
}

// GetParticipantByFrogAndPool 获取青蛙在特定奖池中的参与信息
func GetParticipantByFrogAndPool(frogID, poolID uint) (PoolParticipant, error) {
	var participant PoolParticipant
	result := DB.Where("frog_id = ? AND pool_id = ?", frogID, poolID).First(&participant)
	return participant, result.Error
}
