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

// RegisterClient 注册新的WebSocket客户端
func (m *WebSocketManager) RegisterClient(userID uint, conn *websocket.Conn) {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()

	// 如果已存在旧连接，关闭它
	if oldConn, exists := m.clients[userID]; exists {
		oldConn.Close()
	}

	m.clients[userID] = conn
}

// UnregisterClient 注销WebSocket客户端
func (m *WebSocketManager) UnregisterClient(userID uint) {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()

	if conn, exists := m.clients[userID]; exists {
		conn.Close()
		delete(m.clients, userID)
	}
}

// BroadcastHungerUpdate 广播饥饿值更新
func (m *WebSocketManager) BroadcastHungerUpdate(userID uint, frogID uint, newHungerLevel int) {
	m.clientsMux.RLock()
	defer m.clientsMux.RUnlock()

	if conn, exists := m.clients[userID]; exists {
		message := map[string]interface{}{
			"type":           "hunger-update",
			"frogId":         frogID,
			"newHungerLevel": newHungerLevel,
		}
		conn.WriteJSON(message)
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
	if result.Error != nil {
		log.Printf("Failed to get active frogs: %v", result.Error)
		return
	}

	for _, frog := range frogs {
		// 计算距离上次投喂的时间
		duration := time.Since(frog.LastFeedTime)
		decreaseAmount := int(duration.Seconds() / 3) // 每3秒减少1点

		if decreaseAmount > 0 {
			newHungerLevel := frog.HungerLevel - decreaseAmount
			if newHungerLevel < 0 {
				newHungerLevel = 0
			}

			// 更新饥饿值
			frog.HungerLevel = newHungerLevel
			frog.LastFeedTime = time.Now()
			if err := model.DB.Save(&frog).Error; err != nil {
				log.Printf("Failed to update frog hunger level: %v", err)
				continue
			}

			// 广播更新
			m.BroadcastHungerUpdate(frog.UserID, frog.ID, frog.HungerLevel)
		}
	}
}
