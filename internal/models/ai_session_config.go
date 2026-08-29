package models

// AISessionConfig 存储每个 AI 会话的操作栏配置，与 AISession 一对一关联
type AISessionConfig struct {
	ID                uint   `gorm:"primaryKey" json:"id"`
	SessionID         uint   `gorm:"uniqueIndex;not null" json:"session_id"`
	ModelName         string `gorm:"size:100;default:''" json:"model_name"`
	EnableThinking    bool   `gorm:"default:false" json:"enable_thinking"`
	ReferencedNotes   string `gorm:"type:text" json:"referenced_notes"`
	EnabledSkills     string `gorm:"type:text" json:"enabled_skills"`
	RoleplayNotes     string `gorm:"type:text;default:''" json:"roleplay_notes"`
	RecallNotebookIDs string `gorm:"type:text" json:"recall_notebook_ids"`
	Mode              string `gorm:"size:10;default:'agent'" json:"mode"`
}
