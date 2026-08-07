package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"jot/internal/aicli"
	"jot/internal/models"

	"gitee.com/MM-Q/fastlog"
	"gorm.io/gorm"
)

// VectorService 封装笔记向量索引相关的业务逻辑操作
type VectorService struct {
	db     *gorm.DB
	logger *fastlog.Logger
}

// NewVectorService 创建一个新的 VectorService 实例
func NewVectorService(db *gorm.DB, logger *fastlog.Logger) *VectorService {
	return &VectorService{db: db, logger: logger}
}

// adjacentBlocks 向量召回时命中块前后各补充的相邻块数
// 轻量父块上下文：命中小块后顺带返回其相邻块，近似"子块检索 + 父块上下文"效果
const adjacentBlocks = 1

// maxCardRunes 合并后单张召回卡片的内容长度上限（rune）
// 防止相邻块补充导致注入 token 膨胀
const maxCardRunes = 1200

// IndexNotes 对指定笔记列表逐篇量化：读取笔记正文 → 切块 → 分批 embedding → 先删该笔记旧块再插入新块（幂等）
// progressCb(done, total, title, stage, chunkDone, chunkTotal) 每篇笔记处理时回调：
//   - "embedding"：本篇开始向量化，done 为已完成的篇数，chunkDone/chunkTotal 为当前笔记块级进度
//   - "done"：本篇成功，done 为已完成的篇数，chunkDone=chunkTotal
//   - "error"：本篇失败，done 为已完成的篇数
//
// 单条笔记失败不终止整体，计入 failed；软删除笔记（deleted_at 非空）在查询阶段跳过
// 返回 (success, failed int, err error)，err 仅当整体性错误（如无有效笔记或 embedding client 配置错误）
func (s *VectorService) IndexNotes(ctx context.Context, aicli *aicli.Client, noteIDs []uint, progressCb func(done, total int, title string, stage string, chunkDone, chunkTotal int)) (success, failed int, err error) {
	// embedding client 配置检查：模型未配置时无法量化，直接返回整体性错误
	if aicli == nil || aicli.Model == "" {
		return 0, 0, fmt.Errorf("embedding 模型未配置")
	}

	// 查询未软删除的笔记（跳过回收站中的笔记）
	var notes []models.Note
	if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", noteIDs).Find(&notes).Error; err != nil {
		s.logger.Errorw("VectorService.IndexNotes 查询笔记失败", fastlog.Error(err))
		return 0, 0, fmt.Errorf("查询笔记失败: %w", err)
	}
	if len(notes) == 0 {
		return 0, 0, fmt.Errorf("没有找到有效的笔记")
	}

	total := len(notes)
	for i, note := range notes {
		// 调用方取消时提前终止，返回已处理结果
		if ctx.Err() != nil {
			return success, failed, ctx.Err()
		}

		// 切块：单块上限 500 rune；正文为空或切不出块时跳过本篇
		chunks := ChunkContent(note.Content, 500)
		if len(chunks) == 0 {
			continue
		}

		// 进度回调：开始 embedding（本篇尚未完成，done 为已完成的篇数）
		done := i
		if progressCb != nil {
			progressCb(done, total, note.Title, "embedding", 0, len(chunks))
		}

		// 分批生成所有块的向量（每批 16 块，批间回调块级进度）
		embeddings, err := aicli.EmbedWithProgress(ctx, chunks, 16, func(doneChunk, totalChunk int) {
			if progressCb != nil {
				progressCb(i, total, note.Title, "embedding", doneChunk, totalChunk)
			}
		})
		if err != nil {
			failed++
			s.logger.Errorw("VectorService.IndexNotes embedding 失败", fastlog.Uint("note_id", note.ID), fastlog.Error(err))
			if progressCb != nil {
				progressCb(i+1, total, note.Title, "error", 0, 0)
			}
			continue
		}

		// 组装向量记录：Dim 取首个向量的维度
		dim := 0
		if len(embeddings) > 0 {
			dim = len(embeddings[0])
		}
		rows := make([]models.NoteVector, 0, len(chunks))
		for idx, text := range chunks {
			rows = append(rows, models.NoteVector{
				NoteID:     note.ID,
				ChunkIndex: idx,
				ChunkText:  text,
				Embedding:  Float32ToBlob(embeddings[idx]),
				Dim:        dim,
				Model:      aicli.Model,
			})
		}

		// 事务内先删旧块再插入新块，重复索引时幂等
		txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("note_id = ?", note.ID).Delete(&models.NoteVector{}).Error; err != nil {
				return err
			}
			return tx.Create(&rows).Error
		})
		if txErr != nil {
			failed++
			s.logger.Errorw("VectorService.IndexNotes 写入向量失败", fastlog.Uint("note_id", note.ID), fastlog.Error(txErr))
			if progressCb != nil {
				progressCb(i+1, total, note.Title, "error", 0, 0)
			}
			continue
		}

		success++
		if progressCb != nil {
			progressCb(i+1, total, note.Title, "done", len(chunks), len(chunks))
		}
	}
	return success, failed, nil
}

// GetIndexStatus 返回向量索引统计信息：已量化笔记数（去重 note_id）、片段总数、占用字节
func (s *VectorService) GetIndexStatus() (noteCount, chunkCount int, sizeBytes int64, err error) {
	var noteCnt int64
	if err := s.db.Model(&models.NoteVector{}).Distinct("note_id").Count(&noteCnt).Error; err != nil {
		return 0, 0, 0, err
	}

	var chunkCnt int64
	if err := s.db.Model(&models.NoteVector{}).Count(&chunkCnt).Error; err != nil {
		return 0, 0, 0, err
	}

	// 占用字节 = 向量 BLOB 字节 + 切块文本字节
	var size int64
	if err := s.db.Model(&models.NoteVector{}).Select("COALESCE(SUM(LENGTH(chunk_text) + LENGTH(embedding)), 0)").Scan(&size).Error; err != nil {
		return 0, 0, 0, err
	}

	return int(noteCnt), int(chunkCnt), size, nil
}

// CountAllVectors 统计 note_vectors 表总记录数
func (s *VectorService) CountAllVectors() (int64, error) {
	var cnt int64
	if err := s.db.Model(&models.NoteVector{}).Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

// CountVectorsByModel 统计指定 embedding 模型在 note_vectors 表中的向量记录数
func (s *VectorService) CountVectorsByModel(model string) (int64, error) {
	var cnt int64
	if err := s.db.Model(&models.NoteVector{}).Where("model = ?", model).Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

// DeleteAllVectors 清空 note_vectors 表（物理删除所有向量记录）
func (s *VectorService) DeleteAllVectors() error {
	return s.db.Where("1 = 1").Delete(&models.NoteVector{}).Error
}

// VectorRecall 向量召回：将 query 向量化后，通过 sqlite-vec 扩展在 SQL 内计算余弦距离，
// 按距离升序召回 TopN 命中块，并补充命中块前后各 1 个相邻块，按笔记合并组装卡片
// 返回 CardRecallResult（FormattedText 注入 system message + Cards 用于前端展示）
// 任一前置条件不满足时返回 nil（静默跳过，不注入不发射）：
//   - embedClient 为空或其 Model 为空（未配置量化连接/模型）
//   - note_vectors 表中没有与当前 Model 匹配的向量数据（未量化或换了模型）
//   - query 向量化失败或召回结果为空
//
// notebookIDs 非空时仅召回指定笔记本下的笔记（join notes 过滤）
func (s *VectorService) VectorRecall(ctx context.Context, query string, limit int, embedClient *aicli.Client, notebookIDs ...uint) *CardRecallResult {
	if query == "" || limit <= 0 {
		return nil
	}
	// embedding client 配置检查：模型未配置时无法向量化 query，静默跳过
	if embedClient == nil || embedClient.Model == "" {
		s.logger.Debugw("VectorService.VectorRecall 跳过：embedding 模型未配置")
		return nil
	}

	// 前置检查：当前模型是否有向量数据，无则静默跳过
	var cnt int64
	if err := s.db.WithContext(ctx).Model(&models.NoteVector{}).Where("model = ?", embedClient.Model).Count(&cnt).Error; err != nil {
		s.logger.Errorw("VectorService.VectorRecall 查询向量计数失败", fastlog.Error(err))
		return nil
	}
	if cnt == 0 {
		s.logger.Debugw("VectorService.VectorRecall 跳过：当前模型无向量数据",
			fastlog.String("model", embedClient.Model))
		return nil
	}

	// query 向量化：取首个向量作为查询向量
	embeddings, err := embedClient.Embed(ctx, []string{query})
	if err != nil || len(embeddings) == 0 {
		s.logger.Errorw("VectorService.VectorRecall query 向量化失败", fastlog.Error(err))
		return nil
	}
	queryVec := embeddings[0]
	// query 向量转 JSON 数组字符串，供 sqlite-vec 的 vec_f32 解析
	queryVecJSON, err := json.Marshal(queryVec)
	if err != nil {
		s.logger.Errorw("VectorService.VectorRecall query 向量序列化失败", fastlog.Error(err))
		return nil
	}

	// sqlite-vec 函数式检索：SQL 内计算余弦距离（vec_distance_cosine），按距离升序取 TopN
	// dist < 1.0 等价原逻辑 score > 0 过滤；可选按笔记本范围过滤（join notes 跳过软删除笔记）
	// 列名加 note_vectors 前缀，避免 JOIN notes 时 id 列歧义；JOIN 必须紧跟 FROM（位于 WHERE 之前）
	vecSQL := "SELECT note_vectors.id, note_vectors.note_id, note_vectors.chunk_index, note_vectors.chunk_text, note_vectors.model " +
		"FROM note_vectors"
	args := []interface{}{}
	if len(notebookIDs) > 0 {
		vecSQL += " JOIN notes ON notes.id = note_vectors.note_id " +
			"AND notes.deleted_at IS NULL AND notes.notebook_id IN ?"
		args = append(args, notebookIDs)
	}
	vecSQL += " WHERE note_vectors.model = ? " +
		"AND vec_distance_cosine(note_vectors.embedding, vec_f32(?)) < 1.0" +
		" ORDER BY vec_distance_cosine(note_vectors.embedding, vec_f32(?)) ASC LIMIT ?"
	args = append(args, embedClient.Model, string(queryVecJSON), string(queryVecJSON), limit)

	var hits []models.NoteVector
	if err := s.db.WithContext(ctx).Raw(vecSQL, args...).Scan(&hits).Error; err != nil {
		s.logger.Errorw("VectorService.VectorRecall sqlite-vec 检索失败", fastlog.Error(err))
		return nil
	}
	if len(hits) == 0 {
		s.logger.Debugw("VectorService.VectorRecall 无命中",
			fastlog.String("query", query), fastlog.String("model", embedClient.Model))
		return nil
	}

	// 按命中 note_id 批量查询笔记元信息（标题/文件后缀），用于组装卡片
	noteIDs := make([]uint, 0, len(hits))
	for _, h := range hits {
		noteIDs = append(noteIDs, h.NoteID)
	}
	var notes []models.Note
	if err := s.db.WithContext(ctx).Where("id IN ?", noteIDs).Find(&notes).Error; err != nil {
		s.logger.Errorw("VectorService.VectorRecall 查询笔记失败", fastlog.Error(err))
		return nil
	}
	noteMeta := make(map[uint]models.Note, len(notes))
	for _, n := range notes {
		noteMeta[n.ID] = n
	}

	// 二次查询命中笔记的全部块（按 ChunkIndex 升序），用于相邻块补充与按笔记合并卡片
	var blocks []models.NoteVector
	if err := s.db.WithContext(ctx).Where("model = ? AND note_id IN ?", embedClient.Model, noteIDs).
		Order("chunk_index ASC").Find(&blocks).Error; err != nil {
		s.logger.Errorw("VectorService.VectorRecall 查询命中笔记块失败", fastlog.Error(err))
		return nil
	}
	byNote := make(map[uint][]models.NoteVector)
	for _, v := range blocks {
		byNote[v.NoteID] = append(byNote[v.NoteID], v)
	}

	// 按笔记收集需要返回的 ChunkIndex 集合：命中块本身 + 前后 adjacentBlocks 个相邻块
	hitIndexes := make(map[uint]map[int]bool)
	for _, h := range hits {
		noteID := h.NoteID
		if hitIndexes[noteID] == nil {
			hitIndexes[noteID] = make(map[int]bool)
		}
		idx := h.ChunkIndex
		for _, v := range byNote[noteID] {
			if v.ChunkIndex >= idx-adjacentBlocks && v.ChunkIndex <= idx+adjacentBlocks {
				hitIndexes[noteID][v.ChunkIndex] = true
			}
		}
	}

	// 命中笔记顺序按 hits 首次出现先后（= 距离升序），保持相关度排序稳定
	hitOrder := make([]uint, 0, len(hitIndexes))
	seenNote := make(map[uint]bool, len(hitIndexes))
	for _, h := range hits {
		if !seenNote[h.NoteID] {
			seenNote[h.NoteID] = true
			hitOrder = append(hitOrder, h.NoteID)
		}
	}

	// 组装格式化上下文文本与结构化卡片
	// Content 为命中块 + 相邻块按 ChunkIndex 拼接（按笔记合并为一张卡片）
	var b strings.Builder
	b.WriteString("以下是用户笔记库中与问题相关的笔记片段（来源：本地笔记向量检索），请优先参考这些笔记内容回答用户的问题：\n\n")

	cards := make([]RecallCard, 0, len(hitOrder))
	for _, noteID := range hitOrder {
		note, ok := noteMeta[noteID]
		if !ok {
			continue
		}
		var parts []string
		for _, v := range byNote[noteID] {
			if hitIndexes[noteID][v.ChunkIndex] {
				parts = append(parts, v.ChunkText)
			}
		}
		content := strings.Join(parts, "\n\n")
		// 单卡片内容长度上限，防止相邻块补充导致注入 token 膨胀
		if runeLen(content) > maxCardRunes {
			content = string([]rune(content)[:maxCardRunes])
		}
		fmt.Fprintf(&b, "--- 📄 《%s》 ---\n%s\n\n", note.Title, content)
		cards = append(cards, RecallCard{
			ID:        note.ID,
			Title:     note.Title,
			Content:   content,
			FileExt:   note.FileExt,
			Truncated: false,
		})
	}
	if len(cards) == 0 {
		s.logger.Debugw("VectorService.VectorRecall 组装卡片为空")
		return nil
	}
	b.WriteString("请基于以上笔记内容回答用户的问题。如果笔记内容不足以回答，请如实说明。")

	s.logger.Debugw("VectorService.VectorRecall 命中",
		fastlog.Int("cards_count", len(cards)),
		fastlog.String("query", query),
		fastlog.String("model", embedClient.Model))

	return &CardRecallResult{
		FormattedText: b.String(),
		Cards:         cards,
	}
}
