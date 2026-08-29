package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"

	"jot/internal/aierrors"
	"jot/internal/einocli"
	"jot/internal/models"

	"gitee.com/MM-Q/fastlog"
	"github.com/go-ego/gse"
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

// chunkCandidateMultiplier 向量检索 chunk 候选放大倍数：
// vectorSearch 的 SQL LIMIT 取 limit×该倍数，先多捞候选块（第 6 名以后也有机会），
// 再由 selectTopNotes 按笔记聚合截断回 limit 个笔记，保证相关度优先 + 笔记多样性
const chunkCandidateMultiplier = 5

// maxChunksPerNote 每个命中笔记最多保留的命中块数，防止单篇笔记命中块过多挤占其他笔记的卡片槽位
const maxChunksPerNote = 4

// chunkMaxRunes 笔记切块单块 rune 上限（IndexNotes 写路径与 classifyVectorNotes 状态比对共用，
// 两处必须一致，否则内容未变也会被判为"需重新嵌入"）
const chunkMaxRunes = 600

// ===== GSE 中文分词器（懒加载，EMBED 嵌入式词典） =====

// 停用词表，过滤分词结果中的高频无意义单字
var stopWords = map[rune]struct{}{
	// 高频单字停用词
	'的': {}, '了': {}, '是': {}, '在': {}, '有': {}, '我': {}, '他': {}, '她': {},
	'它': {}, '们': {}, '这': {}, '那': {}, '什': {}, '么': {}, '怎': {}, '哪': {},
	'你': {}, '之': {}, '于': {}, '其': {}, '着': {}, '过': {},
	'里': {}, '为': {}, '因': {}, '所': {}, '以': {}, '但': {}, '如': {}, '果': {},
	'虽': {}, '然': {}, '而': {}, '且': {}, '或': {}, '与': {}, '和': {}, '同': {},
	'及': {}, '又': {}, '也': {}, '对': {}, '就': {}, '被': {}, '把': {}, '让': {},
	'向': {}, '往': {}, '从': {}, '到': {}, '去': {}, '能': {}, '会': {}, '要': {},
	'可': {}, '没': {}, '不': {}, '很': {}, '太': {}, '更': {}, '最': {}, '都': {},
	'只': {}, '还': {}, '再': {}, '才': {}, '刚': {}, '已': {}, '正': {}, '将': {},
	'该': {}, '应': {}, '需': {}, '必': {}, '须': {}, '够': {}, '出': {}, '入': {},
	'上': {}, '下': {}, '大': {}, '小': {}, '多': {}, '少': {}, '来': {}, '做': {},
	'用': {}, '问': {}, '说': {}, '看': {}, '想': {}, '知': {}, '道': {}, '给': {},
	'跟': {}, '比': {}, '次': {}, '个': {}, '种': {}, '些': {}, '点': {}, '等': {},
	'第': {}, '每': {}, '各': {}, '几': {}, '两': {}, '百': {}, '千': {}, '万': {},
	'亿': {}, '哦': {}, '啊': {}, '嗯': {}, '呢': {}, '吧': {}, '吗': {}, '呀': {},
	'嘛': {}, '哈': {}, '哇': {}, '呵': {}, '嘿': {}, '喔': {},
}

// isStopWord 判断 rune 是否为停用词
func isStopWord(r rune) bool {
	_, ok := stopWords[r]
	return ok
}

// maxRecallKeywords 卡片召回最大关键词数，防止超长 query 导致 LIKE 查询性能问题
const maxRecallKeywords = 20

// kwHighFreqDivisor / kwHighFreqMin 高频词过滤阈值：
// 命中块数超过 max(totalChunks/kwHighFreqDivisor, kwHighFreqMin) 的 token 视为无区分度高频词，检索时丢弃
// 依据实测："数据"命中 ~93% 块、"2061"命中 ~1% 块；"数据"这类词进 OR LIKE 只会刷屏
const kwHighFreqDivisor = 10
const kwHighFreqMin = 100

var (
	gseSeg     gse.Segmenter
	gseOnce    sync.Once
	gseInitErr error
)

// initGseSegmenter 初始化 gse 分词器，载入 EMBED 内置词典
func initGseSegmenter() {
	gseSeg = gse.Segmenter{}
	gseInitErr = gseSeg.LoadDictEmbed()
}

// tokenize 使用 gse 对输入文本做分词
// - 调用 gse.Cut 精确模式+HMM
// - 过滤停用词（仅对单字词做停用词检查）
// - 过滤纯标点/符号
// - 去重
func tokenize(text string) []string {
	gseOnce.Do(initGseSegmenter)
	if gseInitErr != nil {
		return nil
	}
	words := gseSeg.Cut(text, true)
	seen := make(map[string]bool)
	var result []string
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w == "" || seen[w] {
			continue
		}
		// 过滤纯标点/符号
		if isAllPunct(w) {
			continue
		}
		// 过滤停用词（仅对单字词做停用词检查）
		runes := []rune(w)
		if len(runes) == 1 && isStopWord(runes[0]) {
			continue
		}
		seen[w] = true
		result = append(result, w)
	}
	return result
}

// isAllPunct 判断字符串是否全部为标点/符号
func isAllPunct(s string) bool {
	for _, r := range s {
		if !unicode.IsPunct(r) && !unicode.IsSymbol(r) {
			return false
		}
	}
	return true
}

// IndexNotes 对指定笔记列表逐篇嵌入：读取笔记正文 → 切块 → 分批 embedding → 先删该笔记旧块再插入新块（幂等）
// progressCb(done, total, title, stage, chunkDone, chunkTotal) 每篇笔记处理时回调：
//   - "embedding"：本篇开始向量化，done 为已完成的篇数，chunkDone/chunkTotal 为当前笔记块级进度
//   - "done"：本篇成功，done 为已完成的篇数，chunkDone=chunkTotal
//   - "error"：本篇失败，done 为已完成的篇数
//
// 单条笔记失败不终止整体，计入 failed；软删除笔记（deleted_at 非空）在查询阶段跳过
// 返回 (success, failed int, err error)，err 仅当整体性错误（如无有效笔记或 embedding client 配置错误）
func (s *VectorService) IndexNotes(ctx context.Context, embedClient *einocli.Client, noteIDs []uint, progressCb func(done, total int, title string, stage string, chunkDone, chunkTotal int, errMsg string)) (success, failed int, err error) {
	// embedding client 配置检查：模型未配置时无法嵌入，直接返回整体性错误
	if embedClient == nil || embedClient.Model == "" {
		return 0, 0, fmt.Errorf("embedding 模型未配置")
	}

	// 查询未软删除的笔记（跳过回收站中的笔记）；预加载 Tags 用于构造分块元数据前缀
	var notes []models.Note
	if err := s.db.WithContext(ctx).Preload("Tags").Where("id IN ? AND deleted_at IS NULL", noteIDs).Find(&notes).Error; err != nil {
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

		// 构造分块元数据前缀（标题/标签/创建时间），提升每块检索命中率
		tagNames := make([]string, 0, len(note.Tags))
		for _, tag := range note.Tags {
			tagNames = append(tagNames, tag.Name)
		}
		// 标签排序保证确定性：与 classifyVectorNotes 状态比对口径一致，
		// 避免 GORM Preload 顺序变化导致内容未变却误判"需重新嵌入"
		sort.Strings(tagNames)
		meta := ChunkMeta{
			Title:     note.Title,
			Tags:      tagNames,
			CreatedAt: note.CreatedAt,
		}

		// 切块：单块上限 600 rune（含元数据前缀）；正文为空或切不出块时跳过本篇
		chunks := ChunkContent(note.Content, chunkMaxRunes, meta)
		if len(chunks) == 0 {
			continue
		}

		// 进度回调：开始 embedding（本篇尚未完成，done 为已完成的篇数）
		done := i
		if progressCb != nil {
			progressCb(done, total, note.Title, "embedding", 0, len(chunks), "")
		}

		// 分批生成所有块的向量（每批 16 块，批间回调块级进度）
		embeddings, err := embedClient.EmbedWithProgress(ctx, chunks, 16, func(doneChunk, totalChunk int) {
			if progressCb != nil {
				progressCb(i, total, note.Title, "embedding", doneChunk, totalChunk, "")
			}
		})
		if err != nil {
			failed++
			s.logger.Errorw("VectorService.IndexNotes embedding 失败", fastlog.Uint("note_id", note.ID), fastlog.Error(err))
			// 与 AI 助手一致：错误分类为中文友好提示后随进度事件推送，前端直接展示
			userMsg := err.Error()
			if aiErr := aierrors.ClassifyError(err); aiErr != nil && aiErr.UserMsg != "" {
				userMsg = aiErr.UserMsg
			}
			if progressCb != nil {
				progressCb(i+1, total, note.Title, "error", 0, 0, userMsg)
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
				Model:      embedClient.Model,
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
				progressCb(i+1, total, note.Title, "error", 0, 0, txErr.Error())
			}
			continue
		}

		success++
		if progressCb != nil {
			progressCb(i+1, total, note.Title, "done", len(chunks), len(chunks), "")
		}
	}
	return success, failed, nil
}

// GetIndexStatus 返回向量索引统计信息：已嵌入笔记数（去重 note_id）、片段总数、占用字节
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

// ===== 嵌入状态分类（未嵌入 / 需重新嵌入 / 已最新） =====

// vectorNoteStatus 嵌入状态分类结果：按笔记 ID 集合分类，供状态统计与定向嵌入入口共用
type vectorNoteStatus struct {
	TotalNotes   int
	IndexedNotes int
	UnindexedIDs []uint // 未嵌入（无向量记录）的非软删笔记
	StaleIDs     []uint // 已嵌入但当前内容与嵌入时不一致，需重新嵌入
	UpToDateIDs  []uint // 已嵌入且当前内容与嵌入时一致
}

// classifyVectorNotes 对全部非软删笔记做嵌入状态分类（一次计算，多个调用方复用）：
//  1. 未嵌入 = 无任何向量记录的非软删笔记
//  2. 需重新嵌入 = 已有向量记录，但用当前内容（标题/标签/创建时间/正文）重新切块后
//     与 note_vectors 中存储的块文本不一致（块数不同或任一文本不同）
//  3. 已最新 = 重新切块结果与存储块完全一致
//
// 复用 IndexNotes 同一套 ChunkContent 切块口径（maxRunes=600）；标签名排序保证与写路径一致，
// 存量旧块若因标签顺序不同被误判为需重新嵌入，重新嵌入一次后即稳定（自愈）
func (s *VectorService) classifyVectorNotes(ctx context.Context) (*vectorNoteStatus, error) {
	status := &vectorNoteStatus{}

	// 1. 全部非软删笔记 id（软删/回收站笔记不参与统计与定向嵌入）
	var allIDs []uint
	if err := s.db.WithContext(ctx).Model(&models.Note{}).
		Where("deleted_at IS NULL").Pluck("id", &allIDs).Error; err != nil {
		s.logger.Errorw("VectorService.classifyVectorNotes 查询笔记失败", fastlog.Error(err))
		return nil, fmt.Errorf("查询笔记失败: %w", err)
	}
	status.TotalNotes = len(allIDs)

	// 2. 已有向量的非软删笔记：note_id -> 按 chunk_index 升序的存储块文本
	var rows []struct {
		NoteID    uint
		ChunkText string
	}
	if err := s.db.WithContext(ctx).Raw(
		"SELECT nv.note_id, nv.chunk_text FROM note_vectors nv " +
			"JOIN notes n ON n.id = nv.note_id AND n.deleted_at IS NULL " +
			"ORDER BY nv.note_id, nv.chunk_index").Scan(&rows).Error; err != nil {
		s.logger.Errorw("VectorService.classifyVectorNotes 查询向量失败", fastlog.Error(err))
		return nil, fmt.Errorf("查询向量记录失败: %w", err)
	}
	byNote := make(map[uint][]string)
	indexedSet := make(map[uint]bool)
	for _, r := range rows {
		byNote[r.NoteID] = append(byNote[r.NoteID], r.ChunkText)
		indexedSet[r.NoteID] = true
	}
	status.IndexedNotes = len(indexedSet)

	// 3. 未嵌入：无向量记录的非软删笔记
	for _, id := range allIDs {
		if !indexedSet[id] {
			status.UnindexedIDs = append(status.UnindexedIDs, id)
		}
	}

	// 4. 对已嵌入笔记做内容比对：重新切块 vs 存储块
	if len(indexedSet) > 0 {
		indexedIDs := make([]uint, 0, len(indexedSet))
		for id := range indexedSet {
			indexedIDs = append(indexedIDs, id)
		}
		// GORM 自动追加 deleted_at IS NULL 过滤（索引集中的笔记本就非软删，双保险）
		var notes []models.Note
		if err := s.db.WithContext(ctx).Preload("Tags").Where("id IN ?", indexedIDs).Find(&notes).Error; err != nil {
			s.logger.Errorw("VectorService.classifyVectorNotes 查询笔记内容失败", fastlog.Error(err))
			return nil, fmt.Errorf("查询笔记内容失败: %w", err)
		}
		for _, note := range notes {
			// 与 IndexNotes 一致的标签排序，保证切块口径确定
			tagNames := make([]string, 0, len(note.Tags))
			for _, tag := range note.Tags {
				tagNames = append(tagNames, tag.Name)
			}
			sort.Strings(tagNames)
			meta := ChunkMeta{
				Title:     note.Title,
				Tags:      tagNames,
				CreatedAt: note.CreatedAt,
			}
			current := ChunkContent(note.Content, chunkMaxRunes, meta)
			if chunksEqual(current, byNote[note.ID]) {
				status.UpToDateIDs = append(status.UpToDateIDs, note.ID)
			} else {
				status.StaleIDs = append(status.StaleIDs, note.ID)
			}
		}
	}

	return status, nil
}

// chunksEqual 比较重新切块结果与存储块文本是否完全一致（块数 + 逐块文本）
func chunksEqual(current, stored []string) bool {
	if len(current) != len(stored) {
		return false
	}
	for i := range current {
		if current[i] != stored[i] {
			return false
		}
	}
	return true
}

// GetVectorNoteOverview 返回嵌入状态统计：总笔记数 / 未嵌入 / 需重新嵌入 / 已最新（均为非软删笔记口径）
func (s *VectorService) GetVectorNoteOverview() (totalNotes, unindexedNotes, staleNotes, upToDateNotes int, err error) {
	status, err := s.classifyVectorNotes(context.Background())
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return status.TotalNotes, len(status.UnindexedIDs), len(status.StaleIDs), len(status.UpToDateIDs), nil
}

// GetUnindexedNoteIDs 返回未嵌入的非软删笔记 ID 列表（供"仅嵌入未嵌入笔记"入口使用）
func (s *VectorService) GetUnindexedNoteIDs() ([]uint, error) {
	status, err := s.classifyVectorNotes(context.Background())
	if err != nil {
		return nil, err
	}
	return status.UnindexedIDs, nil
}

// GetStaleNoteIDs 返回需重新嵌入（内容已变化）的非软删笔记 ID 列表
func (s *VectorService) GetStaleNoteIDs() ([]uint, error) {
	status, err := s.classifyVectorNotes(context.Background())
	if err != nil {
		return nil, err
	}
	return status.StaleIDs, nil
}

// ===== 关键词检索 + 混合召回 =====

// enableKeywordRecall 关键词检索开关
// 临时禁用关键词召回用于对比测试时置为 false，恢复后改回 true 即可（仅影响 HybridRecall 内部，无需改其他代码）
const enableKeywordRecall = true

// recallHit 表示一个合并后的命中块，记录命中来源和评分信息
type recallHit struct {
	vec     models.NoteVector
	sources int // bit 0=向量, bit 1=关键词 → 1=仅向量, 2=仅关键词, 3=双命中
	kwScore int // 关键词命中 token 数（仅关键词路有意义）
}

// filterHighFreqTokens 剔除命中数超过阈值的高频词（无区分度），返回保留的 token 列表
// threshold = max(total/kwHighFreqDivisor, kwHighFreqMin)
func filterHighFreqTokens(tokens []string, counts []int, total int) []string {
	threshold := total / kwHighFreqDivisor
	if threshold < kwHighFreqMin {
		threshold = kwHighFreqMin
	}
	kept := make([]string, 0, len(tokens))
	for i, t := range tokens {
		if counts[i] > threshold {
			continue
		}
		kept = append(kept, t)
	}
	return kept
}

// rankKwHits 按"命中 token 数"降序排序候选块（同分按块 id 升序，保证稳定），截断到 limit
// 与 HybridRecall 中 kwScore 口径一致（都统计块文本包含的 token 数）
func rankKwHits(hits []models.NoteVector, tokens []string, limit int) []models.NoteVector {
	if limit <= 0 {
		return nil
	}
	type scored struct {
		hit   models.NoteVector
		score int
	}
	list := make([]scored, 0, len(hits))
	for _, h := range hits {
		sc := 0
		for _, t := range tokens {
			if strings.Contains(h.ChunkText, t) {
				sc++
			}
		}
		list = append(list, scored{h, sc})
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].score != list[j].score {
			return list[i].score > list[j].score
		}
		return list[i].hit.ID < list[j].hit.ID
	})
	if len(list) > limit {
		list = list[:limit]
	}
	out := make([]models.NoteVector, 0, len(list))
	for _, s := range list {
		out = append(out, s.hit)
	}
	return out
}

// KeywordRecall 关键词检索：GSE 分词后在 note_vectors.chunk_text 上执行 LIKE 匹配
// 不依赖 embedding 模型，可跨模型检索；返回命中的 NoteVector 列表
// notebookIDs 非空时仅检索指定笔记本下的笔记
//
// 检索流程（第一级修复）：
//  1. 统计总块数与各 token 命中数（COUNT + LIKE）
//  2. 高频词过滤：命中数超过 max(总块数/10, 100) 的 token 丢弃（如"数据"这类 ~93% 命中率的无区分度词）
//  3. 候选放大：主查询 LIMIT 取 limit×chunkCandidateMultiplier，避免好块被直接截断
//  4. 截断前排序：按"命中 token 数"降序 + 块 id 升序，截断回 limit
func (s *VectorService) KeywordRecall(ctx context.Context, query string, limit int, notebookIDs ...uint) ([]models.NoteVector, error) {
	tokens := tokenize(query)
	if len(tokens) == 0 {
		return nil, nil
	}

	// 限制关键词数量，防止超长 LIKE 查询
	if len(tokens) > maxRecallKeywords {
		tokens = tokens[:maxRecallKeywords]
	}

	// 构建基础过滤子句：JOIN notes 过滤软删除笔记，notebookIDs 非空时限定笔记本
	// 计数与主检索共用，保证口径一致
	base := "FROM note_vectors JOIN notes ON notes.id = note_vectors.note_id AND notes.deleted_at IS NULL"
	cntArgs := []interface{}{}
	if len(notebookIDs) > 0 {
		base += " AND notes.notebook_id IN ?"
		cntArgs = append(cntArgs, notebookIDs)
	}

	// 总块数（用于相对阈值）
	var total int
	if err := s.db.WithContext(ctx).Raw("SELECT COUNT(*) "+base, cntArgs...).Scan(&total).Error; err != nil {
		s.logger.Errorw("VectorService.KeywordRecall 统计总块数失败", fastlog.Error(err))
		return nil, fmt.Errorf("统计笔记块总数失败: %w", err)
	}

	// 各 token 命中数（COUNT + LIKE）
	counts := make([]int, len(tokens))
	for i, t := range tokens {
		args := append(append([]interface{}{}, cntArgs...), "%"+t+"%")
		if err := s.db.WithContext(ctx).Raw("SELECT COUNT(*) "+base+" WHERE note_vectors.chunk_text LIKE ?", args...).Scan(&counts[i]).Error; err != nil {
			s.logger.Errorw("VectorService.KeywordRecall 统计 token 命中数失败", fastlog.String("token", t), fastlog.Error(err))
			return nil, fmt.Errorf("统计关键词命中数失败: %w", err)
		}
	}

	// 高频词过滤：无区分度 token 丢弃；全部被过滤时关键词路不贡献，返回空
	before := len(tokens)
	tokens = filterHighFreqTokens(tokens, counts, total)
	if len(tokens) == 0 {
		s.logger.Debugw("VectorService.KeywordRecall 无有效关键词",
			fastlog.String("query", query),
			fastlog.Int("dropped", before))
		return nil, nil
	}

	// 构建主检索 SQL：OR 拼接各 token 的 LIKE，LIMIT 放大避免好块被截断
	// 不加 model 过滤——关键词检索跨所有模型
	kwSQL := "SELECT note_vectors.id, note_vectors.note_id, note_vectors.chunk_index, note_vectors.chunk_text, note_vectors.model " +
		"FROM note_vectors JOIN notes ON notes.id = note_vectors.note_id AND notes.deleted_at IS NULL"
	args := []interface{}{}
	if len(notebookIDs) > 0 {
		kwSQL += " AND notes.notebook_id IN ?"
		args = append(args, notebookIDs)
	}
	kwSQL += " WHERE "
	for i, t := range tokens {
		if i > 0 {
			kwSQL += " OR "
		}
		kwSQL += "note_vectors.chunk_text LIKE ?"
		args = append(args, "%"+t+"%")
	}
	kwSQL += " LIMIT ?"
	args = append(args, limit*chunkCandidateMultiplier)

	var hits []models.NoteVector
	if err := s.db.WithContext(ctx).Raw(kwSQL, args...).Scan(&hits).Error; err != nil {
		s.logger.Errorw("VectorService.KeywordRecall SQL 检索失败", fastlog.Error(err))
		return nil, fmt.Errorf("关键词检索失败: %w", err)
	}

	// 截断前排序：按命中 token 数降序（同分按块 id 升序），截断回 limit
	hits = rankKwHits(hits, tokens, limit)

	s.logger.Debugw("VectorService.KeywordRecall 命中",
		fastlog.String("query", query),
		fastlog.String("tokens", strings.Join(tokens, " / ")),
		fastlog.Int("dropped", before-len(tokens)),
		fastlog.Int("hits", len(hits)))

	return hits, nil
}

// vectorSearch 向量检索：将 query 向量化后，通过 sqlite-vec 扩展在 SQL 内计算余弦距离，
// 按距离升序召回 TopN 命中块
// embedClient 为 nil 或模型未配置时返回 (nil, nil) 静默跳过，由调用方决定是否降级为仅关键词检索
func (s *VectorService) vectorSearch(ctx context.Context, query string, limit int, embedClient *einocli.Client, notebookIDs []uint) ([]models.NoteVector, error) {
	// embedding client 配置检查：模型未配置时无法向量化 query，静默跳过
	if embedClient == nil || embedClient.Model == "" {
		s.logger.Debugw("VectorService.vectorSearch 跳过：embedding 模型未配置")
		return nil, nil
	}

	// 前置检查：当前模型是否有向量数据，无则静默跳过
	var cnt int64
	if err := s.db.WithContext(ctx).Model(&models.NoteVector{}).Where("model = ?", embedClient.Model).Count(&cnt).Error; err != nil {
		s.logger.Errorw("VectorService.vectorSearch 查询向量计数失败", fastlog.Error(err))
		return nil, fmt.Errorf("查询向量计数失败: %w", err)
	}
	if cnt == 0 {
		s.logger.Debugw("VectorService.vectorSearch 跳过：当前模型无向量数据",
			fastlog.String("model", embedClient.Model))
		return nil, nil
	}

	// query 向量化：取首个向量作为查询向量
	embeddings, err := embedClient.Embed(ctx, []string{query})
	if err != nil {
		s.logger.Errorw("VectorService.vectorSearch query 向量化失败", fastlog.Error(err))
		return nil, fmt.Errorf("query 向量化失败: %w", err)
	}
	if len(embeddings) == 0 {
		s.logger.Errorw("VectorService.vectorSearch query 向量化失败", fastlog.String("detail", "返回空向量"))
		return nil, fmt.Errorf("query 向量化失败: 返回空向量")
	}
	// query 向量转 JSON 数组字符串，供 sqlite-vec 的 vec_f32 解析
	queryVecJSON, err := json.Marshal(embeddings[0])
	if err != nil {
		s.logger.Errorw("VectorService.vectorSearch query 向量序列化失败", fastlog.Error(err))
		return nil, fmt.Errorf("query 向量序列化失败: %w", err)
	}

	// sqlite-vec 函数式检索：SQL 内计算余弦距离（vec_distance_cosine），按距离升序取候选 TopN
	// LIMIT 取 limit×chunkCandidateMultiplier：先多捞候选块，再由 selectTopNotes 按笔记聚合截断回 limit 个笔记
	// dist < 1.0 等价原逻辑 score > 0 过滤；无条件 JOIN notes 过滤软删除笔记（回收站笔记不参与召回），
	// 指定笔记本时 ON 条件追加 notebook_id 过滤；JOIN 必须紧跟 FROM（位于 WHERE 之前），
	// 列名加 note_vectors 前缀避免 JOIN notes 时 id 列歧义
	vecSQL := "SELECT note_vectors.id, note_vectors.note_id, note_vectors.chunk_index, note_vectors.chunk_text, note_vectors.model " +
		"FROM note_vectors JOIN notes ON notes.id = note_vectors.note_id AND notes.deleted_at IS NULL"
	args := []interface{}{}
	if len(notebookIDs) > 0 {
		vecSQL += " AND notes.notebook_id IN ?"
		args = append(args, notebookIDs)
	}
	vecSQL += " WHERE note_vectors.model = ? " +
		"AND vec_distance_cosine(note_vectors.embedding, vec_f32(?)) < 1.0" +
		" ORDER BY vec_distance_cosine(note_vectors.embedding, vec_f32(?)) ASC LIMIT ?"
	args = append(args, embedClient.Model, string(queryVecJSON), string(queryVecJSON), limit*chunkCandidateMultiplier)

	var hits []models.NoteVector
	if err := s.db.WithContext(ctx).Raw(vecSQL, args...).Scan(&hits).Error; err != nil {
		s.logger.Errorw("VectorService.vectorSearch sqlite-vec 检索失败", fastlog.Error(err))
		return nil, fmt.Errorf("向量检索失败: %w", err)
	}

	s.logger.Debugw("VectorService.vectorSearch 命中",
		fastlog.String("query", query),
		fastlog.String("model", embedClient.Model),
		fastlog.Int("hits", len(hits)))

	return hits, nil
}

// HybridRecall 混合召回：并行执行向量检索与关键词检索，按 (note_id, chunk_index) 去重合并
// 排序优先级：双命中（向量+关键词）> 仅向量命中 > 仅关键词命中
// 合并后补充相邻块并组装卡片，返回 CardRecallResult
func (s *VectorService) HybridRecall(ctx context.Context, query string, limit int, embedClient *einocli.Client, notebookIDs ...uint) (*CardRecallResult, error) {
	if query == "" || limit <= 0 {
		return nil, nil
	}

	// 向量检索路
	vecHits, err := s.vectorSearch(ctx, query, limit, embedClient, notebookIDs)
	if err != nil {
		s.logger.Errorw("VectorService.HybridRecall 向量检索失败", fastlog.Error(err))
		return nil, err
	}

	// 关键词检索路（enableKeywordRecall=false 时临时禁用，仅走向量检索）
	var kwHits []models.NoteVector
	if enableKeywordRecall {
		kwHits, err = s.KeywordRecall(ctx, query, limit, notebookIDs...)
		if err != nil {
			s.logger.Errorw("VectorService.HybridRecall 关键词检索失败", fastlog.Error(err))
			// 关键词检索失败不中断，继续使用向量结果
			kwHits = nil
		}
	}

	// 合并去重：按 (note_id, chunk_index) 作为唯一键
	hitMap := make(map[string]*recallHit)
	keyOf := func(noteID uint, chunkIdx int) string {
		return fmt.Sprintf("%d:%d", noteID, chunkIdx)
	}

	for _, h := range vecHits {
		k := keyOf(h.NoteID, h.ChunkIndex)
		if existing, ok := hitMap[k]; ok {
			existing.sources |= 1 // 标记向量命中
		} else {
			hitMap[k] = &recallHit{vec: h, sources: 1}
		}
	}

	// 关键词命中 token 列表（用于计算 kwScore）
	kwTokens := tokenize(query)
	for _, h := range kwHits {
		k := keyOf(h.NoteID, h.ChunkIndex)
		score := 0
		for _, t := range kwTokens {
			if strings.Contains(h.ChunkText, t) {
				score++
			}
		}
		if existing, ok := hitMap[k]; ok {
			existing.sources |= 2 // 标记关键词命中
			existing.kwScore = score
		} else {
			hitMap[k] = &recallHit{vec: h, sources: 2, kwScore: score}
		}
	}

	if len(hitMap) == 0 {
		s.logger.Debugw("VectorService.HybridRecall 无命中",
			fastlog.String("query", query),
			fastlog.Int("vec_hits", len(vecHits)),
			fastlog.Int("kw_hits", len(kwHits)))
		return nil, nil
	}

	// 排序：双命中 > 仅向量 > 仅关键词；同优先级内保持向量原始顺序（距离升序）
	// 向量命中先入 hitMap，天然保持距离升序；关键词命中追加在后
	merged := make([]*recallHit, 0, len(hitMap))
	// 先收集向量命中的（保持原始顺序）
	vecSeen := make(map[string]bool)
	for _, h := range vecHits {
		k := keyOf(h.NoteID, h.ChunkIndex)
		if !vecSeen[k] {
			vecSeen[k] = true
			merged = append(merged, hitMap[k])
		}
	}
	// 再收集仅关键词命中的
	for _, h := range kwHits {
		k := keyOf(h.NoteID, h.ChunkIndex)
		if !vecSeen[k] {
			vecSeen[k] = true
			merged = append(merged, hitMap[k])
		}
	}

	// 按优先级稳定排序：双命中(3) > 仅向量(1) > 仅关键词(2)
	// 同优先级内仅关键词块按 kwScore 降序
	sortHybridHits(merged)

	// 按笔记聚合选择：保留前 limit 个不同笔记的块，同一笔记最多 maxChunksPerNote 个
	// 替代原按 chunk 截断（merged[:limit]），避免多个命中块来自同一笔记时卡片过少、多样性差
	mergedVecs := make([]models.NoteVector, 0, len(merged))
	for _, h := range merged {
		mergedVecs = append(mergedVecs, h.vec)
	}
	hits := selectTopNotes(mergedVecs, limit, maxChunksPerNote)

	// ===== 以下为相邻块扩展 + 卡片组装（复用原 VectorRecall 逻辑） =====

	// 按命中 note_id 批量查询笔记元信息
	noteIDs := make([]uint, 0, len(hits))
	for _, h := range hits {
		noteIDs = append(noteIDs, h.NoteID)
	}
	var notes []models.Note
	if err := s.db.WithContext(ctx).Where("id IN ?", noteIDs).Find(&notes).Error; err != nil {
		s.logger.Errorw("VectorService.HybridRecall 查询笔记失败", fastlog.Error(err))
		return nil, fmt.Errorf("查询命中笔记失败: %w", err)
	}
	noteMeta := make(map[uint]models.Note, len(notes))
	for _, n := range notes {
		noteMeta[n.ID] = n
	}

	// 二次查询命中笔记的全部块（按 ChunkIndex 升序）
	// 注意：不按 model 过滤——混合召回中关键词命中可能跨模型
	var blocks []models.NoteVector
	if err := s.db.WithContext(ctx).Where("note_id IN ?", noteIDs).
		Order("chunk_index ASC").Find(&blocks).Error; err != nil {
		s.logger.Errorw("VectorService.HybridRecall 查询命中笔记块失败", fastlog.Error(err))
		return nil, fmt.Errorf("查询命中笔记块失败: %w", err)
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

	// 命中笔记顺序按 hits 首次出现先后（= 排序后顺序）
	hitOrder := make([]uint, 0, len(hitIndexes))
	seenNote := make(map[uint]bool, len(hitIndexes))
	for _, h := range hits {
		if !seenNote[h.NoteID] {
			seenNote[h.NoteID] = true
			hitOrder = append(hitOrder, h.NoteID)
		}
	}

	// 组装格式化上下文文本与结构化卡片
	var b strings.Builder
	b.WriteString("以下是用户笔记库中与问题相关的笔记片段（来源：本地笔记混合检索），请优先参考这些笔记内容回答用户的问题：\n\n")

	cards := make([]RecallCard, 0, len(hitOrder))
	for _, noteID := range hitOrder {
		note, ok := noteMeta[noteID]
		if !ok {
			continue
		}
		var parts []string
		for _, v := range byNote[noteID] {
			if hitIndexes[noteID][v.ChunkIndex] {
				parts = append(parts, stripMetaPrefix(v.ChunkText))
			}
		}
		content := strings.Join(parts, "\n\n")
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
		s.logger.Debugw("VectorService.HybridRecall 组装卡片为空")
		return nil, nil
	}
	b.WriteString("请基于以上笔记内容回答用户的问题。如果笔记内容不足以回答，请如实说明。")

	// 统计双命中/向量/关键词各路命中数
	dualCnt, vecOnlyCnt, kwOnlyCnt := 0, 0, 0
	for _, h := range merged {
		switch h.sources {
		case 3:
			dualCnt++
		case 1:
			vecOnlyCnt++
		case 2:
			kwOnlyCnt++
		}
	}
	s.logger.Debugw("VectorService.HybridRecall 命中",
		fastlog.Int("cards_count", len(cards)),
		fastlog.Int("dual_hits", dualCnt),
		fastlog.Int("vec_only_hits", vecOnlyCnt),
		fastlog.Int("kw_only_hits", kwOnlyCnt),
		fastlog.String("query", query))

	return &CardRecallResult{
		FormattedText: b.String(),
		Cards:         cards,
	}, nil
}

// sortHybridHits 按命中优先级稳定排序：双命中(3) > 仅向量(1) > 仅关键词(2)
// 同优先级内：仅关键词块按 kwScore 降序（命中 token 数越多越靠前）；其余保持原始顺序
func sortHybridHits(hits []*recallHit) {
	// 优先级映射：sources 3→0(最高), 1→1, 2→2
	priority := func(sources int) int {
		switch sources {
		case 3:
			return 0
		case 1:
			return 1
		default:
			return 2
		}
	}
	// 稳定排序：按优先级分组，同优先级内仅关键词块按 kwScore 降序
	// 使用插入排序（数据量小，通常 < 2*limit）
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0; j-- {
			pj, pj1 := priority(hits[j].sources), priority(hits[j-1].sources)
			if pj < pj1 {
				hits[j], hits[j-1] = hits[j-1], hits[j]
			} else if pj == pj1 && pj == 2 && hits[j].kwScore > hits[j-1].kwScore {
				// 同为仅关键词命中时，按 kwScore 降序
				hits[j], hits[j-1] = hits[j-1], hits[j]
			} else {
				break
			}
		}
	}
}

// selectTopNotes 按笔记聚合选择命中块：hits 已按相关度（距离升序/优先级）排序，
// 保留前 limit 个不同笔记的块，同一笔记最多保留 maxPerNote 个块
// 避免按 chunk 级截断导致的笔记多样性差（多个命中块来自同一笔记时卡片过少）
// 返回的新切片保持传入顺序（即相关度顺序）
func selectTopNotes(hits []models.NoteVector, limit, maxPerNote int) []models.NoteVector {
	if limit <= 0 {
		return nil
	}
	if maxPerNote <= 0 {
		maxPerNote = 1
	}
	noteCnt := make(map[uint]int, limit)
	out := make([]models.NoteVector, 0, len(hits))
	for _, h := range hits {
		cnt := noteCnt[h.NoteID]
		if cnt == 0 {
			// 新笔记：已集满 limit 个则跳过
			if len(noteCnt) >= limit {
				continue
			}
		} else if cnt >= maxPerNote {
			// 已有笔记：达到块数上限则跳过该笔记后续命中
			continue
		}
		noteCnt[h.NoteID] = cnt + 1
		out = append(out, h)
	}
	return out
}

// VectorRecall 卡片召回（对外接口，签名不变）
// 内部委托 HybridRecall 执行向量+关键词混合检索
// 当 embedClient 为 nil 或模型无数据时，仍可仅走关键词检索路
//
// 返回分类：
//   - (result, nil)：召回成功
//   - (nil, nil)：预期跳过（query 为空/无命中/卡片为空），调用方可静默
//   - (nil, err)：意外错误，调用方应提示用户
func (s *VectorService) VectorRecall(ctx context.Context, query string, limit int, embedClient *einocli.Client, notebookIDs ...uint) (*CardRecallResult, error) {
	return s.HybridRecall(ctx, query, limit, embedClient, notebookIDs...)
}
