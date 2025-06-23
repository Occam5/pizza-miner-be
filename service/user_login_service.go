package service

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"log"
	"singo/model"
	"singo/serializer"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/mr-tron/base58"
	"gorm.io/gorm"
)

// UserLoginService 管理用户登录的服务
type UserLoginService struct {
	WalletAddress string `form:"walletAddress" json:"walletAddress" binding:"required"`
	Message       string `form:"message" json:"message" binding:"required"`
	Timestamp     string `form:"timestamp" json:"timestamp" binding:"required"`
	Signature     string `form:"signature" json:"signature" binding:"required"`
}

const (
	jwtSecret = "your-secret-key" // TODO: Move to configuration
	jwtExpiry = 24 * time.Hour
)

// generateToken 生成JWT token
func generateToken(walletAddress string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"wallet_address": walletAddress,
		"exp":            time.Now().Add(jwtExpiry).Unix(),
	})
	return token.SignedString([]byte(jwtSecret))
}

// verifySignature 验证Solana钱包签名
func (service *UserLoginService) verifySignature() bool {
	// 构建签名消息，使用\r\n作为换行符以匹配前端
	message := service.Message
	if !strings.Contains(message, "Timestamp:") {
		message = fmt.Sprintf("%s\r\n\r\nTimestamp: %s", service.Message, service.Timestamp)
	}

	// 解码签名（Base58格式）
	sig, err := base58.Decode(service.Signature)
	if err != nil {
		log.Printf("Failed to decode signature for wallet %s: %v", service.WalletAddress, err)
		return false
	}

	// 解码钱包地址（Base58）
	pubKey, err := base58.Decode(service.WalletAddress)
	if err != nil {
		log.Printf("Failed to decode wallet address %s: %v", service.WalletAddress, err)
		return false
	}

	// 验证公钥长度
	if len(pubKey) != ed25519.PublicKeySize {
		return false
	}

	// 验证签名
	return ed25519.Verify(pubKey, []byte(message), sig)
}

// Login 用户登录函数
func (service *UserLoginService) Login(c *gin.Context) serializer.Response {
	// 将timestamp字符串转换为int64
	timestamp, err := strconv.ParseInt(service.Timestamp, 10, 64)
	if err != nil {
		return serializer.ParamErr("Invalid timestamp format", err)
	}

	// 验证签名时效性（5分钟内）
	if time.Now().Unix()-timestamp > 300 {
		return serializer.ParamErr("Signature expired", nil)
	}

	// 验证签名
	if !service.verifySignature() {
		return serializer.ParamErr(fmt.Sprintf("Invalid signature for wallet: %s", service.WalletAddress), nil)
	}

	// 获取或创建用户
	user, err := model.GetUserByWallet(service.WalletAddress)
	isNewUser := false

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 用户不存在，创建新用户
			user, err = model.CreateUser(service.WalletAddress)
			if err != nil {
				return serializer.DBErr("Failed to create user", err)
			}
			isNewUser = true
		} else {
			// 其他数据库错误
			return serializer.DBErr("Database error", err)
		}
	}

	// 获取用户的青蛙状态
	frog, err := model.GetFrogByUserID(user.ID)
	isActive := false
	if err == nil || model.IsRecordNotFoundError(err) {
		isActive = frog != nil && frog.IsActive
	}

	// 生成JWT token
	token, err := generateToken(service.WalletAddress)
	if err != nil {
		return serializer.Err(serializer.CodeEncryptError, "Failed to generate token", err)
	}

	return serializer.Response{
		Code: 0,
		Data: gin.H{
			"token":     token,
			"isNewUser": isNewUser,
			"user": gin.H{
				"walletAddress":    user.WalletAddress,
				"unclaimedRewards": user.UnclaimedRewards,
				"historyRewards":   user.HistoryRewards,
				"isActive":         isActive,
			},
		},
		Msg: "Login successful",
	}
}
