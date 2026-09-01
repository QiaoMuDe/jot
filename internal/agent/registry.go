package agent

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"

	"jot/internal/agent/tools"
	"jot/internal/mcpserver"

	"gitee.com/MM-Q/fastlog"
)

// BuildParams 构建工具所需的装配上下文：父包 Deps、本轮 Request 与工具执行上下文。
type BuildParams struct {
	deps Deps
	req  Request
	ctx  *tools.Context
}

// planOnlyTools 仅在 Plan 模式下注册的工具名集合（Agent 模式跳过）。
var planOnlyTools = map[string]bool{
	"create_plan": true,
	"update_plan": true,
}

// planExecExcluded Plan 模式执行阶段排除的工具名集合。
// create_plan 在预规划阶段已通过 generatePlan() 完成，执行阶段不再需要。
var planExecExcluded = map[string]bool{
	"create_plan": true,
}

// loadMCPTools 从数据库加载 MCP 服务器配置，建立连接并获取工具。
// 返回工具列表（已过滤禁用工具）。失败仅记录日志，不中断调用方。
// ctx 用于控制连接超时，deps 提供 MCP 连接池和数据库，toolCtx 用于工具包装，disabled 是禁用工具黑名单。
func loadMCPTools(ctx context.Context, deps Deps, toolCtx *tools.Context, disabled map[string]bool) []tool.BaseTool {
	if deps.MCPServerDB == nil || deps.MCPPool == nil {
		return nil
	}

	mcpCfg, err := mcpserver.LoadFromDB(deps.MCPServerDB)
	if err != nil {
		if deps.Logger != nil {
			deps.Logger.Debugw("MCP 服务器配置读取失败，跳过 MCP 工具装配",
				fastlog.Error(err))
		}
		return nil
	}
	if len(mcpCfg.Servers) == 0 {
		if deps.Logger != nil {
			deps.Logger.Debugw("无启用的 MCP 服务器，跳过 MCP 工具装配")
		}
		return nil
	}

	// 单条服务器校验失败时该条被跳过，其余合法条目正常装配；逐条输出告警便于定位
	if deps.Logger != nil {
		for _, loadErr := range mcpCfg.LoadErrors {
			deps.Logger.Warnw("MCP 服务器配置校验失败，该服务器已跳过",
				fastlog.Error(loadErr))
		}
	}
	enabledServers := mcpCfg.EnabledServers()
	if len(enabledServers) == 0 {
		if deps.Logger != nil {
			deps.Logger.Debugw("MCP 配置无启用的服务器，跳过 MCP 工具装配")
		}
		return nil
	}

	// 并行取/建会话，串行处理结果（保持工具顺序与日志顺序稳定）：
	// 未命中池时现场建连（WarmupOne），goroutine 并发最多 3 台
	// （stdio 为本地子进程，限制并发拉起进程数）；
	// 每台内部已有 10s 连接 + 10s 工具发现超时兜底，goroutine 不会永久挂起。
	type mcpResult struct {
		server   mcpserver.Server
		sess     *mcpserver.Session
		err      error
		duration time.Duration
	}
	results := make([]mcpResult, len(enabledServers))
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	for i, server := range enabledServers {
		wg.Add(1)
		go func(i int, server mcpserver.Server) {
			defer wg.Done()
			sem <- struct{}{} // 获取并发槽位
			defer func() { <-sem }()
			connStart := time.Now()
			// 优先复用预热池；未命中则现场连接并入池（兜底）
			mcpSess := deps.MCPPool.Session(server.Name)
			var err error
			if mcpSess == nil {
				mcpSess, err = deps.MCPPool.WarmupOne(ctx, server)
			}
			results[i] = mcpResult{server: server, sess: mcpSess, err: err, duration: time.Since(connStart)}
		}(i, server)
	}
	wg.Wait()

	// 按索引顺序串行处理结果：日志输出与工具装配顺序和串行实现完全一致
	var toolList []tool.BaseTool
	for i := range results {
		r := results[i]
		if r.err != nil {
			if deps.Logger != nil {
				deps.Logger.Warnw("MCP 服务器连接失败，跳过该服务器",
					fastlog.String("server", r.server.Name),
					fastlog.Int("duration_ms", int(r.duration.Milliseconds())),
					fastlog.Error(r.err))
			}
			continue
		}
		// 连接由池持有常驻，本轮不关闭
		if r.sess.Skipped > 0 && deps.Logger != nil {
			deps.Logger.Warnw("部分 MCP 工具因 Info 解析失败被跳过",
				fastlog.String("server", r.server.Name),
				fastlog.Int("skipped", r.sess.Skipped))
		}
		var toolNames []string
		for _, t := range r.sess.Tools {
			invokable, ok := t.(tool.InvokableTool)
			if !ok {
				if deps.Logger != nil {
					deps.Logger.Warnw("MCP 工具不支持执行，已跳过",
						fastlog.String("server", r.server.Name))
				}
				continue
			}
			// 取改名后的工具名（mcp_{服务器名}_{工具名}），供 WrapWithError 日志与调用记录使用
			mcpToolName := r.server.Name
			if info, err := t.Info(ctx); err == nil && info != nil {
				mcpToolName = info.Name
			}
			// 检查是否在禁用名单中：被禁工具跳过注册，模型不可见也不可调用
			if disabled[mcpToolName] {
				continue
			}
			toolNames = append(toolNames, mcpToolName)
			toolList = append(toolList, tools.WrapWithError(mcpToolName, invokable, toolCtx))
		}
		// 上线日志：记录本服务器装配完成的 MCP 工具（改名后名称）与取/建会话耗时，
		// 便于排查工具是否生效及定位慢服务器（池复用场景耗时接近 0）
		if deps.Logger != nil {
			deps.Logger.Infow("MCP 服务器工具已上线",
				fastlog.String("server", r.server.Name),
				fastlog.Int("count", len(toolNames)),
				fastlog.String("tools", strings.Join(toolNames, ", ")),
				fastlog.Int("duration_ms", int(r.duration.Milliseconds())))
		}
	}
	return toolList
}

// buildToolMetas 从工具列表提取元信息（名称和描述）。
// 供 generatePlan 生成可用工具列表字符串使用。
func buildToolMetas(ctx context.Context, toolList []tool.BaseTool) []tools.ToolMeta {
	metas := make([]tools.ToolMeta, 0, len(toolList))
	for _, t := range toolList {
		if info, err := t.Info(ctx); err == nil && info != nil {
			metas = append(metas, tools.ToolMeta{
				Name:  info.Name,
				Label: info.Desc,
			})
		}
	}
	return metas
}

// buildTools 统一装配 Agent 工具，并按 disabled 和 planMode 过滤。
// planMode=false 时不注册 planOnlyTools 中的工具（create_plan/update_plan）。
// 新增工具：1) 在 tools/ 子包新增工具文件与导出构造器；2) 在此追加一行注册；
// 3) 在 tools/meta.go 的 BuiltinTools 追加展示文案（名称须与注册名一致）；
// 无需改动 Run() 的事件消费逻辑。
func buildTools(p BuildParams, disabled map[string]bool, planMode bool) []tool.BaseTool {
	type namedTool struct {
		name string
		t    tool.BaseTool
	}
	// 先收集「名字 + 已包装工具」的中间结构，再按 disabled[name] 跳过被禁工具
	all := []namedTool{
		{"read_url", tools.WrapWithError("read_url", tools.NewReadURL(p.deps.Setting, p.ctx), p.ctx)},
		{"http_request", tools.WrapWithError("http_request", tools.NewHTTP(p.deps.Setting, p.ctx), p.ctx)},
		{"recall_notes", tools.WrapWithError("recall_notes", tools.NewRecallNotes(p.deps.Vector, p.deps.Setting, p.deps.GetEmbedConfig, p.req.RecallNotebookIDs, p.ctx), p.ctx)},
		{"get_current_time", tools.WrapWithError("get_current_time", tools.NewGetCurrentTime(), p.ctx)},
		{"json_validate", tools.WrapWithError("json_validate", tools.MustJSONValidate(), p.ctx)},
		{"json_format", tools.WrapWithError("json_format", tools.MustJSONFormat(), p.ctx)},
		{"json_extract", tools.WrapWithError("json_extract", tools.MustJSONExtract(), p.ctx)},
		{"manage_todo", tools.WrapWithError("manage_todo", tools.NewManageTodo(p.deps.Todo, p.ctx), p.ctx)},
		{"manage_notebook", tools.WrapWithError("manage_notebook", tools.NewManageNotebook(p.deps.Notebook, p.ctx), p.ctx)},
		{"manage_tag", tools.WrapWithError("manage_tag", tools.NewManageTag(p.deps.Tag, p.ctx), p.ctx)},
		{"manage_note", tools.WrapWithError("manage_note", tools.NewManageNote(p.deps.Note, p.deps.Tag, p.deps.Setting, p.ctx), p.ctx)},
		{"read_note_section", tools.WrapWithError("read_note_section", tools.NewReadNoteSection(p.deps.Note, p.deps.Setting, p.ctx), p.ctx)},
		{"get_stats", tools.WrapWithError("get_stats", tools.NewGetStats(p.deps.Stats, p.deps.Vector, p.ctx), p.ctx)},
		{"ask_user", tools.WrapWithError("ask_user", tools.NewAskUser(p.ctx), p.ctx)},
		{"create_plan", tools.WrapWithError("create_plan", tools.NewCreatePlan(p.ctx), p.ctx)},
		{"update_plan", tools.WrapWithError("update_plan", tools.NewUpdatePlan(p.ctx), p.ctx)},
	}
	filtered := make([]tool.BaseTool, 0, len(all))
	for _, n := range all {
		if disabled[n.name] {
			continue
		}
		// Agent 模式下跳过仅 Plan 模式可用的工具
		if !planMode && planOnlyTools[n.name] {
			continue
		}
		// Plan 模式执行阶段跳过预规划阶段已使用的工具（create_plan）
		if planMode && planExecExcluded[n.name] {
			continue
		}
		filtered = append(filtered, n.t)
	}
	return filtered
}
