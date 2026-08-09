package services

import (
	"reflect"
	"testing"

	"jot/internal/models"
)

// TestFilterHighFreqTokens 验证高频词过滤：命中数超过 max(总块数/10, 100) 的 token 被剔除
func TestFilterHighFreqTokens(t *testing.T) {
	// 场景1：模拟"小时数据的代码是多少"分词结果，数据为全库高频词被滤，小时未超阈值保留
	tokens := []string{"数据", "小时", "代码", "2061"}
	counts := []int{7459, 800, 300, 84}
	got := filterHighFreqTokens(tokens, counts, 8000)
	want := []string{"小时", "代码", "2061"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("场景1 不符：期望 %v，实际 %v", want, got)
	}

	// 场景2：总块数少时阈值下限兜底（max(total/10, 100)=100），低频 token 全部保留
	tokens2 := []string{"数据", "代码"}
	got2 := filterHighFreqTokens(tokens2, []int{60, 10}, 300)
	want2 := []string{"数据", "代码"}
	if !reflect.DeepEqual(got2, want2) {
		t.Errorf("场景2 不符：期望 %v，实际 %v", want2, got2)
	}

	// 场景3：全部超阈值时返回空（关键词路不贡献）
	got3 := filterHighFreqTokens([]string{"数据", "系统"}, []int{5000, 3000}, 8000)
	if len(got3) != 0 {
		t.Errorf("场景3 应返回空，实际 %v", got3)
	}
}

// TestRankKwHits 验证候选块排序：按命中 token 数降序，同分按块 id 升序，截断到 limit
func TestRankKwHits(t *testing.T) {
	tokens := []string{"小时", "数据", "代码"}
	hits := []models.NoteVector{
		{ID: 3, ChunkText: "代码"},     // score 1
		{ID: 1, ChunkText: "小时数据代码"}, // score 3
		{ID: 2, ChunkText: "数据"},     // score 1
		{ID: 4, ChunkText: "小时数据代码"}, // score 3
	}

	// 场景1+2：多词命中排前，同分（score=3 的 ID 1、4 与 score=1 的 ID 2、3）按 id 升序
	got := rankKwHits(hits, tokens, 10)
	wantIDs := []uint{1, 4, 2, 3}
	if len(got) != len(wantIDs) {
		t.Fatalf("排序结果数量不符：期望 %d，实际 %d", len(wantIDs), len(got))
	}
	for i, id := range wantIDs {
		if got[i].ID != id {
			t.Errorf("第 %d 位不符：期望 ID %d，实际 ID %d", i, id, got[i].ID)
		}
	}

	// 场景3：候选多于 limit 时截断到 limit
	got2 := rankKwHits(hits, tokens, 2)
	if len(got2) != 2 {
		t.Errorf("截断结果数量不符：期望 2，实际 %d", len(got2))
	}
	if got2[0].ID != 1 || got2[1].ID != 4 {
		t.Errorf("截断后应取最相关两个（ID 1、4），实际 %d、%d", got2[0].ID, got2[1].ID)
	}

	// limit<=0 返回 nil
	if rankKwHits(hits, tokens, 0) != nil {
		t.Error("limit<=0 应返回 nil")
	}
}
