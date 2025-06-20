package server

import (
	"singo/api"
	"singo/middleware"

	"github.com/gin-gonic/gin"
)

// NewRouter 路由配置
func NewRouter() *gin.Engine {
	r := gin.Default()

	// 中间件
	r.Use(middleware.Cors())
	r.Use(middleware.CurrentUser())

	// 路由
	v1 := r.Group("/api/v1")
	{
		v1.POST("ping", api.Ping)

		// 用户登录
		v1.POST("auth/login", api.UserLogin)

		// 需要登录保护的
		auth := v1.Group("")
		auth.Use(middleware.AuthRequired())
		{
			// User Routing
			auth.GET("users/me", api.UserMe)
			auth.POST("users/claim-rewards", api.ClaimRewards)

			// Game Routing
			auth.POST("game/activate", api.GameActivate)
			auth.PUT("game/hunger", api.UpdateHunger)
			auth.POST("game/catch-big-prize", api.CatchBigPrize)

			// Pool Routing
			auth.GET("pools/current", api.GetCurrentPool)

			// WebSocket连接
			auth.GET("game/ws", api.WebSocketHandler)
		}
	}
	return r
}
