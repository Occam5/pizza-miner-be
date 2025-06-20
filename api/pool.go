package api

import (
	"singo/model"
	"singo/serializer"

	"github.com/gin-gonic/gin"
)

// GetCurrentPool 获取当前奖池状态
func GetCurrentPool(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		c.JSON(200, ErrorResponse(nil))
		return
	}

	// 获取用户当前参与的奖池
	var pool model.PrizePool

	// 查找用户最近参与的未完成的奖池
	result := model.DB.Joins("JOIN pool_participants ON pool_participants.pool_id = prize_pools.id").
		Where("pool_participants.user_id = ? AND prize_pools.status != ?", user.ID, model.PoolStatusCompleted).
		Order("prize_pools.created_at DESC").
		First(&pool)

	if result.Error != nil {
		c.JSON(200, serializer.Response{
			Code: 0,
			Data: gin.H{
				"pool":         nil,
				"participants": []interface{}{},
			},
		})
		return
	}

	// 获取奖池所有参与者
	participants, err := model.GetParticipantsByPoolID(pool.ID)
	if err != nil {
		c.JSON(200, serializer.DBErr("Failed to get participants", err))
		return
	}

	// 构建参与者信息
	var participantsData []gin.H
	for _, p := range participants {
		participantsData = append(participantsData, gin.H{
			"walletAddress":  p.WalletAddress,
			"serialNumber":   p.SerialNumber,
			"canSeeBigPrize": p.WalletAddress == pool.CurrentBigPrizeHolder,
		})
	}

	c.JSON(200, serializer.Response{
		Code: 0,
		Data: gin.H{
			"pool": gin.H{
				"id":             pool.ID,
				"status":         pool.Status,
				"currentPlayers": pool.CurrentPlayers,
				"prizeAmount":    pool.PrizeAmount,
			},
			"participants": participantsData,
		},
	})
}
