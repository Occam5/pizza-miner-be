package middleware

import (
	"strings"
	"singo/model"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gin-gonic/gin"
)

// CurrentUser 获取登录用户
func CurrentUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// 从 Bearer token 中提取token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		// 解析token
		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			return []byte("your-secret-key"), nil // TODO: 使用配置中的密钥
		})

		if err != nil || !token.Valid {
			c.Next()
			return
		}

		// 获取claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
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
