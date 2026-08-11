package store

import (
	"context"
	"fmt"
	"sort"
)

// bruteStore 是纯 Go 暴力检索实现（sqlite-vec 扩展不可用时的回退方案）。
type bruteStore struct {
	baseStore
}

// Search 加载全部 chunks，逐条解码向量后用余弦相似度计算距离（距离 = 1 - 相似度），
// 排序后返回 topK 命中。
func (s *bruteStore) Search(ctx context.Context, queryVec []float32, topK int) ([]SearchHit, error) {
	var chunks []Chunk
	if err := s.db.WithContext(ctx).Where("model = ?", s.embedModel).Find(&chunks).Error; err != nil {
		return nil, fmt.Errorf("加载 chunks 失败: %w", err)
	}

	type scored struct {
		hit  SearchHit
		dist float64
	}
	candidates := make([]scored, 0, len(chunks))
	for _, c := range chunks {
		vec, err := BlobToFloat32(c.Embedding)
		if err != nil {
			return nil, fmt.Errorf("解码块 %d 向量失败: %w", c.ID, err)
		}
		sim := CosineSimilarity(queryVec, vec)
		candidates = append(candidates, scored{
			hit: SearchHit{
				DocID:      c.DocID,
				ChunkIndex: c.ChunkIndex,
				Text:       c.Text,
			},
			dist: 1 - sim, // 距离 = 1 - 相似度，与 sqlite-vec 的 cosine distance 语义一致
		})
	}

	// 按距离升序排序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].dist < candidates[j].dist
	})

	if topK > len(candidates) {
		topK = len(candidates)
	}
	candidates = candidates[:topK]

	// 批量回填文档名
	ids := make([]uint, 0, len(candidates))
	for _, cd := range candidates {
		ids = append(ids, cd.hit.DocID)
	}
	nameByID, err := s.docNameMap(ctx, ids)
	if err != nil {
		return nil, err
	}

	hits := make([]SearchHit, 0, len(candidates))
	for _, cd := range candidates {
		cd.hit.DocName = nameByID[cd.hit.DocID]
		cd.hit.Distance = cd.dist
		hits = append(hits, cd.hit)
	}
	return hits, nil
}

// Status 返回实现名与统计信息。
func (s *bruteStore) Status(ctx context.Context) (string, int, int, error) {
	return s.baseStore.Status(ctx, "pure-go-brute")
}
