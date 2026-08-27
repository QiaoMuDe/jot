package services

// stats_service.go 实现数据统计聚合服务。
//
// StatsService 是应用数据概览（DataStats）的单一事实来源：聚合 NoteService（笔记/回收站/
// 置顶/笔记本数）、TagService（标签数）、AIService（会话/消息/token/耗时）、TodoService
// （待办数）、PasswordService（密码记录数）以及数据库文件大小，供数据管理页面（App.GetDataStats 绑定）与 Agent 工具
// （get_stats）共用，保证两者口径完全一致，避免重复逻辑漂移。
// 数据库文件路径通过构造器注入函数获取（默认传入 database.DefaultDBPath），避免
// services 包反向依赖 internal/database 造成循环引用。

import (
	"fmt"
	"os"
)

// StatsService 数据统计聚合服务（只读）。
type StatsService struct {
	note   *NoteService           // 笔记统计（GetStats）
	tag    *TagService            // 标签数（Count）
	todo   *TodoService           // 待办数（Count / CountCompleted）
	pw     *PasswordService       // 密码记录数（Count）
	ai     *AIService             // AI 用量与耗时统计
	dbPath func() (string, error) // 数据库文件路径获取函数（由 app 层注入 database.DefaultDBPath）
}

// NewStatsService 创建数据统计聚合服务。
// dbPath 用于获取 SQLite 数据库文件路径以计算占用大小，app 层传 database.DefaultDBPath。
func NewStatsService(note *NoteService, tag *TagService, todo *TodoService, pw *PasswordService, ai *AIService, dbPath func() (string, error)) *StatsService {
	return &StatsService{note: note, tag: tag, todo: todo, pw: pw, ai: ai, dbPath: dbPath}
}

// GetDataStats 聚合应用数据统计概览。
// 在 NoteService.GetStats 返回的笔记维度基础上，补充标签数、数据库文件大小、
// AI 用量（会话/消息/token/平均响应与思考时长/最长响应）与待办统计，
// 与数据管理页面的展示口径完全一致。
func (s *StatsService) GetDataStats() (*DataStats, error) {
	stats, err := s.note.GetStats()
	if err != nil {
		return nil, err
	}

	// 标签数（NoteService.GetStats 不统计标签）
	if tagCount, err := s.tag.Count(); err == nil {
		stats.TotalTags = tagCount
	}

	// 数据库文件大小
	if dbPath, pathErr := s.dbPath(); pathErr == nil {
		if fi, statErr := os.Stat(dbPath); statErr == nil {
			size := fi.Size()
			stats.DBSize = size
			stats.DBSizeStr = fmt.Sprintf("%d B", size)
			if size >= 1024*1024 {
				stats.DBSizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
			} else if size >= 1024 {
				stats.DBSizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
			}
		}
	}

	// AI 会话/消息统计
	if aiSessions, err := s.ai.CountSessions(); err == nil {
		stats.AISessions = aiSessions
	}
	if aiMessages, err := s.ai.CountMessages(); err == nil {
		stats.AIMessages = aiMessages
	}

	// AI 性能统计
	if totalTokens, err := s.ai.SumTokens(); err == nil {
		stats.TotalTokens = totalTokens
	}
	if avgResponseTime, err := s.ai.AvgResponseTime(); err == nil {
		stats.AvgResponseTime = avgResponseTime
	}
	if avgThinkingTime, err := s.ai.AvgThinkingTime(); err == nil {
		stats.AvgThinkingTime = avgThinkingTime
	}
	if maxResponseTime, err := s.ai.MaxResponseTime(); err == nil {
		stats.MaxResponseTime = maxResponseTime
	}

	// 待办统计
	if totalTodos, err := s.todo.Count(); err == nil {
		stats.TotalTodos = totalTodos
	}
	if completedTodos, err := s.todo.CountCompleted(); err == nil {
		stats.CompletedTodos = completedTodos
	}

	// 密码记录统计
	if pwCount, err := s.pw.Count(); err == nil {
		stats.TotalPasswords = pwCount
	}

	return stats, nil
}

// GetMonthCounts 返回指定年月的每日笔记数（委托 NoteService），供 get_stats 工具 month 动作使用。
func (s *StatsService) GetMonthCounts(year, month int) (map[int]int, error) {
	return s.note.GetMonthCounts(year, month)
}
