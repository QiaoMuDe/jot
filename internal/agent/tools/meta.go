package tools

// 本文件是 Agent 内置工具清单（展示文案）的单一事实来源：
// BuiltinTools 返回全部内置工具的名称与一行中文说明，供前端工具清单展示使用；
// 新增内置工具时须同步在此追加一条（名称须与 registry.go 注册名一致），
// 并保持顺序稳定（先后顺序即展示顺序）。
type ToolMeta struct {
	Name     string // 英文工具名（与 registry.go 注册名一致）
	Label    string // 一行中文说明
	PlanOnly bool   // 仅 Plan 模式可用（Agent 模式下不注册）
	AlwaysOn bool   // 常驻/不可禁用（后端装配时强制从禁用集合剔除；前端不可勾选）
}

// BuiltinTools 返回全部内置工具元信息，顺序即展示顺序。
func BuiltinTools() []ToolMeta {
	return []ToolMeta{
		{Name: "read_url", Label: "读取网页链接内容"},
		{Name: "http_request", Label: "发起 HTTP 请求调用 API（GET/POST/PUT/DELETE）"},
		{Name: "recall_notes", Label: "召回本地笔记，基于向量相似度检索"},
		{Name: "json_validate", Label: "校验 JSON 合法性并定位错误位置"},
		{Name: "json_format", Label: "美化格式化 JSON 文本"},
		{Name: "json_extract", Label: "按路径从 JSON 提取字段"},
		{Name: "manage_todo", Label: "管理待办（创建/查看/修改/勾选等）"},
		{Name: "manage_notebook", Label: "管理笔记本（创建/重命名/查看等）"},
		{Name: "manage_tag", Label: "管理标签（创建/查看/重命名/改色等）"},
		{Name: "manage_memory", Label: "管理长期记忆（保存/更新/删除/列出）", AlwaysOn: true},
		{Name: "browse_notes", Label: "浏览/阅读笔记（列出搜索或按行号/分段查看全文）"},
		{Name: "manage_note", Label: "管理笔记（创建/编辑/置顶/移动/打标签等写操作）"},
		{Name: "get_stats", Label: "获取笔记/待办/笔记本等数据统计"},
		{Name: "ask_user", Label: "向用户发起澄清提问（1-3 个问题，单选/多选）", AlwaysOn: true},
		{Name: "create_plan", Label: "制定执行计划（拆解目标为步骤列表）", PlanOnly: true},
		{Name: "update_plan", Label: "更新执行计划（标记步骤完成/跳过/新增）", PlanOnly: true},
		{Name: "read_file", Label: "读取工作目录内的文件（支持分页续读）"},
		{Name: "write_file", Label: "在工作目录内创建/覆盖/追加写入文件"},
		{Name: "edit_file", Label: "精准编辑工作目录内的文件（片段替换/行级替换）"},
		{Name: "ls_dir", Label: "列出某目录的直接内容（单层，像 ls）"},
		{Name: "glob", Label: "按通配符模式查找工作目录内的文件路径"},
		{Name: "grep_file", Label: "在工作目录内按内容搜索匹配行（类似 grep）"},
		{Name: "copy_file", Label: "复制工作目录内的文件/目录到目标位置（类似 cp）"},
		{Name: "move_file", Label: "移动工作目录内的文件/目录到目标位置（类似 mv）"},
		{Name: "delete_file", Label: "删除工作目录内的文件/目录（类似 rm，需强制审批）"},
		{Name: "mkdir_dir", Label: "在工作目录内递归创建目录（类似 mkdir -p）"},
		{Name: "run_command", Label: "在工作目录内执行系统命令（不支持 shell 语法）"},
	}
}
