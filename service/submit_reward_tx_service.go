package service

import (
	"singo/model"
	"singo/serializer"
)

// SubmitRewardTxService 提交奖励交易服务
type SubmitRewardTxService struct {
	SignedTransaction string  `form:"signedTransaction" json:"signedTransaction" binding:"required"`
	Amount            float64 `form:"amount" json:"amount" binding:"required"`
}

// Submit 提交已签名的奖励交易
func (service *SubmitRewardTxService) Submit(user *model.User) serializer.Response {
	// 验证金额是否匹配
	if service.Amount != user.UnclaimedRewards {
		return serializer.Response{
			Code: 40001,
			Msg:  "Invalid amount",
		}
	}

	// 验证并提交交易
	txHash, err := VerifyAndSubmitTransaction(service.SignedTransaction, user.WalletAddress, service.Amount)
	if err != nil {
		return serializer.Err(serializer.CodeDBError, "Failed to verify or submit transaction", err)
	}

	// 更新用户奖励数据
	user.HistoryRewards += service.Amount
	user.UnclaimedRewards = 0
	if err := user.UpdateRewards(0, user.HistoryRewards); err != nil {
		return serializer.DBErr("Failed to update rewards", err)
	}

	return serializer.Response{
		Code: 0,
		Data: map[string]interface{}{
			"success":         true,
			"amount":          service.Amount,
			"transactionHash": txHash,
		},
		Msg: "Transaction submitted successfully",
	}
}
