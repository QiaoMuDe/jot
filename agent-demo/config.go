package main

import (
	"flag"
	"fmt"
	"os"
)

// 默认配置值
const (
	defaultBaseURL = "https://api.deepseek.com/v1" // 默认 OpenAI 兼容端点：DeepSeek
	defaultModel   = "deepseek-chat"               // 默认模型
)

// Config 保存 Agent 运行所需的配置三要素。
type Config struct {
	BaseURL string // OpenAI 兼容 API 端点
	APIKey  string // API 认证密钥
	Model   string // 模型名称
}

// parseConfig 解析配置，优先级：命令行 flag > 环境变量 > 默认值。
func parseConfig() *Config {
	cfg := &Config{}

	// flag 的默认值取"环境变量 -> 默认值"，从而实现三级优先级
	flag.StringVar(&cfg.BaseURL, "base-url", envOr("AGENT_DEMO_BASE_URL", defaultBaseURL),
		"OpenAI 兼容 API 的 BaseURL（环境变量 AGENT_DEMO_BASE_URL）")
	flag.StringVar(&cfg.APIKey, "api-key", envOr("AGENT_DEMO_API_KEY", ""),
		"API Key（环境变量 AGENT_DEMO_API_KEY）")
	flag.StringVar(&cfg.Model, "model", envOr("AGENT_DEMO_MODEL", defaultModel),
		"模型名称（环境变量 AGENT_DEMO_MODEL）")
	flag.Parse()

	return cfg
}

// envOr 读取环境变量，未设置时返回默认值。
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// maskAPIKey 对 API Key 做脱敏处理，只显示前 4 位与后 4 位。
func maskAPIKey(key string) string {
	if key == "" {
		return "(未设置)"
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// printConfigSummary 打印脱敏后的配置摘要。
func printConfigSummary(cfg *Config) {
	fmt.Println("========== Agent 配置摘要 ==========")
	fmt.Printf("BaseURL : %s\n", cfg.BaseURL)
	fmt.Printf("Model   : %s\n", cfg.Model)
	fmt.Printf("APIKey  : %s\n", maskAPIKey(cfg.APIKey))
	fmt.Println("====================================")
}
