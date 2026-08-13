package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"

	"jot/internal/agent/tools"
)

// Session 一次 MCP 服务器会话：持有客户端连接与发现包装后的工具列表。
// 由调用方（agent.Run）在每轮对话结束时调用 Close 释放连接。
// Skipped 记录因 Info 解析失败（错误/为空/无名）被跳过的工具数，供调用方日志告警。
type Session struct {
	ServerName string
	Tools      []tool.BaseTool
	Skipped    int
	cli        client.MCPClient
}

// OpenSession 连接 MCP 服务器并发现包装工具：
// Connect 握手 → mcpp.GetTools 拉取全部工具 → 逐个改名包装（mcp_{服务器名}_{工具名}）→ 填入 Session。
// 单个工具 Info 异常时跳过该工具（累加 Session.Skipped），不影响其余工具装配。
func OpenSession(ctx context.Context, s Server) (*Session, error) {
	cli, err := Connect(ctx, s)
	if err != nil {
		return nil, err
	}
	// 工具发现（ListTools）是独立网络往返，单独包超时，避免远程服务器挂起无限阻塞整轮装配
	discoverCtx, cancel := context.WithTimeout(ctx, ConnectTimeout)
	defer cancel()
	baseTools, err := mcpp.GetTools(discoverCtx, &mcpp.Config{Cli: cli})
	if err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("MCP 服务器 %s 工具发现失败: %w", s.Name, err)
	}

	sess := &Session{ServerName: s.Name, cli: cli}
	sess.Tools = make([]tool.BaseTool, 0, len(baseTools))
	for _, t := range baseTools {
		info, err := t.Info(ctx)
		if err != nil || info == nil || info.Name == "" {
			sess.Skipped++
			continue
		}
		sess.Tools = append(sess.Tools, &mcpTool{
			serverName:   s.Name,
			inner:        t,
			originalName: info.Name,
		})
	}
	return sess, nil
}

// Close 关闭 MCP 客户端连接；cli 为 nil 时安全返回。
func (sess *Session) Close() error {
	if sess == nil || sess.cli == nil {
		return nil
	}
	return sess.cli.Close()
}

// mcpTool 改名包装器：把 MCP 工具重命名为 mcp_{服务器名}_{原始工具名}，
// 避免与内置工具或跨服务器工具重名冲突；执行委托内层工具。
// 实现 ActionTextProvider，使父包经 tools.WrapWithError 包装后仍能断言到友好动作文案。
type mcpTool struct {
	serverName   string
	inner        tool.BaseTool
	originalName string
	// cachedInfo 首次 Info 构建后的改名工具信息缓存。工具定义在本轮会话内不变，
	// 缓存可消除 agent.go 装配（取名、建索引）与 eino 框架多次调用 Info 时的重复 JSON deepcopy。
	cachedInfo *schema.ToolInfo
}

var _ tool.InvokableTool = (*mcpTool)(nil)
var _ tools.ActionTextProvider = (*mcpTool)(nil)

// Info 返回改名后的工具信息：深拷贝内层 ToolInfo（避免修改 eino 框架共享对象），
// Name 改为 mcp_{serverName}_{originalName}。首次调用后缓存结果，后续直接返回
// 浅拷贝副本（复制标量与 Name，指针字段共享只读语义），调用方修改返回值不影响缓存。
func (m *mcpTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	if m.cachedInfo != nil {
		copied := *m.cachedInfo
		return &copied, nil
	}
	info, err := m.inner.Info(ctx)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	copied, err := deepCopyToolInfo(info)
	if err != nil {
		return nil, err
	}
	copied.Name = "mcp_" + m.serverName + "_" + m.originalName
	m.cachedInfo = copied
	// 首次调用同样返回浅拷贝副本，保证调用方修改返回值不影响缓存
	firstCopy := *copied
	return &firstCopy, nil
}

// InvokableRun 委托内层工具执行；GetTools 返回的工具均实现 InvokableTool。
func (m *mcpTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	invokable, ok := m.inner.(tool.InvokableTool)
	if !ok {
		return "", fmt.Errorf("MCP 工具 %s 不支持执行（未实现 InvokableRun）", m.originalName)
	}
	return invokable.InvokableRun(ctx, argumentsInJSON, opts...)
}

// ActionText 提供工具开始执行时的动作文案（前端状态条展示"调用 {服务器} 的 {工具}"）。
func (m *mcpTool) ActionText(argumentsInJSON string) string {
	return fmt.Sprintf("调用 %s 的 %s", m.serverName, m.originalName)
}

// deepCopyToolInfo 通过 JSON 序列化重建 ToolInfo（其自定义 MarshalJSON/UnmarshalJSON
// 会完整重建 Desc / Params / ParamsOneOf / Extra），避免多个包装器共享同一对象导致改名互相污染。
func deepCopyToolInfo(info *schema.ToolInfo) (*schema.ToolInfo, error) {
	b, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	var copied schema.ToolInfo
	if err := json.Unmarshal(b, &copied); err != nil {
		return nil, err
	}
	return &copied, nil
}
