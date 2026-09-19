package agent

// 本文件定义 os_agent（操作系统子 Agent）实例: 通用子 Agent 机制（subagent.go 的
// delegatedAgentTool/newDelegatedAgentTool）+ 一份配置（osAgentConfig）即完成装配,
// 是「新增域子 Agent」的样板（新增子 Agent 开发规范见 subagent.go 文件头）。
//
// 背景: read_file/write_file/edit_file/ls_dir/glob/grep_file/copy_item/move_item/
// delete_item/mkdir_dir/run_command/run_python/transfer_item 这 13 个文件/命令工具
// 原先直接注册在父层（registry.go buildTools）, 每轮 LLM 调用都要付十几份工具描述
// token。现在封装为 os_agent 委托工具: 父层只看到 os_agent 一个工具, 内层是独立
// ChatModelAgent（复用同一会话 *openai.ChatModel 客户端）, 白名单 13 个工具；内层
// 每步工具调用以 ai:tool-status 事件实时转发, 审批语义沿用同一会话 Approver。

import (
	"context"
	"runtime"

	"github.com/cloudwego/eino-ext/components/model/openai"

	"jot/internal/agent/tools"
	"jot/internal/config"
	"jot/internal/services"
)

// osSubAgentInstruction 内层 os 子 Agent 的系统提示词: 角色 + 边界 + 审批说明 + 结果摘要要求。
// 末尾拼接当前运行平台（runtime.GOOS）, 模型据此感知平台、避免照抄不可用命令
// 导致 run_command 报错浪费工具调用。
var osSubAgentInstruction = `你是操作系统任务执行子 Agent, 负责在工作区内完成文件与命令类任务。

边界: 
- 仅允许操作 ~/.jot/workspace/ 工作区内的路径（路径合法性由各工具自行校验, 越界会被拒绝）。路径可直接写相对工作区的路径（如 notes/a.md）, 或 ~/.jot/workspace/... 形式, 两者均可解析。
- 不要猜测 ~ 的展开路径（如 /home/user/.jot/workspace）, 家目录展开由工具自行处理; 一律使用相对工作区的路径（如 memsh-0.0.15/cmd）或以 ~/.jot/workspace/ 开头的路径, 不要传本机绝对路径（不同平台绝对路径格式不同, 会导致路径无效）。
- 与用户桌面交换文件（下载产出到桌面 / 上传桌面资料到工作区）用 transfer_item, 仅能操作工作目录与桌面两端, 无法访问其它位置。
- Python 代码/脚本请用 run_python 工具（自动探测环境解释器, 无需指定 python/python3/py 名）, 不要用 run_command 自行猜测解释器名。
- 文件与命令之外的诉求（笔记读写、联网搜索、向用户提问等）不要自行处理, 在最终回复中说明「该诉求由主 Agent 处理」。
- 危险操作（如 run_command 命中黑名单、transfer_item 下载到桌面）可能触发用户确认（取决于当前审批模式）；若被拒绝请改用其他方式完成, 或向用户说明原因后避开该操作。

任务完成时给出简洁的结构化摘要（控制篇幅）: 做了什么、关键结果、遗留问题。` + "\n\n当前平台: " + runtime.GOOS

// osSubAgentToolNames 内层白名单: 13 个文件/命令工具（即原先直接注册在父层的文件
// 工具家族 + run_python + transfer_item）。顺序即注册顺序。
var osSubAgentToolNames = []string{
	"read_file", "write_file", "edit_file", "ls_dir", "glob", "grep_file",
	"copy_item", "move_item", "delete_item", "mkdir_dir", "run_command",
	"run_python", "transfer_item",
}

// osAgentConfig os_agent 差异配置: 通用工厂 + 一份配置即完成装配。
// name 同时作为内层 ChatModelAgent 名（重构前为 "os-agent", 统一为 "os_agent", 无代码依赖旧名）。
var osAgentConfig = subAgentConfig{
	name:          "os_agent",
	description:   "操作系统任务执行子 Agent: 在工作区内读写文件、查找内容、执行命令",
	instruction:   osSubAgentInstruction,
	toolNames:     osSubAgentToolNames,
	maxIterations: config.AISubAgentMaxIterationsDefault,
	actionPrefix:  "执行操作系统任务: ",
	infoDesc:      "将文件与命令类任务委托给操作系统子 Agent 执行。当任务需要在工作区（~/.jot/workspace）内读写文件、查找内容、编辑、复制/移动/删除、创建目录、执行命令、运行 Python 代码/脚本, 或与用户桌面交换文件（下载/上传）时调用；内层子 Agent 会自主规划并调用 read_file/write_file/edit_file/ls_dir/glob/grep_file/copy_item/move_item/delete_item/mkdir_dir/run_command/run_python/transfer_item 完成。request 为完整任务描述（一句话说明目标、路径与约束）。仅处理工作区内的文件与命令诉求及与桌面的文件交换；笔记、网络等其它诉求请直接使用对应工具, 不要委托给本工具。",
	requestDesc:   "要委托给操作系统子 Agent 执行的完整任务描述（目标、涉及路径、约束）",
}

// buildOSSubAgent 构造 os_agent 委托工具（registry.go 装配入口；chatModel 为 nil 或构造失败返回 nil）。
// 迭代上限从设置项 ai_sub_agent_max_iterations 读取（拷贝配置后覆盖，避免修改包级共享配置），
// 默认值与上限引用 config 常量，与种子初始化 / 设置页校验口径一致。
func buildOSSubAgent(runCtx context.Context, chatModel *openai.ChatModel, innerCtx *tools.Context, setting *services.SettingService) *delegatedAgentTool {
	cfg := osAgentConfig
	cfg.maxIterations = iterationLimitFromSetting(setting, "ai_sub_agent_max_iterations",
		config.AISubAgentMaxIterationsDefault, config.AISubAgentMaxIterationsMax)
	return newDelegatedAgentTool(runCtx, chatModel, innerCtx, cfg)
}
