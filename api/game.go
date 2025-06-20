package api

import (
	"net/http"
	"singo/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该限制
	},
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
	user := CurrentUser(c)
	if user == nil {
		c.JSON(200, ErrorResponse(nil))
		return
	}

	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}

	// 注册WebSocket连接
	service.GetWebSocketManager().RegisterClient(user.ID, conn)

	// 处理连接关闭
	defer func() {
		service.GetWebSocketManager().UnregisterClient(user.ID)
		conn.Close()
	}()

	// 保持连接并处理消息
	for {
		messageType, _, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// 心跳包
		if messageType == websocket.PingMessage {
			if err := conn.WriteMessage(websocket.PongMessage, nil); err != nil {
				break
			}
		}
	}
}
