package agent

// 本文件实现 Agent 前置意图感知（阶段 1 规则层）：
// 在进入 ReAct 循环前对当前用户输入做轻量规则意图分类，产出意图标签、
// 针对性强化提示（注入 Instruction）与高置信工具裁剪（并入禁用集合）。
// 定位为"感知与信号强化"层：不改动循环逻辑，最终决策权仍在循环内模型；
// 仅对高置信的纯时间查询做工具裁剪（闲聊判断不可靠，不裁剪），其余仅注入提示，避免误伤。

import (
	"strings"

	"jot/internal/agent/tools"
)

// Intent 意图类型。
type Intent int

const (
	IntentUnknown   Intent = iota // 未识别
	IntentChitchat                // 闲聊/问候（无任务，裁剪全部内置工具）
	IntentTime                    // 纯时间/日期查询（裁剪其余内置工具）
	IntentSearch                  // 联网搜索需求（注入提示）
	IntentStats                   // 数据统计需求（注入提示）
	IntentNoteWrite               // 笔记/待办/标签写操作（注入 confirm 强化提示）
	IntentVague                   // 需求模糊（注入 ask_user 强制反问提示）
	IntentQuery                   // 本地知识/笔记查询（默认，无动作）
)

// String 返回意图的英文标识（日志用）。
func (i Intent) String() string {
	switch i {
	case IntentChitchat:
		return "chitchat"
	case IntentTime:
		return "time"
	case IntentSearch:
		return "search"
	case IntentStats:
		return "stats"
	case IntentNoteWrite:
		return "note_write"
	case IntentVague:
		return "vague"
	case IntentQuery:
		return "query"
	default:
		return "unknown"
	}
}

// IntentResult 意图感知结果。
type IntentResult struct {
	Intent  Intent   // 意图类型
	Disable []string // 高置信裁剪的工具名列表（并入禁用集合，可 nil）
	Prompt  string   // 注入 Instruction 的针对性强化提示（可空串）
}

// ClassifyIntent 对用户输入做规则意图分类（阶段 1，纯本地规则、零 API 成本）。
// 分类顺序：纯时间 → 纯闲聊 → 写操作 → 模糊 → 统计 → 搜索 → 默认。
// 仅纯时间查询做工具裁剪（短句 + 无任务词双条件防误伤）；闲聊因难以与
// "应声 + 正题"区分不做裁剪，其余只注入提示。
func ClassifyIntent(userText string) IntentResult {
	text := strings.TrimSpace(userText)
	if text == "" {
		return IntentResult{Intent: IntentUnknown}
	}

	// 1. 纯时间/日期查询：高置信才裁剪（保留 get_current_time）
	if isPureTimeQuery(text) {
		return IntentResult{Intent: IntentTime, Disable: exceptToolNames("get_current_time")}
	}

	// 2. 纯闲聊：闲聊难与"应声 + 正题"区分（如"好的，把今天的安排发我一下"），
	//    为避免误裁剪全部工具导致功能受限，仅标记意图（供日志观测），不做工具裁剪
	if isPureChitchat(text) {
		return IntentResult{Intent: IntentChitchat}
	}

	// 3. 写操作：注入 confirm 强化提示（不裁剪）
	if isWriteOp(text) {
		return IntentResult{
			Intent: IntentNoteWrite,
			Prompt: "【本请求意图识别】本条请求疑似笔记/待办/标签等写操作，执行前必须先向用户确认修改意图（确认流程见上文写操作规范），不得在未获用户同意时执行。",
		}
	}

	// 4. 模糊需求：注入 ask_user 强制反问提示（不裁剪）
	if isVague(text) {
		return IntentResult{
			Intent: IntentVague,
			Prompt: "【本请求意图识别】本条请求需求模糊、信息不完整，必须先调用 ask_user 工具向用户反问澄清（见上文强制调用规范），严禁猜测后直接执行。",
		}
	}

	// 5. 统计需求：注入 get_stats 提示（不裁剪）
	if isStatsQuery(text) {
		return IntentResult{
			Intent: IntentStats,
			Prompt: "【本请求意图识别】本条请求为数据统计类需求，应优先调用 get_stats 工具获取统计数据，再基于数据回答。",
		}
	}

	// 6. 联网搜索需求：注入 MCP 搜索工具提示（不裁剪）
	if isSearchQuery(text) {
		return IntentResult{
			Intent: IntentSearch,
			Prompt: "【本请求意图识别】本条请求为联网搜索类需求，可直接调用 MCP 服务器提供的搜索工具获取实时信息；是否先检索本地笔记由【本地知识优先】规范决定。",
		}
	}

	// 其余归为本地知识/笔记查询或未知
	return IntentResult{Intent: IntentQuery}
}

// ── 规则辅助函数（关键词表集中维护，便于调整） ──

// taskWords 任务词：命中说明请求带有实际任务，不能算"纯"闲聊/时间查询。
var taskWords = []string{
	"查", "搜", "创建", "新建", "新增", "修改", "编辑", "删除", "删掉", "移动",
	"置顶", "统计", "整理", "写", "笔记", "待办", "标签", "笔记本", "帮", "改",
}

// hasTaskWord 判断文本是否含任务词。
func hasTaskWord(s string) bool {
	for _, k := range taskWords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// timeWords 时间/日期查询词。
var timeWords = []string{
	"现在几点", "几点了", "现在时间", "当前时间", "现在是几点",
	"今天几号", "今天是几号", "今天星期几", "今天是星期几", "现在日期", "今天日期",
}

// hasTimeWord 判断文本是否含时间查询词。
func hasTimeWord(s string) bool {
	for _, k := range timeWords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// isPureTimeQuery 纯时间查询：命中时间词 + 整句短（≤15 rune）+ 无任务词。
func isPureTimeQuery(s string) bool {
	if len([]rune(s)) > 15 {
		return false
	}
	return hasTimeWord(s) && !hasTaskWord(s)
}

// chitchatWords 问候/感谢/道别等纯闲聊词。
var chitchatWords = []string{
	"你好", "您好", "嗨", "哈喽", "hello", "hi", "早上好", "中午好", "下午好", "晚上好",
	"谢谢", "感谢", "辛苦了", "再见", "拜拜", "晚安", "好的", "嗯", "嗯嗯", "ok",
	"知道了", "明白了",
}

// isPureChitchat 纯闲聊：命中闲聊词 + 整句短（≤20 rune）+ 无任务词。
func isPureChitchat(s string) bool {
	if len([]rune(s)) > 20 {
		return false
	}
	for _, k := range chitchatWords {
		if strings.Contains(s, k) {
			return !hasTaskWord(s)
		}
	}
	return false
}

// writeVerbs 写操作动作词。
var writeVerbs = []string{
	"创建", "新建", "新增", "添加", "修改", "编辑", "更新", "改一下", "改下",
	"删除", "删掉", "移动", "移到", "置顶", "取消置顶", "打标签", "加标签",
	"移除标签", "重命名", "改名",
}

// writeObjs 写操作对象词。
var writeObjs = []string{
	"笔记", "待办", "任务", "todo", "笔记本", "标签",
}

// isWriteOp 写操作意图：动作词 × 对象词任一组合命中。
func isWriteOp(s string) bool {
	for _, v := range writeVerbs {
		if !strings.Contains(s, v) {
			continue
		}
		for _, o := range writeObjs {
			if strings.Contains(s, o) {
				return true
			}
		}
	}
	return false
}

// vagueWords 模糊需求词。
var vagueWords = []string{
	"随便", "都行", "都可以", "你看着办", "你看情况", "你来定", "你决定",
	"整理一下", "简单弄一下", "大概", "差不多", "帮我看看",
}

// isVague 模糊需求意图。
func isVague(s string) bool {
	for _, k := range vagueWords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// statsWords 统计需求词。
var statsWords = []string{
	"统计", "汇总", "概览", "一共", "总共", "多少篇", "几条", "几个", "数量",
}

// isStatsQuery 统计需求意图。
func isStatsQuery(s string) bool {
	for _, k := range statsWords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// localObjWords 本地对象词：命中说明是本地知识检索而非联网搜索。
var localObjWords = []string{
	"笔记", "待办", "标签", "笔记本",
}

// hasLocalObj 判断文本是否含本地对象词。
func hasLocalObj(s string) bool {
	for _, k := range localObjWords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// searchWords 联网搜索词。
var searchWords = []string{
	"搜索", "搜一下", "查一下", "查询", "查查", "搜索一下", "联网", "网上",
	"最新", "新闻", "资讯", "天气", "实时",
}

// isSearchQuery 联网搜索意图：先排除时间词与本地对象词，再匹配搜索词。
func isSearchQuery(s string) bool {
	if hasTimeWord(s) || hasLocalObj(s) {
		return false
	}
	for _, k := range searchWords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// builtinToolNames 返回全部内置工具名（与 tools.BuiltinTools() 一致），
// 供裁剪列表生成，避免工具名硬编码漂移。
func builtinToolNames() []string {
	metas := tools.BuiltinTools()
	names := make([]string, 0, len(metas))
	for _, m := range metas {
		names = append(names, m.Name)
	}
	return names
}

// exceptToolNames 返回除 keep 外的全部内置工具名（时间查询时保留 get_current_time）。
func exceptToolNames(keep ...string) []string {
	keepSet := make(map[string]bool, len(keep))
	for _, k := range keep {
		keepSet[k] = true
	}
	all := builtinToolNames()
	out := make([]string, 0, len(all))
	for _, n := range all {
		if !keepSet[n] {
			out = append(out, n)
		}
	}
	return out
}
