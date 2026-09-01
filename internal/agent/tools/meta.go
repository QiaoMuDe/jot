package tools

// 本文件是 Agent 内置工具清单（展示文案）的单一事实来源：
// BuiltinTools 返回全部内置工具的名称与一行中文说明，供前端工具清单展示使用；
// 新增内置工具时须同步在此追加一条（名称须与 registry.go 注册名一致），
// 并保持顺序稳定（先后顺序即展示顺序）。
type ToolMeta struct {
	Name     string // 英文工具名（与 registry.go 注册名一致）
	Label    string // 一行中文说明
	PlanOnly bool   // 仅 Plan 模式可用（Agent 模式下不注册）
}

// BuiltinTools 返回全部内置工具元信息，顺序即展示顺序。
func BuiltinTools() []ToolMeta {
	return []ToolMeta{
		{Name: "read_url", Label: "读取网页链接内容"},
		{Name: "http_request", Label: "发起 HTTP 请求调用 API（GET/POST/PUT/DELETE）"},
		{Name: "recall_notes", Label: "召回本地笔记，基于向量相似度检索"},
		{Name: "get_current_time", Label: "获取当前时间/日期/星期"},
		{Name: "json_validate", Label: "校验 JSON 合法性并定位错误位置"},
		{Name: "json_format", Label: "美化格式化 JSON 文本"},
		{Name: "json_extract", Label: "按路径从 JSON 提取字段"},
		{Name: "manage_todo", Label: "管理待办（创建/查看/修改/勾选等）"},
		{Name: "manage_notebook", Label: "管理笔记本（创建/重命名/查看等）"},
		{Name: "manage_tag", Label: "管理标签（创建/查看/重命名/改色等）"},
		{Name: "manage_note", Label: "管理笔记（创建/查看/编辑/置顶/移动/打标签等）"},
		{Name: "read_note_section", Label: "分段读取笔记内容"},
		{Name: "get_stats", Label: "获取笔记/待办/笔记本等数据统计"},
		{Name: "ask_user", Label: "向用户发起澄清提问（单选/多选）"},
		{Name: "create_plan", Label: "制定执行计划（拆解目标为步骤列表）", PlanOnly: true},
		{Name: "update_plan", Label: "更新执行计划（标记步骤完成/跳过/新增）", PlanOnly: true},
	}
}
