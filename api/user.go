package api

import (
	"singo/serializer"
	"singo/service"

	"github.com/gin-gonic/gin"
)

// UserLogin Solana钱包登录接口
func UserLogin(c *gin.Context) {
	var service service.UserLoginService
	if err := c.ShouldBind(&service); err == nil {
		res := service.Login(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

// UserMe 用户详情
func UserMe(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		c.JSON(200, serializer.Response{
			Code: 40001,
			Msg:  "User not found",
		})
		return
	}

	c.JSON(200, serializer.Response{
		Code: 0,
		Data: gin.H{
			"walletAddress":    user.WalletAddress,
			"unclaimedRewards": user.UnclaimedRewards,
			"historyRewards":   user.HistoryRewards,
		},
	})
}

// ClaimRewards 领取奖励
func ClaimRewards(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		c.JSON(200, serializer.Response{
			Code: 40001,
			Msg:  "User not found",
		})
		return
	}

	// TODO: 实现与Solana链的交互，转账奖励给用户
	amount := user.UnclaimedRewards

	// 更新用户奖励数据
	user.HistoryRewards += amount
	user.UnclaimedRewards = 0
	if err := user.UpdateRewards(0, user.HistoryRewards); err != nil {
		c.JSON(200, serializer.DBErr("Failed to update rewards", err))
		return
	}

	c.JSON(200, serializer.Response{
		Code: 0,
		Data: gin.H{
			"amount": amount,
			// TODO: 添加交易hash
			"transactionHash": "",
		},
		Msg: "Rewards claimed successfully",
	})
}
