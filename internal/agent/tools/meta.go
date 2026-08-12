package tools

// 本文件是 Agent 内置工具清单（展示文案）的单一事实来源：
// BuiltinTools 返回全部内置工具的名称与一行中文说明，供前端工具清单展示使用；
// 新增内置工具时须同步在此追加一条（名称须与 registry.go 注册名一致），
// 并保持顺序稳定（先后顺序即展示顺序）。
type ToolMeta struct {
	Name  string // 英文工具名（与 registry.go 注册名一致）
	Label string // 一行中文说明
}

// BuiltinTools 返回全部内置工具元信息，顺序即展示顺序。
func BuiltinTools() []ToolMeta {
	return []ToolMeta{
		{Name: "refine_search_query", Label: "优化搜索关键词，提升搜索准确率（联网搜索的辅助）"},
		{Name: "web_search", Label: "联网搜索（多来源），获取实时信息"},
		{Name: "recall_notes", Label: "召回本地笔记，基于向量相似度检索"},
		{Name: "get_current_time", Label: "获取当前时间/日期/星期"},
		{Name: "manage_todo", Label: "创建、查看、修改、删除待办"},
		{Name: "manage_notebook", Label: "管理笔记本（创建/重命名/删除等）"},
		{Name: "manage_tag", Label: "管理标签（创建/重命名/合并/删除等）"},
		{Name: "manage_note", Label: "管理笔记（创建/查看/修改/删除等）"},
		{Name: "get_stats", Label: "获取笔记/待办/笔记本等数据统计"},
		{Name: "ask_user", Label: "向用户发起澄清提问（单选/多选）"},
	}
}
