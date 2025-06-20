package model

import (
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	gorm.Model
	WalletAddress    string  `gorm:"uniqueIndex;size:44"` // Solana wallet address
	UnclaimedRewards float64 `gorm:"default:0"`           // 未领取的奖励(SOL)
	HistoryRewards   float64 `gorm:"default:0"`           // 历史总收益(SOL)
}

// GetUser 用ID获取用户
func GetUser(ID interface{}) (User, error) {
	var user User
	result := DB.First(&user, ID)
	return user, result.Error
}

// GetUserByWallet 通过钱包地址获取用户
func GetUserByWallet(walletAddress string) (User, error) {
	var user User
	result := DB.Where("wallet_address = ?", walletAddress).First(&user)
	return user, result.Error
}

// CreateUser 创建用户
func CreateUser(walletAddress string) (User, error) {
	user := User{
		WalletAddress:    walletAddress,
		UnclaimedRewards: 0,
		HistoryRewards:   0,
	}
	result := DB.Create(&user)
	return user, result.Error
}

// UpdateRewards 更新用户奖励
func (user *User) UpdateRewards(unclaimed float64, history float64) error {
	user.UnclaimedRewards = unclaimed
	user.HistoryRewards = history
	return DB.Save(user).Error
}
