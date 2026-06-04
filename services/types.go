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
	TotalNotes   int64 `json:"total_notes"`
	TrashedNotes int64 `json:"trashed_notes"`
	PinnedNotes  int64 `json:"pinned_notes"`
	TotalTags    int64 `json:"total_tags"`
}

// ExportNoteItem 导出笔记的结构
type ExportNoteItem struct {
	Title     string        `json:"title"`
	Content   string        `json:"content"`
	Pinned    bool          `json:"pinned"`
	Tags      []ExportTag   `json:"tags,omitempty"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// ExportTag 导出标签的结构
type ExportTag struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// ImportResult 导入操作的结果
type ImportResult struct {
	SuccessCount int    `json:"success_count"`
	FailCount    int    `json:"fail_count"`
	SkippedCount int    `json:"skipped_count"`
	Message      string `json:"message"`
}
