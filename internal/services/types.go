package services

import "strconv"

// PaginatedResult 分页查询结果的通用结构
type PaginatedResult struct {
	Items    interface{} `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// DataStats 数据统计概览
type DataStats struct {
	TotalNotes      int64   `json:"total_notes"`
	TrashedNotes    int64   `json:"trashed_notes"`
	PinnedNotes     int64   `json:"pinned_notes"`
	TotalTags       int64   `json:"total_tags"`
	TotalNotebooks  int64   `json:"total_notebooks"`
	AISessions      int64   `json:"ai_sessions"`
	AIMessages      int64   `json:"ai_messages"`
	TotalTokens     int64   `json:"total_tokens"`
	AvgResponseTime float64 `json:"avg_response_time"`
	AvgThinkingTime float64 `json:"avg_thinking_time"`
	MaxResponseTime float64 `json:"max_response_time"`
	DBSize          int64   `json:"db_size"`
	DBSizeStr       string  `json:"db_size_str"`
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

// SettingsConfig 设置页全部配置项的统一结构体
type SettingsConfig struct {
	Theme                     string `json:"theme"`
	FontFamily                string `json:"font_family"`
	FontSize                  int    `json:"font_size"`
	CodeHighlightTheme        string `json:"code_highlight_theme"`
	NoteOpenFullscreen        bool   `json:"note_open_fullscreen"`
	SortOrder                 string `json:"sort_order"`
	PageSize                  int    `json:"page_size"`
	QuickNoteEnabled          bool   `json:"quick_note_enabled"`
	CMSyntaxHighlight         bool   `json:"cm_syntax_highlight"`
	AIProvider                string `json:"ai_provider"`
	AIBaseURL                 string `json:"ai_base_url"`
	AIAPIKey                  string `json:"ai_api_key"`
	AIModel                   string `json:"ai_model"`
	TavilyAPIKey              string `json:"tavily_api_key"`
	AIThinkingEnabled         bool   `json:"ai_thinking_enabled"`
	ZhihuAccessSecret         string `json:"zhihu_access_secret"`
	ZhihuSearchEnabled        bool   `json:"zhihu_search_enabled"`
	ZhihuGlobalSearchEnabled  bool   `json:"zhihu_global_search_enabled"`
	TavilySearchEnabled       bool   `json:"tavily_search_enabled"`
	AICardRecallEnabled       bool   `json:"ai_card_recall_enabled"`
	AICardRecallLimit         int    `json:"ai_card_recall_limit"`
	AIRefMaxChars             int    `json:"ai_ref_max_chars"`
	AISearchResultLimit       int    `json:"ai_search_result_limit"`
	TrashCleanupRetentionDays int    `json:"trash_cleanup_retention_days"`
}

// GetAllSettings 从 SettingService 读取全部设置项
func (s *SettingService) GetAllSettings() SettingsConfig {
	return SettingsConfig{
		Theme:                     s.Get("theme"),
		FontFamily:                s.Get("font_family"),
		FontSize:                  parseIntSetting(s.Get("font_size"), 16),
		CodeHighlightTheme:        s.Get("code_highlight_theme"),
		NoteOpenFullscreen:        parseBoolSetting(s.Get("note_open_fullscreen")),
		SortOrder:                 s.Get("sort_order"),
		PageSize:                  parseIntSetting(s.Get("page_size"), 20),
		QuickNoteEnabled:          parseBoolSetting(s.Get("quick_note_enabled")),
		CMSyntaxHighlight:         parseBoolSetting(s.Get("cm_syntax_highlight")),
		AIProvider:                s.Get("ai_provider"),
		AIBaseURL:                 s.Get("ai_base_url"),
		AIAPIKey:                  s.Get("ai_api_key"),
		AIModel:                   s.Get("ai_model"),
		TavilyAPIKey:              s.Get("tavily_api_key"),
		AIThinkingEnabled:         parseBoolSetting(s.Get("ai_thinking_enabled")),
		ZhihuAccessSecret:         s.Get("zhihu_access_secret"),
		ZhihuSearchEnabled:        parseBoolSetting(s.Get("zhihu_search_enabled")),
		ZhihuGlobalSearchEnabled:  parseBoolSetting(s.Get("zhihu_global_search_enabled")),
		TavilySearchEnabled:       parseBoolSetting(s.Get("tavily_search_enabled")),
		AICardRecallEnabled:       parseBoolSetting(s.Get("ai_card_recall_enabled")),
		AICardRecallLimit:         parseIntSetting(s.Get("ai_card_recall_limit"), 5),
		AIRefMaxChars:             parseIntSetting(s.Get("ai_ref_max_chars"), 5000),
		AISearchResultLimit:       parseIntSetting(s.Get("ai_search_result_limit"), 5),
		TrashCleanupRetentionDays: parseIntSetting(s.Get("trash_cleanup_retention_days"), 30),
	}
}

// SaveAllSettings 保存全部设置项到 SettingService，含类型校验
func (s *SettingService) SaveAllSettings(cfg SettingsConfig) error {
	if cfg.PageSize < 1 {
		cfg.PageSize = 20
	} else if cfg.PageSize > 100 {
		cfg.PageSize = 100
	}
	if cfg.FontSize < 8 {
		cfg.FontSize = 16
	} else if cfg.FontSize > 72 {
		cfg.FontSize = 72
	}
	if cfg.AICardRecallLimit < 1 {
		cfg.AICardRecallLimit = 5
	} else if cfg.AICardRecallLimit > 30 {
		cfg.AICardRecallLimit = 30
	}
	if cfg.AIRefMaxChars < 1 {
		cfg.AIRefMaxChars = 5000
	} else if cfg.AIRefMaxChars > 50000 {
		cfg.AIRefMaxChars = 50000
	}
	if cfg.AISearchResultLimit < 1 {
		cfg.AISearchResultLimit = 5
	} else if cfg.AISearchResultLimit > 30 {
		cfg.AISearchResultLimit = 30
	}
	if cfg.TrashCleanupRetentionDays < 1 {
		cfg.TrashCleanupRetentionDays = 30
	} else if cfg.TrashCleanupRetentionDays > 365 {
		cfg.TrashCleanupRetentionDays = 365
	}

	sets := map[string]string{
		"theme":                        cfg.Theme,
		"font_family":                  cfg.FontFamily,
		"font_size":                    strconv.Itoa(cfg.FontSize),
		"code_highlight_theme":         cfg.CodeHighlightTheme,
		"note_open_fullscreen":         strconv.FormatBool(cfg.NoteOpenFullscreen),
		"sort_order":                   cfg.SortOrder,
		"page_size":                    strconv.Itoa(cfg.PageSize),
		"quick_note_enabled":           strconv.FormatBool(cfg.QuickNoteEnabled),
		"cm_syntax_highlight":          strconv.FormatBool(cfg.CMSyntaxHighlight),
		"ai_provider":                  cfg.AIProvider,
		"ai_base_url":                  cfg.AIBaseURL,
		"ai_api_key":                   cfg.AIAPIKey,
		"ai_model":                     cfg.AIModel,
		"tavily_api_key":               cfg.TavilyAPIKey,
		"ai_thinking_enabled":          strconv.FormatBool(cfg.AIThinkingEnabled),
		"zhihu_access_secret":          cfg.ZhihuAccessSecret,
		"zhihu_search_enabled":         strconv.FormatBool(cfg.ZhihuSearchEnabled),
		"zhihu_global_search_enabled":  strconv.FormatBool(cfg.ZhihuGlobalSearchEnabled),
		"tavily_search_enabled":        strconv.FormatBool(cfg.TavilySearchEnabled),
		"ai_card_recall_enabled":       strconv.FormatBool(cfg.AICardRecallEnabled),
		"ai_card_recall_limit":         strconv.Itoa(cfg.AICardRecallLimit),
		"ai_ref_max_chars":             strconv.Itoa(cfg.AIRefMaxChars),
		"ai_search_result_limit":       strconv.Itoa(cfg.AISearchResultLimit),
		"trash_cleanup_retention_days": strconv.Itoa(cfg.TrashCleanupRetentionDays),
	}

	for k, v := range sets {
		if err := s.Set(k, v); err != nil {
			return err
		}
	}
	return nil
}

// parseIntSetting 解析 int 类型设置值，解析失败返回默认值
func parseIntSetting(val string, defaultVal int) int {
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}

// parseBoolSetting 解析 bool 类型设置值，只认 "true" 为 true
func parseBoolSetting(val string) bool {
	return val == "true"
}
