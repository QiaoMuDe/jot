package tools

// 本文件实现 write_file 工具：在 AI 助手工作目录（~/.jot/workspace）内创建/
// 覆盖/追加写入文件，自动创建父目录。经 fsToolBase 做路径边界校验（仅允许操作
// 工作目录内的文件，越权一律拒绝）；覆盖已存在文件（append=false）时由
// Context.Approver 审批钩子接管（critical=false，常规审批可取消）。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// writeFileTool 创建/覆盖/追加写入工作目录内文件的工具。
type writeFileTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*writeFileTool)(nil)
var _ ActionTextProvider = (*writeFileTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *writeFileTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "写入文件"
	}
	if p := strings.TrimSpace(args.Path); p != "" {
		return "写入文件：" + TruncateRunes(p, 30)
	}
	return "写入文件"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *writeFileTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "write_file",
		Desc: "在工作目录内创建/覆盖/追加写入文件。内容超长时写为 multiples 个文件需自行组织，单文件写入按顺序。append=false 时覆盖已存在文件（会被审批机制检查）。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "要写入的文件路径；相对 ~/.jot/workspace 的路径、~/.jot/workspace/ 开头路径或其内绝对路径均可",
				Required: true,
			},
			"content": {
				Type:     schema.String,
				Desc:     "要写入的文件内容",
				Required: true,
			},
			"append": {
				Type:     schema.Boolean,
				Desc:     "true=追加到文件末尾，false=覆盖已存在文件（缺省 false）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行写入：边界校验 → 内容长度校验 → 覆盖场景审批检查 →
// 自动创建父目录后写入。
func (t *writeFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Append  bool   `json:"append"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 write_file 参数失败: %w", err)
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "", errors.New("write_file 参数缺少 path")
	}
	if err := validateTextLen("path", path, maxToolShortText); err != nil {
		return "", err
	}
	if err := validateTextLen("content", args.Content, maxToolLongText); err != nil {
		return "", err
	}
	fullPath, err := t.resolvePath(path)
	if err != nil {
		return "", err
	}

	// 审批检查点：覆盖已存在文件（append=false 且文件已存在）时需取得用户批准；
	// 未被拒绝（Approver 未注入/批准）则继续执行
	if !args.Append {
		if _, statErr := os.Lstat(fullPath); statErr == nil {
			if err := t.requestApproval(ctx, "write_file", fmt.Sprintf("覆盖文件：%s", path), false); err != nil {
				return "", err
			}
		}
	}

	// 自动创建父目录（幂等）
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", fmt.Errorf("创建父目录失败: %w", err)
	}

	if args.Append {
		f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return "", fmt.Errorf("打开文件追加失败: %w", err)
		}
		_, wErr := f.WriteString(args.Content)
		closeErr := f.Close()
		if wErr != nil {
			return "", fmt.Errorf("追加写入失败: %w", wErr)
		}
		if closeErr != nil {
			return "", fmt.Errorf("关闭文件失败: %w", closeErr)
		}
		return fmt.Sprintf("已追加写入文件：%s", path), nil
	}

	if err := os.WriteFile(fullPath, []byte(args.Content), 0o644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}
	return fmt.Sprintf("已写入文件：%s", path), nil
}

// NewWriteFile 创建写入文件工具。
func NewWriteFile(ctx *Context) tool.InvokableTool {
	return &writeFileTool{fsToolBase: fsToolBase{ctx: ctx}}
}
