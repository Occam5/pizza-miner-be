package service

import (
	"errors"
	"fmt"
	"singo/model"
	"singo/serializer"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GameActivateService 游戏激活服务
type GameActivateService struct {
	TransactionHash string `form:"transactionHash" json:"transactionHash" binding:"required"`
}

// GameHungerService 饥饿值更新服务
type GameHungerService struct {
	PizzaValue float64 `form:"pizzaValue" json:"pizzaValue" binding:"required"`
}

// GameCatchPrizeService 抓取大奖服务
type GameCatchPrizeService struct {
	PoolID uint `form:"poolId" json:"poolId" binding:"required"`
}

// Activate 激活青蛙
func (service *GameActivateService) Activate(c *gin.Context, user *model.User) serializer.Response {
	// 验证转账交易
	verified, err := VerifyTransaction(service.TransactionHash, "8bFainbiN7oKWUgat2iXqmcSJtcTs3yiUTtXs8CNW87e")
	if err != nil {
		return serializer.ParamErr(fmt.Sprintf("Failed to verify transaction: %v", err), err)
	}
	if !verified {
		return serializer.ParamErr("Invalid transaction", nil)
	}

	// 创建青蛙
	frog, err := model.CreateFrog(user.ID)
	if err != nil {
		return serializer.DBErr("Failed to create frog", err)
	}

	// 获取或创建奖池
	pool, err := model.GetAvailablePool()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 没有可用的奖池，创建新的
			pool, err = model.CreatePool()
			if err != nil {
				return serializer.DBErr("Failed to create pool", err)
			}
		} else {
			return serializer.DBErr("Failed to get pool", err)
		}
	}

	// 将用户添加到奖池
	err = pool.AddParticipant(user.ID, user.WalletAddress)
	if err != nil {
		return serializer.DBErr("Failed to add participant", err)
	}

	// 获取用户在奖池中的序号
	participant, err := model.GetParticipantByUserAndPool(user.ID, pool.ID)
	if err != nil {
		return serializer.DBErr("Failed to get participant info", err)
	}

	// 启动饥饿值更新工作器（如果还没有启动）
	GetWebSocketManager().StartHungerUpdateWorker()

	return serializer.Response{
		Code: 0,
		Data: gin.H{
			"frog": gin.H{
				"id":          frog.ID,
				"hungerLevel": frog.HungerLevel,
			},
			"poolInfo": gin.H{
				"id":             pool.ID,
				"currentPlayers": pool.CurrentPlayers,
				"serialNumber":   participant.SerialNumber,
			},
		},
	}
}

// UpdateHunger 更新饥饿值
func (service *GameHungerService) UpdateHunger(c *gin.Context, user *model.User) serializer.Response {
	// 获取用户的青蛙
	frog, err := model.GetFrogByUserID(user.ID)
	if err != nil {
		return serializer.DBErr("Failed to get frog", err)
	}

	// 更新饥饿值
	newHungerLevel := frog.HungerLevel + int(service.PizzaValue)
	err = frog.UpdateHungerLevel(newHungerLevel)
	if err != nil {
		return serializer.DBErr("Failed to update hunger level", err)
	}

	// 通过WebSocket广播更新
	wsManager.BroadcastHungerUpdate(user.ID, frog.ID, frog.HungerLevel)

	return serializer.Response{
		Code: 0,
		Data: gin.H{
			"newHungerLevel": frog.HungerLevel,
		},
	}
}

// CatchBigPrize 抓取大奖
func (service *GameCatchPrizeService) CatchBigPrize(c *gin.Context, user *model.User) serializer.Response {
	// 获取奖池
	var pool model.PrizePool
	result := model.DB.First(&pool, service.PoolID)
	if result.Error != nil {
		return serializer.DBErr("Failed to get pool", result.Error)
	}

	// 检查奖池状态
	if pool.Status != model.PoolStatusActive {
		return serializer.ParamErr("Pool is not active", nil)
	}

	// 检查用户是否是参与者
	_, err := model.GetParticipantByUserAndPool(user.ID, pool.ID)
	if err != nil {
		return serializer.ParamErr("User is not in this pool", nil)
	}

	// 完成奖池并发放奖励
	err = pool.CompletePool(user.WalletAddress)
	if err != nil {
		return serializer.DBErr("Failed to complete pool", err)
	}

	// TODO: 实现奖励发放逻辑

	// 广播游戏结束
	wsManager.BroadcastGameOver(pool.ID, user.WalletAddress, pool.PrizeAmount)

	return serializer.Response{
		Code: 0,
		Data: gin.H{
			"success": true,
			"reward":  pool.PrizeAmount,
		},
	}
}
