// Package config 负责 vec-poc 的命令行配置解析：
// 优先级为 命令行 flag > 环境变量 > 默认值。
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Config 保存 vec-poc 的全部运行配置。
type Config struct {
	DBPath     string // SQLite 数据库文件路径
	OllamaURL  string // Ollama 服务地址
	EmbedModel string // Ollama embedding 模型名
	LLMBaseURL string // OpenAI 兼容 LLM 服务地址（必填）
	LLMAPIKey  string // OpenAI 兼容 LLM API Key（必填）
	LLMModel   string // OpenAI 兼容 LLM 模型名（必填）
	TopK       int    // 向量召回 topK
	ForceBrute bool   // 强制使用纯 Go 暴力检索实现
}

// defaultValues 集中管理各字段的默认值。
var defaultValues = Config{
	DBPath:     "./vec-test.db",
	OllamaURL:  "http://localhost:11434",
	EmbedModel: "bge-m3",
	TopK:       5,
}

// Parse 解析命令行 flag 与环境变量，返回最终的配置。
// flag 显式设置时优先；未设置时回落到环境变量；都缺失时使用默认值。
func Parse() *Config {
	// 声明 flag 变量（初始为零值，用于区分"是否显式设置"）
	var (
		dbPath     string
		ollamaURL  string
		embedModel string
		llmBaseURL string
		llmAPIKey  string
		llmModel   string
		topK       int
		forceBrute bool
	)

	flag.StringVar(&dbPath, "db", "", "SQLite 数据库文件路径（默认 ./vec-test.db）")
	flag.StringVar(&ollamaURL, "ollama-url", "", "Ollama 服务地址（默认 http://localhost:11434）")
	flag.StringVar(&embedModel, "embed-model", "", "Ollama embedding 模型名（默认 bge-m3）")
	flag.StringVar(&llmBaseURL, "llm-base-url", "", "OpenAI 兼容 LLM 服务地址（必填，ask 命令）")
	flag.StringVar(&llmAPIKey, "llm-api-key", "", "OpenAI 兼容 LLM API Key（必填，ask 命令）")
	flag.StringVar(&llmModel, "llm-model", "", "OpenAI 兼容 LLM 模型名（必填，ask 命令）")
	flag.IntVar(&topK, "topk", 0, "向量召回 topK（默认 5）")
	flag.BoolVar(&forceBrute, "force-brute", false, "强制使用纯 Go 暴力检索实现")
	flag.Parse()

	// 记录哪些 flag 被显式设置过
	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	cfg := &Config{
		DBPath:     firstNonEmpty(strIf(setFlags["db"], dbPath), os.Getenv("VEC_DB"), defaultValues.DBPath),
		OllamaURL:  firstNonEmpty(strIf(setFlags["ollama-url"], ollamaURL), os.Getenv("VEC_OLLAMA_URL"), defaultValues.OllamaURL),
		EmbedModel: firstNonEmpty(strIf(setFlags["embed-model"], embedModel), os.Getenv("VEC_EMBED_MODEL"), defaultValues.EmbedModel),
		LLMBaseURL: firstNonEmpty(strIf(setFlags["llm-base-url"], llmBaseURL), os.Getenv("VEC_LLM_BASE_URL"), ""),
		LLMAPIKey:  firstNonEmpty(strIf(setFlags["llm-api-key"], llmAPIKey), os.Getenv("VEC_LLM_API_KEY"), ""),
		LLMModel:   firstNonEmpty(strIf(setFlags["llm-model"], llmModel), os.Getenv("VEC_LLM_MODEL"), ""),
		TopK:       defaultValues.TopK,
		ForceBrute: forceBrute,
	}

	// TopK：flag 显式设置 > 环境变量 > 默认 5
	if setFlags["topk"] {
		cfg.TopK = topK
	} else if v, err := strconv.Atoi(os.Getenv("VEC_TOP_K")); err == nil && v > 0 {
		cfg.TopK = v
	}
	if cfg.TopK <= 0 {
		cfg.TopK = defaultValues.TopK
	}
	return cfg
}

// Validate 校验 ask 命令必需的 LLM 配置项是否齐全。
func (c *Config) Validate() error {
	if c.LLMBaseURL == "" {
		return fmt.Errorf("缺少 LLM 服务地址：请用 --llm-base-url 或环境变量 VEC_LLM_BASE_URL 指定")
	}
	if c.LLMAPIKey == "" {
		return fmt.Errorf("缺少 LLM API Key：请用 --llm-api-key 或环境变量 VEC_LLM_API_KEY 指定")
	}
	if c.LLMModel == "" {
		return fmt.Errorf("缺少 LLM 模型名：请用 --llm-model 或环境变量 VEC_LLM_MODEL 指定")
	}
	return nil
}

// firstNonEmpty 返回第一个非空字符串。
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// strIf 按条件返回原值或空串，用于"flag 显式设置时才有值"。
func strIf(ok bool, v string) string {
	if ok {
		return v
	}
	return ""
}
