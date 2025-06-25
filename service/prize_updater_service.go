package service

import (
	"log"
	"math/rand"
	"singo/event"
	"singo/model"
	"sync"
	"time"
)

// PrizeUpdaterService 大奖位置更新服务
type PrizeUpdaterService struct {
	updaters    map[uint]chan struct{} // poolID -> stop channel
	updatersMux sync.RWMutex
}

var (
	prizeUpdater = &PrizeUpdaterService{
		updaters: make(map[uint]chan struct{}),
	}
)

// GetPrizeUpdaterService 获取大奖位置更新服务实例
func GetPrizeUpdaterService() *PrizeUpdaterService {
	return prizeUpdater
}

func init() {
	// 初始化随机数生成器
	rand.Seed(time.Now().UnixNano())

	// 订阅奖池激活事件
	event.Subscribe(event.PoolBecameActive, func(e event.PoolEvent) {
		prizeUpdater.StartUpdater(e.PoolID)
	})
}

// InitializeUpdaters 初始化所有活跃奖池的大奖更新器
func (s *PrizeUpdaterService) InitializeUpdaters() {
	var pools []model.PrizePool
	result := model.DB.Where("status = ?", model.PoolStatusActive).Find(&pools)
	if result.Error != nil {
		log.Printf("获取活跃奖池失败: %v", result.Error)
		return
	}

	for _, pool := range pools {
		s.StartUpdater(pool.ID)
	}
}

// StartUpdater 启动大奖位置更新器
func (s *PrizeUpdaterService) StartUpdater(poolID uint) {
	s.updatersMux.Lock()
	defer s.updatersMux.Unlock()

	// 如果已经存在更新器，先停止它
	if stopCh, exists := s.updaters[poolID]; exists {
		close(stopCh)
		delete(s.updaters, poolID)
	}

	stopCh := make(chan struct{})
	s.updaters[poolID] = stopCh

	go func() {
		// 用于追踪已经出现过大奖的青蛙
		appearedFrogs := make(map[uint]bool)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				// 获取奖池信息
				var pool model.PrizePool
				if err := model.DB.First(&pool, poolID).Error; err != nil {
					log.Printf("获取奖池 %d 信息失败: %v", poolID, err)
					return
				}

				// 如果奖池已完成，停止更新器
				if pool.Status == model.PoolStatusCompleted {
					s.StopUpdater(poolID)
					return
				}

				// 获取所有活跃的青蛙
				participants, err := model.GetParticipantsByPoolID(poolID)
				if err != nil {
					log.Printf("获取奖池 %d 参与者失败: %v", poolID, err)
					continue
				}

				var activeFrogs []uint
				for _, p := range participants {
					var frog model.Frog
					if err := model.DB.First(&frog, p.FrogID).Error; err != nil {
						continue
					}
					if frog.IsActive {
						activeFrogs = append(activeFrogs, frog.ID)
					}
				}

				if len(activeFrogs) == 0 {
					log.Printf("奖池 %d 没有活跃的青蛙", poolID)
					now := time.Now()
					pool.Status = model.PoolStatusCompleted
					pool.CompletedAt = &now
					if err := model.DB.Save(&pool).Error; err != nil {
						log.Printf("更新奖池状态失败: %v", err)
						return
					}

					// 获取WebSocket管理器并广播游戏结束
					wsManager := GetWebSocketManager()
					wsManager.BroadcastGameOver(pool.ID, "", 0)

					// 停止当前奖池的更新器
					s.StopUpdater(poolID)
					return
				}

				// 如果所有活跃青蛙都出现过，重置记录
				if len(appearedFrogs) >= len(activeFrogs) {
					appearedFrogs = make(map[uint]bool)
				}

				// 从未出现过的青蛙中随机选择一个
				var availableFrogs []uint
				for _, frogID := range activeFrogs {
					if !appearedFrogs[frogID] {
						availableFrogs = append(availableFrogs, frogID)
					}
				}

				if len(availableFrogs) == 0 {
					continue
				}

				// 随机选择一个青蛙
				selectedFrogID := availableFrogs[rand.Intn(len(availableFrogs))]
				appearedFrogs[selectedFrogID] = true

				// 获取选中青蛙的钱包地址
				var selectedParticipant model.PoolParticipant
				if err := model.DB.Where("pool_id = ? AND frog_id = ?", poolID, selectedFrogID).First(&selectedParticipant).Error; err != nil {
					log.Printf("获取选中青蛙 %d 的参与者信息失败: %v", selectedFrogID, err)
					continue
				}

				// 更新大奖位置
				if err := pool.UpdateBigPrizeHolder(selectedParticipant.WalletAddress); err != nil {
					log.Printf("更新奖池 %d 大奖位置失败: %v", poolID, err)
					continue
				}

				log.Printf("奖池 %d 大奖位置已更新到青蛙 %d", poolID, selectedFrogID)
				GetWebSocketManager().BroadcastBigPrizeLocation(poolID, selectedParticipant.WalletAddress)
			}
		}
	}()

	log.Printf("奖池 %d 的大奖位置更新器已启动", poolID)
}

// StopUpdater 停止大奖位置更新器
func (s *PrizeUpdaterService) StopUpdater(poolID uint) {
	s.updatersMux.Lock()
	defer s.updatersMux.Unlock()

	if stopCh, exists := s.updaters[poolID]; exists {
		close(stopCh)
		delete(s.updaters, poolID)
		log.Printf("奖池 %d 的大奖位置更新器已停止", poolID)
	}
}
