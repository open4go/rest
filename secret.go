package rest

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// ==================== 配置密钥 ====================
// 密钥不再内置默认值，必须通过以下任一方式提供（优先级从高到低），长度 16/24/32 字节：
//  1. 代码调用 SetDataEncryptKey（建议在 main 中加载配置后调用）
//  2. 环境变量 DATA_ENCRYPT_KEY
//  3. viper 配置 encryptKey.data
//
// 未配置时 EncryptData / DecryptData 返回 ErrDataEncryptKeyNotSet。

// EnvDataEncryptKey 读取数据加密密钥的环境变量名
const EnvDataEncryptKey = "DATA_ENCRYPT_KEY"

// ErrDataEncryptKeyNotSet 未配置数据加密密钥
var ErrDataEncryptKeyNotSet = errors.New("rest: data encrypt key is not configured (set DATA_ENCRYPT_KEY / encryptKey.data or call SetDataEncryptKey)")

// 旧版本内置的默认密钥摘要（已公开，不再允许使用）
var legacyDefaultKeySHA256 = "7a72285387f72d41edc7ab64ba212361f9fa081d53672366c95b5e7e84058c23"

var (
	dataKeyMu      sync.RWMutex
	dataEncryptKey []byte
)

func validateDataKey(b []byte) error {
	if len(b) != 16 && len(b) != 24 && len(b) != 32 {
		return fmt.Errorf("encrypt key length %d is not 16/24/32", len(b))
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) == legacyDefaultKeySHA256 {
		return errors.New("encrypt key is the former built-in default and must be rotated")
	}
	return nil
}

// SetDataEncryptKey sets the AES key after config is loaded.
func SetDataEncryptKey(key string) error {
	b := []byte(key)
	if err := validateDataKey(b); err != nil {
		return err
	}
	dataKeyMu.Lock()
	dataEncryptKey = b
	dataKeyMu.Unlock()
	return nil
}

// currentDataKey 返回已配置的密钥；首次使用时从环境变量 / viper 懒加载。
func currentDataKey() ([]byte, error) {
	dataKeyMu.RLock()
	k := dataEncryptKey
	dataKeyMu.RUnlock()
	if k != nil {
		return k, nil
	}
	key := strings.TrimSpace(os.Getenv(EnvDataEncryptKey))
	if key == "" {
		key = viper.GetString("encryptKey.data")
	}
	if key == "" {
		return nil, ErrDataEncryptKeyNotSet
	}
	if err := SetDataEncryptKey(key); err != nil {
		return nil, fmt.Errorf("rest: invalid data encrypt key: %w", err)
	}
	return []byte(key), nil
}

// EncryptData ==================== 加密手机号 ====================
func EncryptData(data string) (string, error) {
	key, err := currentDataKey()
	if err != nil {
		return "", err
	}
	return EncryptDataWithKey(data, key)
}

// EncryptDataWithKey 使用指定密钥加密（用于密钥轮换 / 数据迁移）
func EncryptDataWithKey(data string, key []byte) (string, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return "", fmt.Errorf("手机号不能为空")
	}

	block, err := aes.NewCipher(key)
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
	key, err := currentDataKey()
	if err != nil {
		return "", err
	}
	return DecryptDataWithKey(encrypted, key)
}

// DecryptDataWithKey 使用指定密钥解密（用于密钥轮换 / 数据迁移：旧密钥解密后用新密钥重新加密）
func DecryptDataWithKey(encrypted string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
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
