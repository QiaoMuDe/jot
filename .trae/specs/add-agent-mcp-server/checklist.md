# Checklist

## 依赖与构建
- [x] eino-ext/components/tool/mcp 依赖已引入，`go build ./...` 通过，eino 核心版本兼容（现有 openai 模型组件与 adk.ChatModelAgent 行为无回归）
- [x] `go vet ./...` 无新增告警

## 配置解析（internal/mcpserver/config.go）
- [x] `mcp-servers.json` 示例文件已放置于项目根目录，格式与 spec.md 定义一致
- [x] `Load` 正确解析合法配置（stdio / sse / http 三传输）
- [x] 非法配置（transport 非法 / name 为空 / stdio 缺 command / url 传输缺 url）被正确拒绝或跳过
- [x] 文件缺失时返回明确错误而非 panic

## 连接层（internal/mcpserver/client.go）
- [x] stdio / sse / http 三种传输客户端按配置正确构造
- [x] Start + Initialize 握手流程完整，ClientInfo 标识 jot
- [x] 连接/握手失败返回含服务器名的包装错误，受 ctx 取消控制

## 工具发现与包装（internal/mcpserver/tools.go）
- [x] `OpenSession` 能拉取 MCP 服务器上的全部工具
- [x] 工具名统一为 `mcp_{服务器名}_{工具名}` 前缀，无与内置工具冲突
- [x] 工具实现 ActionTextProvider，文案为"调用 {服务器名} 的 {工具名}"
- [x] 工具可经 tools.WrapWithError 包装（失败回填模型 + tool_error 事件）

## Agent 接入（internal/agent/agent.go）
- [x] Deps 新增 MCPServerConfigPath 字段，路径为空时回退默认文件名
- [x] Run() 仅装配 enabled=true 的服务器，MCP 工具并入 toolList 且参与 toolByName 索引
- [x] 单服务器连接/发现失败被跳过并记录日志，不影响其他服务器与整体运行
- [x] 无配置文件 / 无启用服务器时 Agent 行为与现状完全一致（零回归）

## 测试与边界
- [x] config_test.go 覆盖合法/非法/缺失文件三类用例
- [x] tools_test.go 用内存 MCP 服务器（NewMCPServer + NewSSEServer）验证握手 → 发现 → 前缀命名 → 真实调用全链路
- [x] 未改动数据库模型 / app.go 绑定 / 前端代码（测试阶段边界确认）
