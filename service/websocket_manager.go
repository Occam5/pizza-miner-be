package service

import (
	"log"
	"singo/model"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketManager 管理所有WebSocket连接
type WebSocketManager struct {
	clients    map[uint]*websocket.Conn // userID -> connection
	clientsMux sync.RWMutex
}

var (
	wsManager = &WebSocketManager{
		clients: make(map[uint]*websocket.Conn),
	}
)

// GetWebSocketManager 获取WebSocket管理器实例
func GetWebSocketManager() *WebSocketManager {
	return wsManager
}

// RegisterClient 注册新的WebSocket客户端并发送初始状态
func (m *WebSocketManager) RegisterClient(userID uint, conn *websocket.Conn) error {
	// 先获取旧连接（如果存在）
	m.clientsMux.RLock()
	oldConn, exists := m.clients[userID]
	m.clientsMux.RUnlock()

	// 如果存在旧连接，先关闭它
	if exists {
		log.Printf("用户 %d 存在旧连接，正在关闭", userID)
		oldConn.Close()
		// 等待一小段时间确保旧连接完全关闭
		time.Sleep(100 * time.Millisecond)
	}

	// 注册新连接
	m.clientsMux.Lock()
	m.clients[userID] = conn
	m.clientsMux.Unlock()
	log.Printf("用户 %d 的WebSocket连接已保存", userID)

	// 发送初始状态
	if err := m.sendInitialState(userID, conn); err != nil {
		log.Printf("用户 %d 发送初始状态失败: %v", userID, err)
		// 不要因为发送失败就中断连接
	} else {
		log.Printf("用户 %d 初始状态发送成功", userID)
	}

	return nil
}

// sendInitialState 发送初始状态给客户端
func (m *WebSocketManager) sendInitialState(userID uint, conn *websocket.Conn) error {
	log.Printf("开始获取用户 %d 的青蛙状态", userID)

	// 设置较长的写超时时间用于发送初始状态
	conn.SetWriteDeadline(time.Now().Add(60 * time.Second))
	defer conn.SetWriteDeadline(time.Now().Add(60 * time.Second)) // 恢复默认写超时

	// 获取用户当前的青蛙状态
	frog, err := model.GetFrogByUserID(userID)
	if err != nil {
		if model.IsRecordNotFoundError(err) {
			log.Printf("用户 %d 没有找到青蛙记录", userID)
			return nil
		}
		log.Printf("获取用户 %d 青蛙状态失败: %v", userID, err)
		return err
	}

	// 如果找到激活的青蛙，发送状态
	if frog != nil && frog.IsActive {
		log.Printf("用户 %d 的青蛙处于激活状态，准备发送状态更新", userID)

		// 发送饥饿值更新
		hungerMessage := map[string]interface{}{
			"type":           "hunger-update",
			"frogId":         frog.ID,
			"newHungerLevel": frog.HungerLevel,
		}

		// 重置写超时并发送消息
		conn.SetWriteDeadline(time.Now().Add(60 * time.Second))
		if err := conn.WriteJSON(hungerMessage); err != nil {
			log.Printf("用户 %d 发送饥饿值更新失败: %v", userID, err)
			return err
		}
		log.Printf("用户 %d 饥饿值更新发送成功", userID)

		// 获取并发送奖池信息
		if err := m.sendPoolInfo(userID, conn); err != nil {
			log.Printf("用户 %d 发送奖池信息失败: %v", userID, err)
			return err
		}
		log.Printf("用户 %d 奖池信息发送成功", userID)
	} else {
		log.Printf("用户 %d 的青蛙未激活或不存在，跳过发送状态", userID)
	}

	return nil
}

// sendPoolInfo 发送奖池信息
func (m *WebSocketManager) sendPoolInfo(userID uint, conn *websocket.Conn) error {
	log.Printf("开始获取用户 %d 的奖池信息", userID)

	// 设置写超时
	conn.SetWriteDeadline(time.Now().Add(60 * time.Second))

	pool, err := model.GetCurrentActivePool(userID)
	if err != nil {
		if model.IsRecordNotFoundError(err) {
			log.Printf("用户 %d 没有找到活跃奖池", userID)
			return nil
		}
		log.Printf("获取用户 %d 奖池信息失败: %v", userID, err)
		return err
	}

	if pool != nil {
		log.Printf("找到用户 %d 的活跃奖池，ID: %d", userID, pool.ID)
		participants, err := model.GetParticipantsByPoolID(pool.ID)
		if err != nil {
			log.Printf("获取奖池 %d 参与者信息失败: %v", pool.ID, err)
			return err
		}

		var participantsData []map[string]interface{}
		for _, p := range participants {
			participantsData = append(participantsData, map[string]interface{}{
				"walletAddress":  p.WalletAddress,
				"serialNumber":   p.SerialNumber,
				"canSeeBigPrize": p.WalletAddress == pool.CurrentBigPrizeHolder,
			})
		}

		// 重置写超时并发送奖池更新
		conn.SetWriteDeadline(time.Now().Add(60 * time.Second))
		poolMessage := map[string]interface{}{
			"type":         "pool-update",
			"poolId":       pool.ID,
			"participants": participantsData,
		}
		if err := conn.WriteJSON(poolMessage); err != nil {
			log.Printf("发送奖池 %d 更新消息失败: %v", pool.ID, err)
			return err
		}
		log.Printf("奖池 %d 更新消息发送成功", pool.ID)

		if pool.CurrentBigPrizeHolder != "" {
			// 重置写超时并发送大奖位置信息
			conn.SetWriteDeadline(time.Now().Add(60 * time.Second))
			bigPrizeMessage := map[string]interface{}{
				"type":          "big-prize-location",
				"poolId":        pool.ID,
				"holderAddress": pool.CurrentBigPrizeHolder,
			}
			if err := conn.WriteJSON(bigPrizeMessage); err != nil {
				log.Printf("发送大奖位置信息失败: %v", err)
				return err
			}
			log.Printf("大奖位置信息发送成功")
		}
	} else {
		log.Printf("用户 %d 没有活跃奖池", userID)
	}

	return nil
}

// HandlePing 处理客户端的ping消息
func (m *WebSocketManager) HandlePing(conn *websocket.Conn) error {
	err := conn.WriteJSON(map[string]string{"type": "pong"})
	if err != nil {
		log.Printf("发送pong响应失败: %v", err)
	} else {
		log.Printf("pong响应发送成功")
	}
	return err
}

// UnregisterClient 注销WebSocket客户端
func (m *WebSocketManager) UnregisterClient(userID uint) {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()

	if conn, exists := m.clients[userID]; exists {
		log.Printf("开始注销用户 %d 的WebSocket客户端", userID)
		delete(m.clients, userID) // 先从map中删除，避免其他goroutine继续使用
		conn.Close()              // 然后关闭连接
		log.Printf("用户 %d 的WebSocket客户端已注销", userID)
	}
}

// BroadcastHungerUpdate 广播饥饿值更新
func (m *WebSocketManager) BroadcastHungerUpdate(userID uint, frogID uint, newHungerLevel int) {
	m.clientsMux.RLock()
	conn, exists := m.clients[userID]
	m.clientsMux.RUnlock()

	if !exists {
		log.Printf("未找到用户 %d 的WebSocket连接", userID)
		return
	}

	log.Printf("准备广播用户 %d 的饥饿值更新: frogID=%d, newHungerLevel=%d", userID, frogID, newHungerLevel)

	// 设置写入超时
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	message := map[string]interface{}{
		"type":           "hunger-update",
		"frogId":         frogID,
		"newHungerLevel": newHungerLevel,
	}

	if err := conn.WriteJSON(message); err != nil {
		log.Printf("发送用户 %d 的饥饿值更新失败: %v", userID, err)
		// 如果是连接关闭错误，移除连接
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			m.UnregisterClient(userID)
		}
	} else {
		log.Printf("成功发送用户 %d 的饥饿值更新", userID)
	}
}

// BroadcastPoolUpdate 广播奖池更新
func (m *WebSocketManager) BroadcastPoolUpdate(poolID uint, participants []map[string]interface{}) {
	m.clientsMux.RLock()
	defer m.clientsMux.RUnlock()

	message := map[string]interface{}{
		"type":         "pool-update",
		"poolId":       poolID,
		"participants": participants,
	}

	// 向所有连接的客户端广播
	for _, conn := range m.clients {
		conn.WriteJSON(message)
	}
}

// BroadcastBigPrizeLocation 广播大奖位置更新
func (m *WebSocketManager) BroadcastBigPrizeLocation(poolID uint, holderAddress string) {
	m.clientsMux.RLock()
	defer m.clientsMux.RUnlock()

	message := map[string]interface{}{
		"type":          "big-prize-location",
		"poolId":        poolID,
		"holderAddress": holderAddress,
	}

	// 向所有连接的客户端广播
	for _, conn := range m.clients {
		conn.WriteJSON(message)
	}
}

// BroadcastGameOver 广播游戏结束
func (m *WebSocketManager) BroadcastGameOver(poolID uint, winnerAddress string, prizeAmount float64) {
	m.clientsMux.RLock()
	defer m.clientsMux.RUnlock()

	message := map[string]interface{}{
		"type":          "game-over",
		"poolId":        poolID,
		"winnerAddress": winnerAddress,
		"prizeAmount":   prizeAmount,
	}

	// 向所有连接的客户端广播
	for _, conn := range m.clients {
		conn.WriteJSON(message)
	}
}

// StartHungerUpdateWorker 启动饥饿值更新工作器
func (m *WebSocketManager) StartHungerUpdateWorker() {
	ticker := time.NewTicker(3 * time.Second)
	go func() {
		for range ticker.C {
			m.updateAllFrogsHunger()
		}
	}()
}

// updateAllFrogsHunger 更新所有激活的青蛙的饥饿值
func (m *WebSocketManager) updateAllFrogsHunger() {
	var frogs []model.Frog
	result := model.DB.Where("is_active = ?", true).Find(&frogs)
	if err := result.Error; err != nil {
		log.Printf("获取激活的青蛙失败: %v", err)
		return
	}

	for _, frog := range frogs {
		duration := time.Since(frog.LastFeedTime)
		decreaseAmount := int(duration.Seconds() / 3)

		if decreaseAmount > 0 {
			newHungerLevel := frog.HungerLevel - decreaseAmount
			if newHungerLevel < 0 {
				newHungerLevel = 0
			}

			frog.HungerLevel = newHungerLevel
			frog.LastFeedTime = time.Now()

			if newHungerLevel == 0 {
				frog.IsActive = false
				log.Printf("青蛙 %d 因饥饿值降至0而停用", frog.ID)
			}

			if err := model.DB.Save(&frog).Error; err != nil {
				log.Printf("更新青蛙饥饿值失败: %v", err)
				continue
			}

			m.BroadcastHungerUpdate(frog.UserID, frog.ID, frog.HungerLevel)
		}
	}
}
