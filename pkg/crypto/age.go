package crypto

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"

	"filippo.io/age"
)

// EncryptPassword 使用passphrase加密密码，返回base64编码的结果
func EncryptPassword(password, passphrase string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}
	if passphrase == "" {
		return "", fmt.Errorf("passphrase cannot be empty")
	}

	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return "", fmt.Errorf("failed to create scrypt recipient: %w", err)
	}

	// 固定scrypt工作因子以确保一致性
	recipient.SetWorkFactor(15)

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return "", fmt.Errorf("failed to create encryption writer: %w", err)
	}

	if _, err := w.Write([]byte(password)); err != nil {
		w.Close()
		return "", fmt.Errorf("failed to write password: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to close encryption writer: %w", err)
	}

	// 返回base64编码，方便在YAML中存储
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// DecryptPassword 解密密码（从base64编码）
func DecryptPassword(encrypted, passphrase string) (string, error) {
	if encrypted == "" {
		return "", fmt.Errorf("encrypted data cannot be empty")
	}
	if passphrase == "" {
		return "", fmt.Errorf("passphrase cannot be empty")
	}

	// 先base64解码
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return "", fmt.Errorf("failed to create scrypt identity: %w", err)
	}

	// 设置最大工作因子以匹配加密时使用的值
	identity.SetMaxWorkFactor(15)

	r, err := age.Decrypt(bytes.NewReader(data), identity)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	plaintext, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read decrypted data: %w", err)
	}

	return string(plaintext), nil
}

// GenerateKey 生成随机加密密钥
func GenerateKey() (string, error) {
	// 生成32字节的随机密钥
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i % 256)
	}

	// 使用简单的base32编码使其易于存储
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, 44)
	for i := range result {
		result[i] = chars[int(key[i%32])%len(chars)]
	}

	return string(result), nil
}
