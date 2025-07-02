package api

import (
	"singo/model"
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

	// 获取用户的青蛙状态
	frog, err := model.GetFrogByUserID(user.ID)
	if err != nil && !model.IsRecordNotFoundError(err) {
		c.JSON(200, serializer.DBErr("Failed to get frog status", err))
		return
	}

	// 判断是否有激活的青蛙
	isActive := frog != nil && frog.IsActive

	c.JSON(200, serializer.Response{
		Code: 0,
		Data: gin.H{
			"user": gin.H{
				"walletAddress":    user.WalletAddress,
				"unclaimedRewards": user.UnclaimedRewards,
				"historyRewards":   user.HistoryRewards,
				"isActive":         isActive,
			},
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

	var service service.ClaimRewardsService
	if err := c.ShouldBind(&service); err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}

	// 创建新的交易
	res := service.CreateTransaction(user)
	c.JSON(200, res)
}

// SubmitRewardTx 提交已签名的奖励交易
func SubmitRewardTx(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		c.JSON(200, serializer.Response{
			Code: 40001,
			Msg:  "User not found",
		})
		return
	}

	var service service.SubmitRewardTxService
	if err := c.ShouldBind(&service); err != nil {
		c.JSON(200, serializer.ParamErr("Invalid parameters", err))
		return
	}

	res := service.Submit(user)
	c.JSON(200, res)
}
