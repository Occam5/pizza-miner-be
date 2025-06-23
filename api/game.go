package api

import (
	"net/http"
	"singo/service"

	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该限制
	},
	HandshakeTimeout: 10 * time.Second,
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
}

// GameActivate 激活青蛙（开始游戏）
func GameActivate(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		c.JSON(200, ErrorResponse(nil))
		return
	}

	var service service.GameActivateService
	if err := c.ShouldBind(&service); err == nil {
		res := service.Activate(c, user)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

// UpdateHunger 更新饥饿值
func UpdateHunger(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		c.JSON(200, ErrorResponse(nil))
		return
	}

	var service service.GameHungerService
	if err := c.ShouldBind(&service); err == nil {
		res := service.UpdateHunger(c, user)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

// CatchBigPrize 抓取大奖
func CatchBigPrize(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		c.JSON(200, ErrorResponse(nil))
		return
	}

	var service service.GameCatchPrizeService
	if err := c.ShouldBind(&service); err == nil {
		res := service.CatchBigPrize(c, user)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

// WebSocketHandler 处理WebSocket连接
func WebSocketHandler(c *gin.Context) {
	// 在升级之前不要写入任何响应头或状态码
	user := CurrentUser(c)
	if user == nil {
		log.Printf("WebSocket连接失败: 用户未认证")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	log.Printf("开始处理用户 %d 的WebSocket连接请求", user.ID)

	// 检查是否是WebSocket升级请求
	if c.GetHeader("Upgrade") != "websocket" {
		log.Printf("非WebSocket升级请求")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// 配置upgrader
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true // 在生产环境中应该根据实际情况配置
	}
	upgrader.ReadBufferSize = 4096
	upgrader.WriteBufferSize = 4096

	// 直接升级连接，不要设置任何响应头
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("用户 %d WebSocket升级失败: %v", user.ID, err)
		return
	}

	log.Printf("用户 %d WebSocket连接升级成功", user.ID)

	// 注册WebSocket连接
	if err := service.GetWebSocketManager().RegisterClient(user.ID, conn); err != nil {
		log.Printf("用户 %d 注册WebSocket客户端失败: %v", user.ID, err)
		conn.Close()
		return
	}

	log.Printf("用户 %d WebSocket客户端注册成功", user.ID)

	// 处理连接关闭
	defer func() {
		log.Printf("用户 %d WebSocket连接准备关闭", user.ID)
		service.GetWebSocketManager().UnregisterClient(user.ID)
		conn.Close()
	}()

	// 保持连接并处理消息
	for {
		// 设置较长的读取超时时间
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("用户 %d WebSocket读取消息错误: %v", user.ID, err)
			} else {
				log.Printf("用户 %d WebSocket连接正常关闭", user.ID)
			}
			break
		}

		// 处理消息
		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err == nil {
				log.Printf("用户 %d 收到消息类型: %v", user.ID, msg["type"])
				if msg["type"] == "ping" {
					// 设置写入超时并发送pong响应
					conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
					if err := service.GetWebSocketManager().HandlePing(conn); err != nil {
						log.Printf("用户 %d 处理ping消息失败: %v", user.ID, err)
						break
					}
					log.Printf("用户 %d 成功响应ping消息", user.ID)
				}
			} else {
				log.Printf("用户 %d 解析消息失败: %v", user.ID, err)
			}
		}
	}
}
