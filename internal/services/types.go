package services

// PaginatedResult 分页查询结果的通用结构
type PaginatedResult struct {
	Items    interface{} `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// DataStats 数据统计概览
type DataStats struct {
	TotalNotes     int64  `json:"total_notes"`
	TrashedNotes   int64  `json:"trashed_notes"`
	PinnedNotes    int64  `json:"pinned_notes"`
	TotalTags      int64  `json:"total_tags"`
	TotalNotebooks int64  `json:"total_notebooks"`
	DBSize         int64  `json:"db_size"`
	DBSizeStr      string `json:"db_size_str"`
}

// ImportResult 导入操作的结果
type ImportResult struct {
	SuccessCount int    `json:"success_count"`
	FailCount    int    `json:"fail_count"`
	SkippedCount int    `json:"skipped_count"`
	Message      string `json:"message"`
}

// NoteRefInfo 笔记引用信息，用于前端 chip 展示
type NoteRefInfo struct {
	ID           uint   `json:"id"`
	Title        string `json:"title"`
	Truncated    bool   `json:"truncated"`
	NotebookName string `json:"notebook_name"`
}

// NoteRefContext 笔记引用上下文，后端一次性完成截断并返回
type NoteRefContext struct {
	Notes   []NoteRefInfo `json:"notes"`
	Context string        `json:"context"`
}
