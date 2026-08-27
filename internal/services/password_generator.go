package services

import (
	"crypto/rand"
	"math/big"
	"strings"

	"github.com/trustelem/zxcvbn"
)

// 字符池常量（与前端保持一致）
const (
	genUpper     = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	genLower     = "abcdefghijklmnopqrstuvwxyz"
	genDigits    = "0123456789"
	genSymbols   = "!@#$%^&*()_+-=[]{}|;:,.<>?"
	genAmbiguous = "lI1O0"
)

// PasswordGenOptions 密码生成选项
type PasswordGenOptions struct {
	Length           int  `json:"length"`           // 密码长度（6-64）
	Count            int  `json:"count"`            // 生成数量（1-20）
	Upper            bool `json:"upper"`            // 包含大写字母
	Lower            bool `json:"lower"`            // 包含小写字母
	Digits           bool `json:"digits"`           // 包含数字
	Symbols          bool `json:"symbols"`          // 包含符号
	ExcludeAmbiguous bool `json:"excludeAmbiguous"` // 排除易混淆字符 (lI1O0)
}

// GeneratedPassword 生成的密码及其强度
type GeneratedPassword struct {
	Password string `json:"password"`
	Score    int    `json:"score"` // 0-4 强度评级
}

// PasswordStrengthResult 密码强度检测结果
type PasswordStrengthResult struct {
	Score   int     `json:"score"`   // 0-4 强度评级
	Guesses float64 `json:"guesses"` // 预估猜测次数
}

// GeneratePasswords 批量生成密码并返回强度评分
func GeneratePasswords(opts PasswordGenOptions) []GeneratedPassword {
	length := opts.Length
	if length < 6 {
		length = 6
	} else if length > 64 {
		length = 64
	}
	count := opts.Count
	if count < 1 {
		count = 1
	} else if count > 20 {
		count = 20
	}

	// 构建字符池
	pool := buildPool(opts)
	if pool == "" {
		pool = genLower
	}

	results := make([]GeneratedPassword, count)
	for i := 0; i < count; i++ {
		pwd := generateRandomString(pool, length)
		score := CheckPasswordStrength(pwd)
		results[i] = GeneratedPassword{Password: pwd, Score: score}
	}
	return results
}

// CheckPasswordStrength 检测密码强度（0-4 评级）。
// 基于 zxcvbn 的猜测次数评分，叠加字符类型上限修正：
// 纯字符类型（仅数字/仅字母等）最高只能评 2（fair），2 种类型最高 3（good）。
func CheckPasswordStrength(password string) int {
	result := zxcvbn.PasswordStrength(password, nil)
	score := result.Score

	// 字符类型统计
	typeCount := 0
	if strings.ContainsAny(password, genLower) {
		typeCount++
	}
	if strings.ContainsAny(password, genUpper) {
		typeCount++
	}
	if strings.ContainsAny(password, genDigits) {
		typeCount++
	}
	if containsNonAlnum(password) {
		typeCount++
	}

	// 类型上限：1 类→最高 2，2 类→最高 3，3/4 类→不限
	typeCap := 4
	switch {
	case typeCount <= 1:
		typeCap = 2
	case typeCount == 2:
		typeCap = 3
	}

	if score > typeCap {
		score = typeCap
	}
	return score
}

// buildPool 根据选项构建字符池
func buildPool(opts PasswordGenOptions) string {
	var b strings.Builder
	if opts.Upper {
		b.WriteString(genUpper)
	}
	if opts.Lower {
		b.WriteString(genLower)
	}
	if opts.Digits {
		b.WriteString(genDigits)
	}
	if opts.Symbols {
		b.WriteString(genSymbols)
	}
	pool := b.String()
	if opts.ExcludeAmbiguous {
		pool = filterAmbiguous(pool)
	}
	return pool
}

// filterAmbiguous 过滤掉易混淆字符
func filterAmbiguous(pool string) string {
	var b strings.Builder
	for _, c := range pool {
		if !strings.ContainsRune(genAmbiguous, c) {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// generateRandomString 使用 crypto/rand 生成密码学安全的随机字符串
func generateRandomString(pool string, length int) string {
	poolLen := big.NewInt(int64(len(pool)))
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, poolLen)
		result[i] = pool[n.Int64()]
	}
	return string(result)
}

// containsNonAlnum 检查字符串中是否包含非字母数字字符
func containsNonAlnum(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return true
		}
	}
	return false
}
