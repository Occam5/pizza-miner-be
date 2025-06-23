package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	SolanaMainnetRPCEndpoint = "https://api.mainnet-beta.solana.com"
	SolanaDevnetRPCEndpoint  = "https://devnet.helius-rpc.com/?api-key=0422440e-2b28-48a5-8683-fb4de54ee525"
	RequiredAmount           = 0.01       // 需要转账的SOL数量
	LAMPORTS_PER_SOL         = 1000000000 // 1 SOL = 10^9 lamports
	maxRetries               = 3          // 最大重试次数
	initialRetryDelay        = 1 * time.Second
)

// 当前使用的网络
var currentNetwork = "devnet" // 可以根据需要修改为 "mainnet"

// TransactionInfo Solana交易信息
type TransactionInfo struct {
	BlockTime   int64       `json:"blockTime"`
	Slot        int64       `json:"slot"`
	Meta        Meta        `json:"meta"`
	Transaction Transaction `json:"transaction"`
}

type Meta struct {
	Fee          uint64            `json:"fee"`
	PostBalances []uint64          `json:"postBalances"`
	PreBalances  []uint64          `json:"preBalances"`
	Status       TransactionStatus `json:"status"`
	Err          interface{}       `json:"err"`
}

type TransactionStatus struct {
	Ok interface{} `json:"Ok"`
}

type Transaction struct {
	Message Message `json:"message"`
}

type Message struct {
	AccountKeys []string `json:"accountKeys"`
}

// RPCResponse Solana RPC响应
type RPCResponse struct {
	JsonRPC string           `json:"jsonrpc"`
	Result  *TransactionInfo `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	ID int `json:"id"`
}

// GetSolanaRPCEndpoint 获取当前网络的RPC端点
func GetSolanaRPCEndpoint() string {
	if currentNetwork == "mainnet" {
		return SolanaMainnetRPCEndpoint
	}
	return SolanaDevnetRPCEndpoint
}

// sendRPCRequest 发送RPC请求（带重试机制）
func sendRPCRequest(requestBody []byte) (*http.Response, error) {
	var lastErr error
	retryDelay := initialRetryDelay

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			log.Printf("第%d次重试RPC请求...", i+1)
			time.Sleep(retryDelay)
			retryDelay *= 2 // 指数退避
		}

		client := &http.Client{
			Timeout: 10 * time.Second, // 设置超时时间
		}

		resp, err := client.Post(GetSolanaRPCEndpoint(), "application/json", bytes.NewBuffer(requestBody))
		if err == nil {
			return resp, nil
		}

		lastErr = err
		log.Printf("RPC请求失败 (尝试 %d/%d): %v", i+1, maxRetries, err)
	}

	return nil, fmt.Errorf("after %d retries: %v", maxRetries, lastErr)
}

// VerifyTransaction 验证转账交易
func VerifyTransaction(transactionHash string, expectedReceiverAddress string) (bool, error) {
	log.Printf("开始验证交易，Hash: %s, 接收地址: %s, 网络: %s",
		transactionHash, expectedReceiverAddress, currentNetwork)

	// 构建RPC请求
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getTransaction",
		"params": []interface{}{
			transactionHash,
			map[string]interface{}{
				"encoding":                       "json",
				"maxSupportedTransactionVersion": 0,
				"commitment":                     "confirmed",
			},
		},
	}

	// 打印请求内容
	requestJSON, _ := json.Marshal(requestBody)
	log.Printf("发送RPC请求到 %s: %s", GetSolanaRPCEndpoint(), string(requestJSON))

	// 发送请求（带重试）
	resp, err := sendRPCRequest(requestJSON)
	if err != nil {
		log.Printf("RPC请求最终失败: %v", err)
		return false, fmt.Errorf("failed to send RPC request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		return false, fmt.Errorf("failed to read response: %v", err)
	}

	// 打印原始响应
	log.Printf("收到RPC响应: %s", string(body))

	// 解析响应
	var response RPCResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("解析响应失败: %v", err)
		return false, fmt.Errorf("failed to parse response: %v", err)
	}

	// 检查RPC错误
	if response.Error != nil {
		log.Printf("RPC返回错误: Code=%d, Message=%s",
			response.Error.Code, response.Error.Message)
		return false, fmt.Errorf("RPC error: %s", response.Error.Message)
	}

	if response.Result == nil {
		log.Printf("未找到交易信息")
		return false, fmt.Errorf("transaction not found")
	}

	// 验证交易状态 - 检查Meta.Err是否为null（表示成功）
	if response.Result.Meta.Err != nil {
		log.Printf("交易执行失败: %v", response.Result.Meta.Err)
		return false, fmt.Errorf("transaction failed: %v", response.Result.Meta.Err)
	}

	// 验证接收地址
	receiverFound := false
	receiverIndex := -1
	log.Printf("交易包含的地址: %v", response.Result.Transaction.Message.AccountKeys)
	for i, address := range response.Result.Transaction.Message.AccountKeys {
		if address == expectedReceiverAddress {
			receiverFound = true
			receiverIndex = i
			log.Printf("找到匹配的接收地址: %s, 索引: %d", address, i)
			break
		}
	}
	if !receiverFound {
		log.Printf("未找到接收地址: %s", expectedReceiverAddress)
		return false, fmt.Errorf("receiver address not found in transaction")
	}

	// 验证转账金额
	if receiverIndex >= 0 && len(response.Result.Meta.PreBalances) > receiverIndex && len(response.Result.Meta.PostBalances) > receiverIndex {
		preBalance := float64(response.Result.Meta.PreBalances[receiverIndex]) / LAMPORTS_PER_SOL
		postBalance := float64(response.Result.Meta.PostBalances[receiverIndex]) / LAMPORTS_PER_SOL
		balanceChange := postBalance - preBalance

		log.Printf("接收地址余额变化: %f SOL (前: %f SOL, 后: %f SOL)",
			balanceChange, preBalance, postBalance)

		if balanceChange < RequiredAmount {
			return false, fmt.Errorf("insufficient transfer amount: got %f SOL, required %f SOL",
				balanceChange, RequiredAmount)
		}
	} else {
		log.Printf("无法验证转账金额：索引越界或余额信息不完整")
		return false, fmt.Errorf("cannot verify transfer amount: invalid balance information")
	}

	log.Printf("交易验证成功")
	return true, nil
}
