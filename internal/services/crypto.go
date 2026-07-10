package services

import (
	"encoding/base64"
	"strings"
)

const encodedPrefix = "(zk)"

// isEncoded 判断字符串是否已带有 (zk) 编码前缀
func isEncoded(s string) bool {
	return strings.HasPrefix(s, encodedPrefix)
}

// EncodeB64 对字符串进行 Base64 编码并加上 (zk) 前缀
// 空字符串返回空字符串
func EncodeB64(s string) string {
	if s == "" {
		return ""
	}
	return encodedPrefix + base64.StdEncoding.EncodeToString([]byte(s))
}

// DecodeB64 解码 (zk) 前缀的 Base64 字符串
// 无前缀则原样返回（兼容存量明文数据）
// 有空字符串返回空字符串
func DecodeB64(s string) string {
	if s == "" {
		return ""
	}
	if !isEncoded(s) {
		return s
	}
	raw := strings.TrimPrefix(s, encodedPrefix)
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		// 解码失败时原样返回
		return s
	}
	return string(decoded)
}
