# 替换卡片召回分词器为 gse Spec

## Why
将当前自定义 2-gram 分词器替换为 go-ego/gse 库，利用其成熟的词典+最短路径/HMM 分词算法，获得更精准的词级切分。保留停用词过滤作为后处理。

## What Changes
- 在 `go.mod` 中添加 `github.com/go-ego/gse` 依赖
- 删除 `tokenize2Gram()`、`isCJK()`、`splitWords()` 三个函数
- 保留 `stopWords` map 和 `isStopWord()` 函数
- 新增基于 gse 的 tokenizer（含懒初始化），替换 `tokenize2Gram` 的调用点
- 更新 `CardRecallSearch` 的注释

## Impact
- Affected specs: 卡片召回
- Affected code: `internal/services/recall_service.go`, `go.mod`/`go.sum`
- 无 breaking changes：输入输出 `func(text string) []string` 签名不变，调用方不变
- 无前端或数据库变更

## ADDED Requirements

### Requirement: gse 分词集成
The system SHALL use gse segmenter for card recall query tokenization.

#### Scenario: 分词结果更精准
- **WHEN** 用户输入搜索 query
- **THEN** gse 输出词级 token（而非 2-gram 字符级），保留停用词过滤后作为关键词

## MODIFIED Requirements

### Requirement: 卡片召回分词
- 原 `tokenize2Gram(text string) []string` 删除
- 新增 `gseTokenizer`（包级全局 Segmenter + sync.Once 懒初始化）+ `tokenize(text string) []string` 函数
- 保留 `stopWords` 和 `isStopWord` 做分词后过滤

## REMOVED Requirements

### Requirement: tokenize2Gram / isCJK / splitWords
**Reason**: 被 gse 替代
**Migration**: 删除三个函数

## Implementation Details

### gse 使用方式
```go
import "github.com/go-ego/gse"

var (
    gseSeg    gse.Segmenter
    gseOnce   sync.Once
    gseInitErr error
)

func initGse() {
    gseSeg = gse.Segmenter{}
    gseInitErr = gseSeg.LoadDict()
}

func tokenize(text string) []string {
    gseOnce.Do(initGse)
    if gseInitErr != nil {
        return nil
    }
    words := gseSeg.Cut(text, true) // precise mode + HMM
    // 去重 + 停用词过滤
    seen := make(map[string]struct{})
    var result []string
    for _, w := range words {
        if _, ok := seen[w]; !ok && !isStopWord([]rune(w)...) {
            seen[w] = struct{}{}
            result = append(result, w)
        }
    }
    return result
}
```
