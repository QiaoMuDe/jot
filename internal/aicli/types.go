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
	BaseURL string // 例如 https://api.openai.com/v1 或任意 OpenAI 兼容端点
	APIKey  string
	Model   string
}
