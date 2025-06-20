package serializer

import "singo/model"

// User 用户序列化器
type User struct {
	ID               uint    `json:"id"`
	WalletAddress    string  `json:"wallet_address"`
	UnclaimedRewards float64 `json:"unclaimed_rewards"`
	HistoryRewards   float64 `json:"history_rewards"`
	CreatedAt        int64   `json:"created_at"`
}

// BuildUser 序列化用户
func BuildUser(user model.User) User {
	return User{
		ID:               user.ID,
		WalletAddress:    user.WalletAddress,
		UnclaimedRewards: user.UnclaimedRewards,
		HistoryRewards:   user.HistoryRewards,
		CreatedAt:        user.CreatedAt.Unix(),
	}
}

// BuildUserResponse 序列化用户响应
func BuildUserResponse(user model.User) Response {
	return Response{
		Data: BuildUser(user),
	}
}
