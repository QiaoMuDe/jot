package database

import "jot/internal/models"

// AllModels 全部数据模型，按"子表在前"顺序排列
// 供 AutoMigrate（InitDB / ResetDatabase）与 DropTable（ResetDatabase）共用，
// 新增模型只需在此注册一处，两端自动同步
var AllModels = []interface{}{
	&models.AIMessage{},       // 子表：SessionID → AISession
	&models.AISessionConfig{}, // 子表：SessionID → AISession
	&models.AISession{},       // 会话
	&models.AIPrompt{},        // 提示词
	&models.APIProfile{},      // API 配置
	&models.Todo{},            // 待办事项
	&models.Setting{},         // 设置
	&models.NoteVector{},      // 子表：NoteID → Note
	&models.Note{},            // 子表：NotebookID → Notebook
	&models.Tag{},             // 标签
	&models.Notebook{},        // 笔记本
	&models.MCPServer{},       // MCP 服务器
	&models.PasswordRecord{},  // 密码记录
	&models.AIMemory{},        // 全局记忆
}
