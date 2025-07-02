package service

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mr-tron/base58"
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

// TreasuryKeyConfig 金库密钥配置
type TreasuryKeyConfig struct {
	PublicKey  string // 金库公钥
	PrivateKey string // 金库私钥
}

// 从环境变量或配置文件加载金库密钥
func loadTreasuryConfig() (*TreasuryKeyConfig, error) {
	return &TreasuryKeyConfig{
		PublicKey:  os.Getenv("TREASURY_PUBLIC_KEY"),
		PrivateKey: os.Getenv("TREASURY_PRIVATE_KEY"),
	}, nil
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

// uint64ToLittleEndian 将uint64转换为小端字节序
func uint64ToLittleEndian(value uint64) []byte {
	bytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		bytes[i] = byte(value >> (i * 8))
	}
	return bytes
}

// CreateRewardTransferTransaction 创建奖励转账交易并用Treasury签名
func CreateRewardTransferTransaction(payerAddress string, amount float64) (string, error) {
	treasury, err := loadTreasuryConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load treasury config: %v", err)
	}

	// 获取最新的blockhash
	blockhashParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getLatestBlockhash",
		"params": []interface{}{
			map[string]interface{}{
				"commitment": "finalized",
			},
		},
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	jsonData, err := json.Marshal(blockhashParams)
	if err != nil {
		return "", fmt.Errorf("failed to marshal params: %v", err)
	}

	resp, err := client.Post(GetSolanaRPCEndpoint(), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send RPC request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var blockhashResult struct {
		Result struct {
			Value struct {
				Blockhash string `json:"blockhash"`
			} `json:"value"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &blockhashResult); err != nil {
		return "", fmt.Errorf("failed to decode blockhash response: %v", err)
	}

	if blockhashResult.Error != nil {
		return "", fmt.Errorf("RPC error: %s", blockhashResult.Error.Message)
	}

	// 构造转账指令数据
	amountInLamports := uint64(amount * LAMPORTS_PER_SOL)
	instructionData := []byte{2, 0, 0, 0} // Transfer instruction
	instructionData = append(instructionData, uint64ToLittleEndian(amountInLamports)...)

	// 构造二进制交易格式
	var txData bytes.Buffer

	// 1. 写入签名数量（预留空间）
	txData.WriteByte(2) // 需要2个签名：feePayer和treasury

	// 2. 写入签名（预留空间）
	for i := 0; i < 2; i++ {
		txData.Write(make([]byte, 64)) // 每个签名64字节
	}

	// 3. 写入消息头
	txData.WriteByte(2) // numRequiredSignatures
	txData.WriteByte(0) // numReadonlySignedAccounts
	txData.WriteByte(0) // numReadonlyUnsignedAccounts

	// 4. 写入账户数量
	txData.WriteByte(3) // 3个账户：feePayer, treasury, system program

	// 5. 写入账户公钥（按照正确的顺序）
	// feePayer公钥（必须是第一个）
	userPubkey, err := base58.Decode(payerAddress)
	if err != nil {
		return "", fmt.Errorf("failed to decode user public key: %v", err)
	}
	txData.Write(userPubkey)

	// Treasury公钥
	treasuryPubkey, err := base58.Decode(treasury.PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode treasury public key: %v", err)
	}
	txData.Write(treasuryPubkey)

	// System Program公钥
	programID, err := base58.Decode("11111111111111111111111111111111")
	if err != nil {
		return "", fmt.Errorf("failed to decode program ID: %v", err)
	}
	txData.Write(programID)

	// 6. 写入最近的blockhash
	recentBlockhash, err := base58.Decode(blockhashResult.Result.Value.Blockhash)
	if err != nil {
		return "", fmt.Errorf("failed to decode blockhash: %v", err)
	}
	txData.Write(recentBlockhash)

	// 7. 写入指令数量
	txData.WriteByte(1) // 1条指令

	// 8. 写入指令
	// Program ID index
	txData.WriteByte(2) // System Program的索引

	// 9. 写入指令账户数量
	txData.WriteByte(2) // 2个账户参与指令

	// 10. 写入指令账户索引（按照正确的顺序）
	txData.WriteByte(1) // treasury索引（发送方）
	txData.WriteByte(0) // feePayer索引（接收方）

	// 11. 写入指令数据长度
	txData.WriteByte(byte(len(instructionData)))

	// 12. 写入指令数据
	txData.Write(instructionData)

	// 先模拟交易
	simulateParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "simulateTransaction",
		"params": []interface{}{
			base64.StdEncoding.EncodeToString(txData.Bytes()),
			map[string]interface{}{
				"sigVerify": false,
				"encoding":  "base64",
			},
		},
	}

	jsonData, err = json.Marshal(simulateParams)
	if err != nil {
		return "", fmt.Errorf("failed to marshal simulate params: %v", err)
	}

	resp, err = client.Post(GetSolanaRPCEndpoint(), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send simulate request: %v", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read simulate response: %v", err)
	}

	var simulateResult struct {
		Result struct {
			Value struct {
				Err  interface{} `json:"err"`
				Logs []string    `json:"logs"`
			} `json:"value"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &simulateResult); err != nil {
		return "", fmt.Errorf("failed to decode simulate response: %v", err)
	}

	if simulateResult.Error != nil {
		return "", fmt.Errorf("RPC error during simulation: %s", simulateResult.Error.Message)
	}

	if simulateResult.Result.Value.Err != nil {
		return "", fmt.Errorf("transaction simulation failed: %v", simulateResult.Result.Value.Err)
	}

	// 打印模拟日志
	log.Printf("Transaction simulation logs: %v", simulateResult.Result.Value.Logs)

	// 返回未签名的交易，让前端处理签名
	return base64.StdEncoding.EncodeToString(txData.Bytes()), nil
}

// VerifyAndSubmitTransaction 验证并提交已完全签名的交易
func VerifyAndSubmitTransaction(signedTx string, expectedReceiver string, expectedAmount float64) (string, error) {
	// 加载 Treasury 配置
	treasury, err := loadTreasuryConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load treasury config: %v", err)
	}

	// 解码交易数据
	txData, err := base64.StdEncoding.DecodeString(signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to decode transaction: %v", err)
	}

	// 添加 Treasury 签名
	// Treasury 私钥
	treasuryPrivateKey, err := base58.Decode(treasury.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode treasury private key: %v", err)
	}

	// 计算签名偏移量
	// 交易格式：
	// - 签名数量 (1 byte)
	// - 签名数组 (每个签名64字节)
	// - 消息数据 (剩余部分)
	numSignatures := txData[0]
	messageStart := 1 + int(numSignatures)*64
	messageData := txData[messageStart:]

	// 计算并添加 Treasury 签名
	signature := ed25519.Sign(treasuryPrivateKey, messageData)

	// 创建新的交易数据缓冲区
	var newTxData bytes.Buffer

	// 写入签名数量（保持不变）
	newTxData.WriteByte(numSignatures)

	// 复制用户签名（第一个签名）
	newTxData.Write(txData[1:65])

	// 写入 Treasury 签名（第二个签名）
	newTxData.Write(signature)

	// 写入消息数据
	newTxData.Write(messageData)

	// 使用完整签名的交易进行验证
	verifyParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "simulateTransaction",
		"params": []interface{}{
			base64.StdEncoding.EncodeToString(newTxData.Bytes()),
			map[string]interface{}{
				"sigVerify":  true,
				"commitment": "finalized",
				"encoding":   "base64",
			},
		},
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	jsonData, err := json.Marshal(verifyParams)
	if err != nil {
		return "", fmt.Errorf("failed to marshal verify params: %v", err)
	}

	resp, err := client.Post(GetSolanaRPCEndpoint(), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send verify request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read verify response: %v", err)
	}

	var simResult struct {
		Result struct {
			Value struct {
				Err interface{} `json:"err"`
			} `json:"value"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &simResult); err != nil {
		return "", fmt.Errorf("failed to decode verify response: %v", err)
	}

	if simResult.Error != nil {
		return "", fmt.Errorf("RPC error: %s", simResult.Error.Message)
	}

	if simResult.Result.Value.Err != nil {
		return "", fmt.Errorf("transaction verification failed: %v", simResult.Result.Value.Err)
	}

	// 提交交易
	submitParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "sendTransaction",
		"params": []interface{}{
			base64.StdEncoding.EncodeToString(newTxData.Bytes()),
			map[string]interface{}{
				"encoding":            "base64",
				"skipPreflight":       false,
				"preflightCommitment": "finalized",
				"maxRetries":          3,
			},
		},
	}

	jsonData, err = json.Marshal(submitParams)
	if err != nil {
		return "", fmt.Errorf("failed to marshal submit params: %v", err)
	}

	resp, err = client.Post(GetSolanaRPCEndpoint(), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to submit transaction: %v", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read submit response: %v", err)
	}

	var result struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to decode submit response: %v", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("RPC error: %s", result.Error.Message)
	}

	return result.Result, nil
}
