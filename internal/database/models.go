package database

import "jot/internal/models"

// AllModels 全部数据模型，按"子表在前"顺序排列
// 供 AutoMigrate（InitDB / ResetDatabase）与 DropTable（ResetDatabase）共用，
// 新增模型只需在此注册一处，两端自动同步
var AllModels = []interface{}{
	&models.AIMessage{},       // 子表：SessionID → AISession
	&models.AISessionConfig{}, // 子表：SessionID → AISession
	&models.AISession{},
	&models.AIPrompt{},
	&models.APIProfile{},
	&models.Todo{},
	&models.Setting{},
	&models.NoteVector{}, // 子表：NoteID → Note
	&models.Note{},       // 子表：NotebookID → Notebook
	&models.Tag{},
	&models.Notebook{},
	&models.MCPServer{},
}
