package rest

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/spf13/viper"
	"strings"
)

// ==================== 配置密钥 ====================
// 建议放在配置文件或环境变量中，长度必须是16、24或32字节
var dataEncryptKey = []byte("HuangLiJi2026_SecretKey_32Bytes!") // 32字节

// init 函数：在包初始化时尝试从 viper 读取配置
func init() {
	// 优先从 viper 配置中读取 encryptKey.data
	if key := viper.GetString("encryptKey.data"); key != "" {
		dataEncryptKey = []byte(key)

		// 确保密钥长度符合 AES-256 要求（16/24/32字节）
		if len(dataEncryptKey) != 16 && len(dataEncryptKey) != 24 && len(dataEncryptKey) != 32 {
			fmt.Printf("警告: encryptKey.data 长度不符合要求 (%d)，已使用默认密钥\n", len(dataEncryptKey))
			dataEncryptKey = []byte("HuangLiJi2026_SecretKey_32Bytes!")
		}
	}
}

// SetDataEncryptKey sets the AES key after config is loaded.
// Call this from main after config.LoadConfig. init() runs too early to read viper.
func SetDataEncryptKey(key string) error {
	b := []byte(key)
	if len(b) != 16 && len(b) != 24 && len(b) != 32 {
		return fmt.Errorf("encrypt key length %d is not 16/24/32", len(b))
	}
	dataEncryptKey = b
	return nil
}

// EncryptData ==================== 加密手机号 ====================
func EncryptData(data string) (string, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return "", fmt.Errorf("手机号不能为空")
	}

	block, err := aes.NewCipher(dataEncryptKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptData ==================== 解密手机号 ====================
func DecryptData(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(dataEncryptKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("密文数据无效")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// DataMatch ==================== 手机号搜索对比（支持后4位搜索） ====================
func DataMatch(encryptedData, searchInput string) bool {
	searchInput = strings.TrimSpace(searchInput)
	if searchInput == "" {
		return false
	}

	// 解密后对比
	realData, err := DecryptData(encryptedData)
	if err != nil {
		return false
	}

	// 支持后4位搜索
	if len(searchInput) == 4 {
		return strings.HasSuffix(realData, searchInput)
	}

	// 支持完整手机号搜索
	return strings.Contains(realData, searchInput) || realData == searchInput
}

// GetDataSuffix ==================== 辅助函数（保持之前逻辑） ====================
func GetDataSuffix(data string) string {
	data = strings.TrimSpace(data)
	if len(data) < 4 {
		return data
	}
	return data[len(data)-4:]
}

func GetDataSec(data string) string {
	data = strings.TrimSpace(data)
	if len(data) != 11 {
		return data
	}
	return data[:3] + "****" + data[7:]
}

// GetDataHash 获取手机号哈希（用于快速索引和对比）
func GetDataHash(data string) string {
	data = strings.TrimSpace(data)
	if data == "" {
		return ""
	}

	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
