package service

import (
	"singo/model"
	"singo/serializer"
)

// ClaimRewardsService 提取奖励服务
type ClaimRewardsService struct{}

// ClaimRewardsResponse 提取奖励请求的响应
type ClaimRewardsResponse struct {
	RawTransaction string  `json:"rawTransaction"`
	Amount         float64 `json:"amount"`
}

// CreateTransaction 创建提取奖励的交易
func (service *ClaimRewardsService) CreateTransaction(user *model.User) serializer.Response {
	if user.UnclaimedRewards <= 0 {
		return serializer.Response{
			Code: 40001,
			Msg:  "No rewards to claim",
		}
	}

	// 构造Solana转账交易
	rawTransaction, err := CreateRewardTransferTransaction(
		user.WalletAddress,    // 用户钱包地址作为gas支付者
		user.UnclaimedRewards, // 转账金额
	)
	if err != nil {
		return serializer.Err(serializer.CodeDBError, "Failed to create transaction", err)
	}

	return serializer.Response{
		Code: 0,
		Data: ClaimRewardsResponse{
			RawTransaction: rawTransaction,
			Amount:         user.UnclaimedRewards,
		},
	}
}
