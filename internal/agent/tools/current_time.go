package tools

// 本文件实现 get_current_time 时间工具：模型在 ReAct 循环中调用它获取
// 当前日期、时间、星期、年份等，用于回答"今天几号""现在几点""今年是哪年"
// 或依赖当前时间背景的问题。无参数、无外部依赖（仅标准库 time），
// 每次调用读取本地时间（App 为本地桌面应用，本地时区即为用户时区）。

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// weekdays 中文星期名，按 time.Weekday 下标（0=星期日）对应。
var weekdays = [...]string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}

// currentTimeTool 当前时间工具。
type currentTimeTool struct{}

// 编译期断言：确保 currentTimeTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*currentTimeTool)(nil)

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (c *currentTimeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "get_current_time",
		Desc:        "获取当前日期与时间（年份、日期、星期、时分秒）。当用户询问现在几点、今天几号/星期几、今年是哪年，或问题需要以当前时间作为背景时调用。无参数。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

// InvokableRun 返回当前日期、时间、星期与年份。无参数、无失败路径。
func (c *currentTimeTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	now := time.Now()
	return fmt.Sprintf("当前日期：%s（%s）\n当前时间：%s\n当前年份：%d",
		now.Format("2006-01-02"),
		weekdays[now.Weekday()],
		now.Format("15:04:05"),
		now.Year()), nil
}

// NewGetCurrentTime 创建当前时间工具。
func NewGetCurrentTime() tool.InvokableTool {
	return &currentTimeTool{}
}
