package store

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"testing"

	"vec-poc/internal/chunk"
)

// mockEmbedder 生成确定性伪向量（不依赖外部 Ollama 服务）：
// 对文本中的字符做 hash 累加得到向量，相同文本得到相同向量，共享字符多的文本相似度更高。
func mockEmbedder(texts []string) ([][]float32, error) {
	const dim = 16
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v := make([]float32, dim)
		for _, r := range []rune(t) {
			v[int(r)%dim]++
		}
		// 归一化，让余弦相似度数值稳定
		var norm float64
		for _, f := range v {
			norm += float64(f) * float64(f)
		}
		if norm > 0 {
			n := float32(math.Sqrt(norm))
			for j := range v {
				v[j] /= n
			}
		}
		out[i] = v
	}
	return out, nil
}

// openTestStore 在临时目录打开独立 DB，按 useVec 选择实现。
func openTestStore(t *testing.T, useVec bool) VectorStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "vec-it.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB 失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return NewStore(db, "mock-embed", useVec)
}

// TestStoreAddSearchBothImpls 验证两个实现的 添加→检索 全链路：
// 相同文本召回距离≈0，相关文本排序优先于不相关文本。
func TestStoreAddSearchBothImpls(t *testing.T) {
	for _, tc := range []struct {
		name   string
		useVec bool
	}{
		{"sqlite-vec", true},
		{"pure-go-brute", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := openTestStore(t, tc.useVec)
			ctx := context.Background()

			// 添加两个主题文档：相关（含"部署 步骤"）与不相关
			relText := "## 部署\n部署步骤：先配置环境变量，再启动服务，最后验证端口。部署需要耐心。"
			irrelText := "## 菜谱\n红烧肉的做法：五花肉切块，焯水，加酱油小火炖四十分钟。"
			relChunks := chunk.ChunkDefault(relText)
			irrelChunks := chunk.ChunkDefault(irrelText)
			n1, err := st.AddDocument(ctx, "rel.md", "rel.md", relText, relChunks, mockEmbedder)
			if err != nil {
				t.Fatalf("AddDocument(rel) 失败: %v", err)
			}
			n2, err := st.AddDocument(ctx, "irrel.md", "irrel.md", irrelText, irrelChunks, mockEmbedder)
			if err != nil {
				t.Fatalf("AddDocument(irrel) 失败: %v", err)
			}
			if n1 == 0 || n2 == 0 {
				t.Fatalf("块数异常: rel=%d irrel=%d", n1, n2)
			}

			// 用"部署"相关文本查询，期望命中 rel.md 且距离明显小于 irrel
			qVec, err := mockEmbedder([]string{"部署步骤：启动服务，验证端口"})
			if err != nil {
				t.Fatal(err)
			}
			hits, err := st.Search(ctx, qVec[0], 5)
			if err != nil {
				t.Fatalf("Search 失败: %v", err)
			}
			if len(hits) == 0 {
				t.Fatal("Search 未返回任何命中")
			}
			if hits[0].DocName != "rel.md" {
				t.Fatalf("期望首命中 rel.md，实际 %q (距离 %.4f)", hits[0].DocName, hits[0].Distance)
			}
			// 找到 irrel 的命中并断言距离更大
			for _, h := range hits {
				if h.DocName == "irrel.md" && h.Distance <= hits[0].Distance {
					t.Fatalf("不相关文档距离(%.4f)不应小于等于相关文档(%.4f)", h.Distance, hits[0].Distance)
				}
			}

			// 相同文本自查询：距离应接近 0
			selfVec, err := mockEmbedder([]string{relText})
			if err != nil {
				t.Fatal(err)
			}
			selfHits, err := st.Search(ctx, selfVec[0], 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(selfHits) == 0 || selfHits[0].Distance > 1e-3 {
				t.Fatalf("自查询距离应≈0，实际 %+v", selfHits)
			}
		})
	}
}

// TestStoreRebuild 验证 Rebuild 清空并重建索引后统计正确。
func TestStoreRebuild(t *testing.T) {
	for _, tc := range []struct {
		name   string
		useVec bool
	}{
		{"sqlite-vec", true},
		{"pure-go-brute", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := openTestStore(t, tc.useVec)
			ctx := context.Background()

			content := "## A\n部署步骤：配置环境变量。\n\n## B\n启动服务并验证端口。"
			_, err := st.AddDocument(ctx, "doc.md", "doc.md", content, chunk.ChunkDefault(content), mockEmbedder)
			if err != nil {
				t.Fatal(err)
			}
			impl, docCount, chunkCount, err := st.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if docCount != 1 || chunkCount == 0 {
				t.Fatalf("重建前统计异常: impl=%s docs=%d chunks=%d", impl, docCount, chunkCount)
			}

			n, err := st.Rebuild(ctx, mockEmbedder)
			if err != nil {
				t.Fatalf("Rebuild 失败: %v", err)
			}
			if n == 0 {
				t.Fatal("Rebuild 后块数为 0，异常")
			}
			impl2, docCount2, chunkCount2, err := st.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if docCount2 != 1 || chunkCount2 != n {
				t.Fatalf("重建后统计异常: impl=%s docs=%d chunks=%d (期望 %d)", impl2, docCount2, chunkCount2, n)
			}
			fmt.Printf("%s: rebuild 后 chunks=%d\n", tc.name, n)
		})
	}
}

// TestProbeVecLoads 验证 sqlite-vec 扩展在 glebarez 驱动下可加载。
func TestProbeVecLoads(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "vec-probe.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()
	version, err := ProbeVec(db)
	if err != nil {
		t.Fatalf("sqlite-vec 探针失败: %v", err)
	}
	t.Logf("sqlite-vec version: %s", version)
}
