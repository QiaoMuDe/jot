package models

// AISessionConfig 存储每个 AI 会话的操作栏配置，与 AISession 一对一关联
type AISessionConfig struct {
	ID                       uint   `gorm:"primaryKey" json:"id"`
	SessionID                uint   `gorm:"uniqueIndex;not null" json:"session_id"`
	ModelName                string `gorm:"size:100;default:''" json:"model_name"`
	EnableThinking           bool   `gorm:"default:false" json:"enable_thinking"`
	ZhihuSearchEnabled       bool   `gorm:"default:false" json:"zhihu_search_enabled"`
	ZhihuGlobalSearchEnabled bool   `gorm:"default:false" json:"zhihu_global_search_enabled"`
	TavilySearchEnabled      bool   `gorm:"default:false" json:"tavily_search_enabled"`
	EnableCardRecall         bool   `gorm:"default:false" json:"enable_card_recall"`
	ReferencedNotes          string `gorm:"type:text" json:"referenced_notes"`
	EnabledSkills            string `gorm:"type:text" json:"enabled_skills"`
	RoleplayNotes            string `gorm:"type:text;default:''" json:"roleplay_notes"`
	RecallNotebookIDs        string `gorm:"type:text" json:"recall_notebook_ids"`
	// AgentEnabled 会话是否为 Agent 模式（模型自主调用工具），默认问答模式
	AgentEnabled bool `gorm:"default:false" json:"agent_enabled"`
}
