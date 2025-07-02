package service

import (
	"errors"
	"fmt"
	"log"
	"os"
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
	treasuryPublicKey := os.Getenv("TREASURY_PUBLIC_KEY")
	if treasuryPublicKey == "" {
		return serializer.ParamErr("Treasury public key not configured", nil)
	}

	// 验证转账交易
	verified, err := VerifyTransaction(service.TransactionHash, treasuryPublicKey)
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

	// 将青蛙添加到奖池
	err = pool.AddParticipant(frog.ID, user.WalletAddress)
	if err != nil {
		return serializer.DBErr("Failed to add participant", err)
	}

	// 获取青蛙在奖池中的序号
	participant, err := model.GetParticipantByFrogAndPool(frog.ID, pool.ID)
	if err != nil {
		return serializer.DBErr("Failed to get participant info", err)
	}

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

	// 计算新的饥饿值
	newHungerLevel := frog.HungerLevel + int(service.PizzaValue)
	log.Printf("用户 %d 的青蛙当前饥饿值: %d, 增加值: %d, 计算后值: %d",
		user.ID, frog.HungerLevel, int(service.PizzaValue), newHungerLevel)

	// 更新饥饿值（会在 UpdateHungerLevel 中自动限制在 0-100 范围内）
	err = frog.UpdateHungerLevel(newHungerLevel)
	if err != nil {
		return serializer.DBErr("Failed to update hunger level", err)
	}

	log.Printf("用户 %d 的青蛙饥饿值已更新为: %d", user.ID, frog.HungerLevel)

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
	// 获取用户的青蛙
	frog, err := model.GetFrogByUserID(user.ID)
	if err != nil {
		return serializer.DBErr("Failed to get frog", err)
	}
	if frog == nil {
		return serializer.ParamErr("User has no active frog", nil)
	}

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

	// 检查青蛙是否是参与者
	_, err = model.GetParticipantByFrogAndPool(frog.ID, pool.ID)
	if err != nil {
		return serializer.ParamErr("Frog is not in this pool", nil)
	}

	// 完成奖池并发放奖励
	err = pool.CompletePool(user.WalletAddress)
	if err != nil {
		return serializer.DBErr("Failed to complete pool", err)
	}

	// 更新获胜者的未领取奖励
	user.UnclaimedRewards += pool.PrizeAmount
	if err := model.DB.Save(user).Error; err != nil {
		return serializer.DBErr("Failed to update user rewards", err)
	}

	// 获取该奖池中的所有参与者
	participants, err := model.GetParticipantsByPoolID(pool.ID)
	if err != nil {
		return serializer.DBErr("Failed to get pool participants", err)
	}

	// 将所有参与者的青蛙饥饿值设置为0并停用
	for _, participant := range participants {
		var participantFrog model.Frog
		if err := model.DB.First(&participantFrog, participant.FrogID).Error; err != nil {
			continue // 跳过错误，继续处理其他青蛙
		}

		participantFrog.HungerLevel = 0
		participantFrog.IsActive = false

		if err := model.DB.Save(&participantFrog).Error; err != nil {
			continue // 跳过错误，继续处理其他青蛙
		}

		// 广播饥饿值更新
		wsManager.BroadcastHungerUpdate(participantFrog.UserID, participantFrog.ID, 0)
	}

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
