package tools

// 本文件实现 manage_tag 标签管理工具：模型在 ReAct 循环中调用它创建标签、列出标签或更新标签，
// 底层复用 services.TagService（GetByName / Create / GetAll / Update），不感知父包 agent 的事件循环细节。
// 一个工具通过 action 参数区分三个动作：
//   - create：创建标签（name 必填、trim 后非空；color 可选、格式 #RRGGBB、缺省 #3b82f6）；
//     名称唯一，同名标签存在时返回提示而非错误（避免触发 WrapWithError 的失败回填）；
//   - list：列出全部标签（created_at ASC，标签规模小且名称唯一，不分页）；
//   - update：更新标签（id 必填、正整数，来自列表中的 [数字] 编号；name 可选新名称、
//     color 可选新颜色，格式 #RRGGBB，至少提供其一；重命名时做排除自身的同名查重）。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"

	"jot/internal/services"

	"gitee.com/MM-Q/fastlog"
)

// manageTagTool 工具结构体：声明所需依赖。
type manageTagTool struct {
	tag *services.TagService // 标签服务
	ctx *Context             // 日志依赖，仅用于 Debugw 记录
}

// 编译期断言：确保实现 tool.InvokableTool。
var _ tool.InvokableTool = (*manageTagTool)(nil)

// requestApproval 通过 Context.Approver 请求用户审批；拒绝时返回拒绝错误文本，
// 调用方不执行写操作。critical 表示是否为不可绕过的危险操作，透传给审批实现。
//
// ctx 为空（测试/独立调用的裸工具、无审批机制）时视为放行；但 ctx 非空而
// Approver 未注入（已装配工具上下文却缺审批器）属于生产装配遗漏——审批会被
// 静默跳过，导致写操作未确认即执行，因此此时直接报错而非放行。
func (m *manageTagTool) requestApproval(ctx context.Context, summary string, critical bool) error {
	if m.ctx == nil {
		return nil
	}
	if m.ctx.Approver == nil {
		return errors.New("manage_tag 需要审批确认，但当前未配置审批机制")
	}
	return m.ctx.Approver.RequestApproval(ctx, "manage_tag", summary, critical)
}

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// 按 action 参数映射动作文案，解析失败回退空串（前端回退"执行"）。
func (m *manageTagTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return ""
	}
	switch args.Action {
	case "create":
		return "创建标签"
	case "list":
		return "列出标签"
	case "update":
		return "更新标签"
	default:
		return "执行"
	}
}

// tagColorPattern 合法标签颜色格式：#RRGGBB。
var tagColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Info 返回工具元信息（模型据此决定是否调用）。
func (m *manageTagTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "manage_tag",
		Desc: "管理标签（标签是笔记的附属属性，用于分类筛选笔记）。当用户要求创建标签、查看标签列表或更新标签时调用。边界：manage_tag 管理标签本身（创建/重命名/改色），manage_note 的 add_tag/remove_tag 动作管理笔记与标签的关联；给笔记打标签用 manage_note（action=add_tag），管理标签定义用本工具。通过 action 参数区分动作：create=创建标签（需提供 name 标签名称，可提供 color 颜色，格式 #RRGGBB，缺省 #3b82f6）；list=列出全部标签（无额外参数）；update=更新标签（需提供 id 标签编号，可提供 name 新名称或 color 新颜色，至少其一）。返回标签列表或操作结果，列表中的编号 [数字] 可用于后续笔记标签操作或 update。写操作（update）将按当前审批模式弹出确认面板，用户批准后才执行。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "要执行的动作：create=创建标签；list=列出全部标签；update=更新标签",
				Enum:     []string{"create", "list", "update"},
				Required: true,
			},
			"id": {
				Type:     schema.Number,
				Desc:     "标签编号（正整数，列表中的 [数字] 即为 id），action=update 时必填",
				Required: false,
			},
			"name": {
				Type:     schema.String,
				Desc:     "标签名称，action=create 时必填，action=update 时可选（提供则重命名）",
				Required: false,
			},
			"color": {
				Type:     schema.String,
				Desc:     "标签颜色（格式 #RRGGBB），action=create 时可选（缺省 #3b82f6），action=update 时可选（提供则改色）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行工具：解析参数 → 校验 action → 按动作分发到 Create / GetAll / Update。
func (m *manageTagTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Action string  `json:"action"`
		ID     float64 `json:"id"`
		Name   string  `json:"name"`
		Color  string  `json:"color"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 manage_tag 参数失败: %w", err)
	}
	args.Action = strings.TrimSpace(args.Action)

	// 用户取消检查：父包事件循环随 ctx 终止，工具直接返回 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if m.ctx != nil && m.ctx.Logger != nil {
		m.ctx.Logger.Debugw("Agent manage_tag 调用",
			fastlog.String("action", args.Action),
			fastlog.Int("id", int(args.ID)),
			fastlog.String("name", args.Name),
			fastlog.String("color", args.Color))
	}

	switch args.Action {
	case "create":
		return m.createTag(ctx, args.Name, args.Color)
	case "list":
		return m.listTags(ctx)
	case "update":
		// 先参数校验，再审批，再执行：避免无效参数（id<=0、name/color 全空、color 非法）先弹出无意义审批窗
		if args.ID <= 0 {
			return "", errors.New("manage_tag 更新标签缺少有效的 id")
		}
		name := strings.TrimSpace(args.Name)
		color := strings.TrimSpace(args.Color)
		if name == "" && color == "" {
			return "", errors.New("manage_tag 更新标签需至少提供 name 或 color")
		}
		if color != "" && !tagColorPattern.MatchString(color) {
			return "", fmt.Errorf("manage_tag 参数非法 color: %s（应为 #RRGGBB 格式）", color)
		}
		// 写操作审批：参数校验通过后请求用户确认（create/list 豁免，按当前审批模式门控）
		if err := m.requestApproval(ctx, fmt.Sprintf("更新标签 #%d（名称/颜色）", int(args.ID)), false); err != nil {
			return "", err
		}
		return m.updateTag(ctx, args.ID, name, color)
	}
	return "", fmt.Errorf("manage_tag 未知 action: %s", args.Action)
}

// createTag 创建标签：name 必填；color 可选且校验 #RRGGBB；同名标签返回提示而非错误。
func (m *manageTagTool) createTag(ctx context.Context, name, color string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("manage_tag 参数缺少 name")
	}
	if err := validateTextLen("name", name, maxToolShortText); err != nil {
		return "", err
	}
	color = strings.TrimSpace(color)
	if color != "" && !tagColorPattern.MatchString(color) {
		return "", fmt.Errorf("manage_tag 参数非法 color: %s（应为 #RRGGBB 格式）", color)
	}

	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// 同名标签查重（名称唯一）
	if existing, err := m.tag.GetByName(name); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	} else {
		return fmt.Sprintf("标签「%s」已存在（编号 [%d]），无需重复创建", existing.Name, existing.ID), nil
	}

	tag, err := m.tag.Create(name, color)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("已创建标签「%s」（编号 [%d]）", tag.Name, tag.ID), nil
}

// listTags 列出全部标签：created_at ASC，标签规模小且名称唯一，不分页。
func (m *manageTagTool) listTags(ctx context.Context) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	tags, err := m.tag.GetAll()
	if err != nil {
		return "", err
	}
	if len(tags) == 0 {
		return "当前没有任何标签", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "当前标签列表（共 %d 个）：\n", len(tags))
	for i := range tags {
		t := tags[i]
		fmt.Fprintf(&b, "[%d] %s · 颜色 %s\n", t.ID, t.Name, t.Color)
	}
	return b.String(), nil
}

// updateTag 更新标签：id 必填、正整数；name/color 至少提供其一；
// color 非空时校验 #RRGGBB；重命名时做排除自身的同名查重（同名返回提示而非错误）。
func (m *manageTagTool) updateTag(ctx context.Context, id float64, name, color string) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_tag 更新标签缺少有效的 id")
	}
	name = strings.TrimSpace(name)
	color = strings.TrimSpace(color)
	if name == "" && color == "" {
		return "未提供任何更新字段：需要 name 新名称或 color 新颜色", nil
	}
	if name != "" {
		if err := validateTextLen("name", name, maxToolShortText); err != nil {
			return "", err
		}
	}
	if color != "" && !tagColorPattern.MatchString(color) {
		return "", fmt.Errorf("manage_tag 参数非法 color: %s（应为 #RRGGBB 格式）", color)
	}

	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// 重命名时查重（排除自身：同名且 ID 相同说明名称未变，不冲突）
	if name != "" {
		if existing, err := m.tag.GetByName(name); err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return "", err
			}
		} else if existing.ID != uint(id) {
			return fmt.Sprintf("标签「%s」已存在（编号 [%d]），无法重命名为同名标签", existing.Name, existing.ID), nil
		}
	}

	tag, err := m.tag.Update(uint(id), name, color)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("已更新标签「%s」（编号 [%d]）· 颜色 %s", tag.Name, tag.ID, tag.Color), nil
}

// NewManageTag 创建工具。构造器签名 = 工具的全部依赖。
func NewManageTag(tag *services.TagService, ctx *Context) tool.InvokableTool {
	return &manageTagTool{tag: tag, ctx: ctx}
}
