package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"vec-poc/internal/chunk"
)

// ProgressFunc 是索引/重建过程中的进度回调（done 为已处理数，total 为总数，msg 为当前处理项描述）。
type ProgressFunc func(done, total int, msg string)

// baseStore 提供 vecStore 与 bruteStore 共用的存储逻辑。
type baseStore struct {
	db         *gorm.DB
	embedModel string
	progress   ProgressFunc // 文档级进度回调，可为 nil
}

// SetProgress 设置文档级进度回调（供 index/rebuild 等批量操作展示进度）。
func (s *baseStore) SetProgress(p ProgressFunc) {
	s.progress = p
}

// AddDocument 创建 Document 记录，对切块批量向量化后写入 Chunk 记录，返回块数。
func (s *baseStore) AddDocument(ctx context.Context, name, sourcePath, content string, chunks []string, embedder Embedder) (int, error) {
	doc := Document{
		Name:       name,
		SourcePath: sourcePath,
		Content:    content,
	}
	if err := s.db.WithContext(ctx).Create(&doc).Error; err != nil {
		return 0, fmt.Errorf("写入 Document 失败: %w", err)
	}
	return s.addChunks(ctx, doc.ID, chunks, embedder)
}

// addChunks 将切块批量向量化后写入 Chunk 记录（不创建 Document，供 AddDocument 与 Rebuild 共用）。
func (s *baseStore) addChunks(ctx context.Context, docID uint, chunks []string, embedder Embedder) (int, error) {
	// 切块为空则文档不产生任何 Chunk
	if len(chunks) == 0 {
		return 0, nil
	}

	// 批量向量化
	embeddings, err := embedder(chunks)
	if err != nil {
		return 0, fmt.Errorf("文档 %d 向量化失败: %w", docID, err)
	}
	if len(embeddings) != len(chunks) {
		return 0, fmt.Errorf("文档 %d 向量化数量不匹配: 期望 %d 实际 %d", docID, len(chunks), len(embeddings))
	}

	rows := make([]Chunk, 0, len(chunks))
	for i, text := range chunks {
		rows = append(rows, Chunk{
			DocID:      docID,
			ChunkIndex: i,
			Text:       text,
			Embedding:  Float32ToBlob(embeddings[i]),
			Dim:        len(embeddings[i]),
			Model:      s.embedModel,
		})
	}
	if err := s.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return 0, fmt.Errorf("写入 Chunk 失败: %w", err)
	}
	return len(rows), nil
}

// Rebuild 清空所有 chunks 后，对全部 documents 重新切块并向量化（不重复创建 Document），返回重建块数。
func (s *baseStore) Rebuild(ctx context.Context, embedder Embedder) (int, error) {
	if err := s.db.WithContext(ctx).Exec("DELETE FROM chunks").Error; err != nil {
		return 0, fmt.Errorf("清空 chunks 失败: %w", err)
	}

	var docs []Document
	if err := s.db.WithContext(ctx).Find(&docs).Error; err != nil {
		return 0, fmt.Errorf("读取文档列表失败: %w", err)
	}

	total := 0
	for i, doc := range docs {
		if s.progress != nil {
			s.progress(i+1, len(docs), doc.Name)
		}
		chunks := chunk.ChunkDefault(doc.Content)
		n, err := s.addChunks(ctx, doc.ID, chunks, embedder)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

// ListDocs 返回全部文档记录（按 ID 升序）。
func (s *baseStore) ListDocs(ctx context.Context) ([]Document, error) {
	var docs []Document
	if err := s.db.WithContext(ctx).Order("id ASC").Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("读取文档列表失败: %w", err)
	}
	return docs, nil
}

// Status 返回实现名与文档数、块数统计。
func (s *baseStore) Status(ctx context.Context, implName string) (string, int, int, error) {
	var docCount, chunkCount int64
	if err := s.db.WithContext(ctx).Model(&Document{}).Count(&docCount).Error; err != nil {
		return "", 0, 0, fmt.Errorf("统计文档数失败: %w", err)
	}
	if err := s.db.WithContext(ctx).Model(&Chunk{}).Count(&chunkCount).Error; err != nil {
		return "", 0, 0, fmt.Errorf("统计块数失败: %w", err)
	}
	return implName, int(docCount), int(chunkCount), nil
}

// docNameMap 批量查询文档 ID 到文件名的映射，供检索结果回填 DocName。
func (s *baseStore) docNameMap(ctx context.Context, docIDs []uint) (map[uint]string, error) {
	nameByID := make(map[uint]string, len(docIDs))
	if len(docIDs) == 0 {
		return nameByID, nil
	}
	var docs []Document
	if err := s.db.WithContext(ctx).Select("id, name").Where("id IN ?", docIDs).Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("查询文档名称失败: %w", err)
	}
	for _, d := range docs {
		nameByID[d.ID] = d.Name
	}
	return nameByID, nil
}

// vecStore 是基于 sqlite-vec 扩展的向量检索实现。
type vecStore struct {
	baseStore
}

// Search 使用 sqlite-vec 的 vec_distance_cosine 在 SQL 内完成最近邻检索。
func (s *vecStore) Search(ctx context.Context, queryVec []float32, topK int) ([]SearchHit, error) {
	vecSQL, err := VecF32SQL(queryVec)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(
		"SELECT doc_id, chunk_index, text, vec_distance_cosine(embedding, %s) AS dist "+
			"FROM chunks WHERE model = ? ORDER BY dist ASC LIMIT ?",
		vecSQL,
	)
	var rows []struct {
		DocID      uint
		ChunkIndex int
		Text       string
		Dist       float64
	}
	if err := s.db.WithContext(ctx).Raw(query, s.embedModel, topK).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("sqlite-vec 检索失败: %w", err)
	}

	hits := make([]SearchHit, 0, len(rows))
	if len(rows) == 0 {
		return hits, nil
	}
	// 批量回填文档名
	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.DocID)
	}
	nameByID, err := s.docNameMap(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		hits = append(hits, SearchHit{
			DocID:      r.DocID,
			DocName:    nameByID[r.DocID],
			ChunkIndex: r.ChunkIndex,
			Text:       r.Text,
			Distance:   r.Dist,
		})
	}
	return hits, nil
}

// Status 返回实现名与统计信息。
func (s *vecStore) Status(ctx context.Context) (string, int, int, error) {
	return s.baseStore.Status(ctx, "sqlite-vec")
}
