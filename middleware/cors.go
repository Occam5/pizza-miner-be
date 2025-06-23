package middleware

import (
	"regexp"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Cors 跨域配置
func Cors() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{
		"Origin",
		"Content-Length",
		"Content-Type",
		"Cookie",
		"Authorization",
		"Accept",
		"X-Requested-With",
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Headers",
		"Upgrade",
		"Connection",
		"Sec-WebSocket-Key",
		"Sec-WebSocket-Version",
		"Sec-WebSocket-Extensions",
		"Sec-WebSocket-Protocol",
	}
	if gin.Mode() == gin.ReleaseMode {
		// 生产环境需要配置跨域域名，否则403
		config.AllowOrigins = []string{"http://www.example.com"}
	} else {
		// 测试环境下模糊匹配本地开头的请求
		config.AllowOriginFunc = func(origin string) bool {
			if origin == "" {
				return true
			}
			if strings.HasPrefix(origin, "ws://") {
				origin = "http://" + strings.TrimPrefix(origin, "ws://")
			} else if strings.HasPrefix(origin, "wss://") {
				origin = "https://" + strings.TrimPrefix(origin, "wss://")
			}

			if regexp.MustCompile(`^http://127\.0\.0\.1:\d+$`).MatchString(origin) {
				return true
			}
			if regexp.MustCompile(`^http://localhost:\d+$`).MatchString(origin) {
				return true
			}
			return false
		}
	}
	config.AllowCredentials = true

	// 创建CORS处理器
	corsHandler := cors.New(config)

	return func(c *gin.Context) {
		// 检查是否是WebSocket升级请求
		if c.GetHeader("Upgrade") == "websocket" {
			c.Next()
			return
		}

		// 对于非WebSocket请求，使用标准CORS处理
		corsHandler(c)
	}
}
