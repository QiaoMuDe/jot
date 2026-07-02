package aicli

// Message 表示 AI 对话中的一条消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamCallbacks 流式响应的回调接口
type StreamCallbacks struct {
	OnChunk    func(text string)
	OnThinking func(text string)
	OnDone     func(fullContent string, thinkingElapsed float64, totalElapsed float64)
	OnError    func(errMsg string)
}

// Config 客户端配置
type Config struct {
	Provider string // "openai" 或 "ollama"
	BaseURL  string // 例如 http://localhost:11434（Ollama）或 https://api.openai.com/v1（OpenAI）
	APIKey   string
	Model    string
}
