package tools

// 本文件测试 summarize_text 工具：参数校验（缺 text / 超长）、取消路径。
// LLM 真实调用（AIService.CallAI）依赖外部服务，离线测试不覆盖，
// 降级返回原文与真实摘要输出留手动验证。

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestSummarizeTextArgValidation 覆盖参数校验：缺 text、超长 text、超长 instructions。
// 校验失败返回错误（经 WrapWithError 回填模型），不触发 LLM 调用。
func TestSummarizeTextArgValidation(t *testing.T) {
	tool := NewSummarizeText(nil) // ai 为 nil 即可：参数校验在调用 CallAI 之前

	// 缺 text
	if _, err := tool.InvokableRun(context.Background(), `{}`); err == nil {
		t.Fatal("缺 text 应返回错误")
	}
	if _, err := tool.InvokableRun(context.Background(), `{"text": "  "}`); err == nil {
		t.Fatal("空 text 应返回错误")
	}

	// 超长 text（> maxToolLongText rune）
	longText := strings.Repeat("长", maxToolLongText+1)
	args := `{"text": "` + longText + `"}`
	if _, err := tool.InvokableRun(context.Background(), args); err == nil {
		t.Fatal("超长 text 应返回错误")
	}

	// 超长 instructions（> maxToolShortText rune）
	longInstr := strings.Repeat("要", maxToolShortText+1)
	args = `{"text": "正文", "instructions": "` + longInstr + `"}`
	if _, err := tool.InvokableRun(context.Background(), args); err == nil {
		t.Fatal("超长 instructions 应返回错误")
	}
}

// TestSummarizeTextCancel 覆盖取消路径：ctx 已取消时，参数校验通过后调用
// CallAI 会因 ctx 取消失败，工具应返回 ctx.Err()（终止循环）而非降级原文。
func TestSummarizeTextCancel(t *testing.T) {
	tool := NewSummarizeText(nil) // ai 为 nil：若走到 CallAI 会 panic，正好验证提前返回

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := tool.InvokableRun(ctx, `{"text": "需要摘要的正文"}`)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ctx 取消应返回 context.Canceled，got %v", err)
	}
}

// TestSummarizeTextInfo 校验工具元信息与 ActionText。
func TestSummarizeTextInfo(t *testing.T) {
	tool := NewSummarizeText(nil)
	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info err: %v", err)
	}
	if info.Name != "summarize_text" {
		t.Fatalf("工具名应为 summarize_text，got %q", info.Name)
	}
	if provider, ok := tool.(ActionTextProvider); ok {
		if got := provider.ActionText("{}"); got != "摘要长文本" {
			t.Fatalf("ActionText 应为固定文案，got %q", got)
		}
	} else {
		t.Fatal("summarize_text 应实现 ActionTextProvider")
	}
}
