package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"jot/internal/agent/tools"
)

// Session 一次 MCP 服务器会话：持有客户端连接与发现包装后的工具列表。
// 由调用方（agent.Run / Pool）负责生命周期：Pool 中常驻跨轮复用，Run 兜底路径本轮用完即关。
// Skipped 记录因工具定义异常（空名）被跳过的工具数，供调用方日志告警。
type Session struct {
	ServerName string
	Tools      []tool.BaseTool
	Skipped    int
	srv        Server     // 服务器配置快照，供断线重连
	mu         sync.Mutex // 保护 cli/cancel/closed（重连与关闭并发）
	cli        *mcp.ClientSession
	cancel     func()
	closed     bool // Close 后置位，拒绝重连
}

// OpenSession 连接 MCP 服务器并发现包装工具：
// Connect 握手 → session.ListTools 拉取全部工具 → 逐个改名包装（mcp_{服务器名}_{工具名}）→ 填入 Session。
// 单个工具定义异常时跳过该工具（累加 Session.Skipped），不影响其余工具装配。
func OpenSession(ctx context.Context, s Server) (*Session, error) {
	cli, cancel, err := Connect(ctx, s)
	if err != nil {
		return nil, err
	}
	// 工具发现（ListTools）是独立网络往返，单独包超时，避免远程服务器挂起无限阻塞整轮装配
	discoverCtx, discoverCancel := context.WithTimeout(ctx, ConnectTimeout)
	defer discoverCancel()
	listResult, err := cli.ListTools(discoverCtx, &mcp.ListToolsParams{})
	if err != nil {
		cancel()
		_ = cli.Close()
		return nil, fmt.Errorf("MCP 服务器 %s 工具发现失败: %w", s.Name, err)
	}

	sess := &Session{ServerName: s.Name, srv: s, cli: cli, cancel: cancel}
	sess.Tools = make([]tool.BaseTool, 0, len(listResult.Tools))
	for _, td := range listResult.Tools {
		if td == nil || td.Name == "" {
			sess.Skipped++
			continue
		}
		sess.Tools = append(sess.Tools, &mcpTool{
			serverName:   s.Name,
			sess:         sess,
			toolDef:      td,
			originalName: td.Name,
		})
	}
	return sess, nil
}

// Close 关闭 MCP 客户端连接并取消会话 ctx；幂等，Close 后拒绝重连。
func (sess *Session) Close() error {
	if sess == nil {
		return nil
	}
	sess.mu.Lock()
	if sess.closed {
		sess.mu.Unlock()
		return nil
	}
	sess.closed = true
	cancel := sess.cancel
	sess.cancel = nil
	cli := sess.cli
	sess.cli = nil
	sess.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cli == nil {
		return nil
	}
	return cli.Close()
}

// callTool 调用 MCP 服务器工具，连接断开（连接类错误）时自动重连一次并重试。
// 重连失败或非连接类错误原样返回；会话已 Close 或 ctx 已取消时不重连。
// 重连使用 OpenSession 时的服务器配置快照（srv），成功后替换 cli 供后续调用使用。
func (sess *Session) callTool(ctx context.Context, name string, args any) (*mcp.CallToolResult, error) {
	sess.mu.Lock()
	closed := sess.closed
	cli := sess.cli
	sess.mu.Unlock()
	if closed || cli == nil {
		return nil, errors.New("MCP 会话已关闭")
	}

	result, err := cli.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err == nil {
		return result, nil
	}
	// 非连接类错误或上下文已取消：不重连，原样返回
	if !isConnError(err) || ctx.Err() != nil {
		return nil, err
	}

	// 自动重连一次：重建连接（握手+版本协商），替换 cli
	sess.mu.Lock()
	if sess.closed {
		sess.mu.Unlock()
		return nil, err
	}
	sess.mu.Unlock()
	newCli, newCancel, cerr := Connect(ctx, sess.srv)
	if cerr != nil {
		return nil, err // 重连失败：返回原始错误
	}
	sess.mu.Lock()
	if sess.closed {
		sess.mu.Unlock()
		newCancel()
		_ = newCli.Close()
		return nil, err
	}
	oldCli := sess.cli
	sess.cli = newCli
	sess.cancel = newCancel
	sess.mu.Unlock()
	if oldCli != nil {
		_ = oldCli.Close()
	}
	return newCli.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
}

// isConnError 判断错误是否为连接类错误（连接关闭/会话缺失/EOF 等），用于触发自动重连。
func isConnError(err error) bool {
	if errors.Is(err, mcp.ErrConnectionClosed) || errors.Is(err, mcp.ErrSessionMissing) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "EOF") || strings.Contains(msg, "connection closed") || strings.Contains(msg, "session not found")
}

// mcpTool 改名包装器：把 MCP 工具重命名为 mcp_{服务器名}_{原始工具名}，
// 避免与内置工具或跨服务器工具重名冲突；执行委托 Session.callTool（含断线自动重连）。
// 实现 ActionTextProvider，使父包经 tools.WrapWithError 包装后仍能断言到友好动作文案。
type mcpTool struct {
	serverName   string
	sess         *Session
	toolDef      *mcp.Tool
	originalName string
	// cachedInfo 首次 Info 构建后的改名工具信息缓存。工具定义在本轮会话内不变，
	// 缓存可消除 agent.go 装配（取名、建索引）与 eino 框架多次调用 Info 时的重复转换开销。
	cachedInfo *schema.ToolInfo
}

var _ tool.InvokableTool = (*mcpTool)(nil)
var _ tools.ActionTextProvider = (*mcpTool)(nil)

// Info 返回改名后的工具信息：Name 改为 mcp_{serverName}_{originalName}，
// Desc 用服务器描述，Params 由 InputSchema（JSON Schema）转换而来（无法解析时降级为无参数）。
// 首次调用后缓存结果，后续直接返回浅拷贝副本（复制标量与 Name，指针字段共享只读语义），
// 调用方修改返回值不影响缓存。
func (m *mcpTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	if m.cachedInfo != nil {
		copied := *m.cachedInfo
		return &copied, nil
	}
	info := &schema.ToolInfo{
		Name:  "mcp_" + m.serverName + "_" + m.originalName,
		Desc:  m.toolDef.Description,
		Extra: nil,
	}
	if po := inputSchemaToParamsOneOf(m.toolDef.InputSchema); po != nil {
		info.ParamsOneOf = po
	}
	m.cachedInfo = info
	// 首次调用同样返回浅拷贝副本，保证调用方修改返回值不影响缓存
	firstCopy := *info
	return &firstCopy, nil
}

// InvokableRun 调用 MCP 服务器工具并返回结果。返回格式与旧 eino-ext mcp 组件一致：
// CallToolResult 的 JSON 序列化（{"content":[{"type":"text","text":...}],"isError":false}），
// 前端 / 测试解析兼容；多段文本内容以换行拼接。
func (m *mcpTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args any
	if strings.TrimSpace(argumentsInJSON) != "" {
		var argMap map[string]any
		if err := json.Unmarshal([]byte(argumentsInJSON), &argMap); err != nil {
			return "", fmt.Errorf("MCP 工具 %s 参数解析失败: %w", m.originalName, err)
		}
		args = argMap
	}
	result, err := m.sess.callTool(ctx, m.originalName, args)
	if err != nil {
		return "", fmt.Errorf("MCP 工具 %s 调用失败: %w", m.originalName, err)
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("MCP 工具 %s 结果序列化失败: %w", m.originalName, err)
	}
	return string(out), nil
}

// ActionText 提供工具开始执行时的动作文案（前端状态条展示"调用 {服务器} 的 {工具}"）。
func (m *mcpTool) ActionText(argumentsInJSON string) string {
	return fmt.Sprintf("调用 %s 的 %s", m.serverName, m.originalName)
}

// inputSchemaToParamsOneOf 将 MCP 工具的 InputSchema（任意 JSON Schema，通常为 map[string]any）
// 转换为 eino ParamsOneOf（JSON Schema 形式）。转换失败（非法 JSON / 非对象 Schema）时
// 返回 nil，调用方降级为「无参数」工具，保证装配不中断。
func inputSchemaToParamsOneOf(inputSchema any) *schema.ParamsOneOf {
	if inputSchema == nil {
		return nil
	}
	b, err := json.Marshal(inputSchema)
	if err != nil {
		return nil
	}
	var js jsonschema.Schema
	if err := js.UnmarshalJSON(b); err != nil {
		return nil
	}
	return schema.NewParamsOneOfByJSONSchema(&js)
}
