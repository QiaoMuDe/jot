package main

import (
	"reflect"
	"strings"
	"testing"

	"jot/internal/services"
)

// ctxMsg 构造一条 user/assistant 消息：内容为 n 个 ASCII 字符，token 数 = ceil(n/4)。
func ctxMsg(id uint, role string, n int) services.Message {
	return services.Message{ID: id, Role: role, Content: strings.Repeat("a", n)}
}

// selectAIContextTail 从 nonSystem 消息中按摘要边界与预算选取 tail，
// 返回 (tail, tailStart, tailTokens, boundaryPos)。该函数是 truncateAIMessages 与
// GetAIContextUsage 共用的口径，测试聚焦：空输入、预算内、超预算、最后一条必留、
// 边界定位/缺失、轮次对齐、边界+预算组合。
func TestSelectAIContextTail(t *testing.T) {
	tests := []struct {
		name          string
		msgs          []services.Message
		boundaryMsgID uint
		budget        int
		wantTail      []services.Message
		wantTailStart int
		wantTokens    int
		wantBoundary  int
	}{
		{
			name: "空消息",
		},
		{
			name:          "全部在预算内（无边界）",
			msgs:          []services.Message{ctxMsg(1, "user", 8), ctxMsg(2, "assistant", 8), ctxMsg(3, "user", 8)},
			budget:        100,
			wantTail:      []services.Message{ctxMsg(1, "user", 8), ctxMsg(2, "assistant", 8), ctxMsg(3, "user", 8)},
			wantTailStart: 0,
			wantTokens:    6,
		},
		{
			name:          "超预算时仅保留尾部",
			msgs:          []services.Message{ctxMsg(1, "user", 40), ctxMsg(2, "user", 40), ctxMsg(3, "user", 40), ctxMsg(4, "user", 40), ctxMsg(5, "user", 40)},
			budget:        30, // 每条 10 token，倒序累计：m5+m4+m3=30，m2 加入即超
			wantTail:      []services.Message{ctxMsg(3, "user", 40), ctxMsg(4, "user", 40), ctxMsg(5, "user", 40)},
			wantTailStart: 2,
			wantTokens:    30,
		},
		{
			name:          "单条超预算仍保留最后一条",
			msgs:          []services.Message{ctxMsg(1, "user", 1000)},
			budget:        100, // 单条 250 token > 预算，最后一条始终保留
			wantTail:      []services.Message{ctxMsg(1, "user", 1000)},
			wantTailStart: 0,
			wantTokens:    250,
		},
		{
			name:          "摘要边界之后才开始选取",
			msgs:          []services.Message{ctxMsg(1, "user", 8), ctxMsg(2, "user", 8), ctxMsg(3, "user", 8), ctxMsg(4, "user", 8), ctxMsg(5, "user", 8), ctxMsg(6, "user", 8)},
			boundaryMsgID: 3, // 边界=m3（下标 2），边界后从下标 3 起，共 3 条
			budget:        100,
			wantTail:      []services.Message{ctxMsg(4, "user", 8), ctxMsg(5, "user", 8), ctxMsg(6, "user", 8)},
			wantTailStart: 3,
			wantTokens:    6,
			wantBoundary:  3,
		},
		{
			name:          "边界消息被删除时回退全量",
			msgs:          []services.Message{ctxMsg(1, "user", 8), ctxMsg(2, "user", 8), ctxMsg(3, "user", 8)},
			boundaryMsgID: 999, // 不在消息中，回退 0 走全量路径
			budget:        100,
			wantTail:      []services.Message{ctxMsg(1, "user", 8), ctxMsg(2, "user", 8), ctxMsg(3, "user", 8)},
			wantTailStart: 0,
			wantTokens:    6,
		},
		{
			name: "tail 首条对齐为 user 消息",
			msgs: []services.Message{
				ctxMsg(1, "user", 40),
				ctxMsg(2, "assistant", 40),
				ctxMsg(3, "user", 40),
				ctxMsg(4, "assistant", 40),
			},
			budget:        30, // 倒序：m4+m3+m2=30，m1 超预算被丢弃；对齐丢弃首条 assistant(m2)
			wantTail:      []services.Message{ctxMsg(3, "user", 40), ctxMsg(4, "assistant", 40)},
			wantTailStart: 2,
			wantTokens:    20,
		},
		{
			name:          "边界与预算组合",
			msgs:          []services.Message{ctxMsg(1, "user", 8), ctxMsg(2, "user", 8), ctxMsg(3, "user", 8), ctxMsg(4, "user", 8), ctxMsg(5, "user", 8), ctxMsg(6, "user", 8)},
			boundaryMsgID: 2, // 边界=m2（下标 1），边界后为 m3..m6 共 8 token
			budget:        4, // 预算 4 token：仅保留尾部 m5+m6
			wantTail:      []services.Message{ctxMsg(5, "user", 8), ctxMsg(6, "user", 8)},
			wantTailStart: 4,
			wantTokens:    4,
			wantBoundary:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tail, tailStart, tailTokens, boundaryPos := selectAIContextTail(tt.msgs, tt.boundaryMsgID, tt.budget)
			if !reflect.DeepEqual(tail, tt.wantTail) {
				t.Errorf("tail 不匹配：got %v, want %v", tail, tt.wantTail)
			}
			if tailStart != tt.wantTailStart {
				t.Errorf("tailStart 不匹配：got %d, want %d", tailStart, tt.wantTailStart)
			}
			if tailTokens != tt.wantTokens {
				t.Errorf("tailTokens 不匹配：got %d, want %d", tailTokens, tt.wantTokens)
			}
			if boundaryPos != tt.wantBoundary {
				t.Errorf("boundaryPos 不匹配：got %d, want %d", boundaryPos, tt.wantBoundary)
			}
		})
	}
}
