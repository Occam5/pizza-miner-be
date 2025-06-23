package middleware

import (
	"singo/model"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// CurrentUser 获取登录用户
func CurrentUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// 检查是否是WebSocket请求
		if c.GetHeader("Upgrade") == "websocket" {
			// 从URL参数获取token
			token = c.Query("token")
		} else {
			// 从Authorization header获取token
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					token = parts[1]
				}
			}
		}

		if token == "" {
			c.Next()
			return
		}

		// 解析token
		parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte("your-secret-key"), nil // TODO: 使用配置中的密钥
		})

		if err != nil || !parsedToken.Valid {
			c.Next()
			return
		}

		// 获取claims
		if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
			if walletAddress, ok := claims["wallet_address"].(string); ok {
				user, err := model.GetUserByWallet(walletAddress)
				if err == nil {
					c.Set("user", &user)
				}
			}
		}

		c.Next()
	}
}

// AuthRequired 需要登录
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if user, _ := c.Get("user"); user == nil {
			// 对于WebSocket请求，返回401状态码
			if c.GetHeader("Upgrade") == "websocket" {
				c.AbortWithStatus(401)
				return
			}

			// 对于普通请求，返回JSON响应
			c.JSON(200, gin.H{
				"code": 401,
				"msg":  "Need login",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
