package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// legacyDecryptKey 旧版本默认密钥，仅用于把存量密文回写为明文。
// 内网环境不引入新加密，但需要保留解密能力做一次性迁移。
const legacyDecryptKey = "dm-key-2026-secure"

// decryptPassword 兼容旧版本加密格式的密码解密。
// 新数据以明文存储（内网环境），老数据若带 enc:v1: 前缀则用 legacy key 解密。
// 解密失败或明文输入都原样返回，保证存量数据不丢。
func decryptPassword(s string) string {
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, "enc:v1:") {
		// 已经是明文
		return s
	}
	ct := strings.TrimPrefix(s, "enc:v1:")
	raw, err := base64.StdEncoding.DecodeString(ct)
	if err != nil {
		return s
	}
	h := sha256.Sum256([]byte(legacyDecryptKey))
	block, err := aes.NewCipher(h[:])
	if err != nil {
		return s
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return s
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return s
	}
	nonce, body := raw[:ns], raw[ns:]
	pt, err := gcm.Open(nil, nonce, body, nil)
	if err != nil {
		return s
	}
	return string(pt)
}
