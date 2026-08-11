// vec-poc 是"向量召回 CLI 测试工具"：
// 验证 SQLite 向量检索 + Ollama 本地 embedding + 互联网模型(OpenAI 兼容)对话的技术路线。
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vec-poc/internal/chunk"
	"vec-poc/internal/config"
	"vec-poc/internal/embed"
	"vec-poc/internal/llm"
	"vec-poc/internal/store"
)

// app 聚合运行期依赖，供各子命令与交互循环复用。
type app struct {
	cfg      *config.Config
	st       store.VectorStore
	embedder store.Embedder // 统一的向量化闭包（内部调用 Ollama）
}

func main() {
	os.Exit(run())
}

// run 解析配置、初始化数据库与检索实现，分发子命令；返回进程退出码。
func run() int {
	cfg := config.Parse()
	ctx := context.Background()

	// 打开数据库并自动建表
	db, err := store.OpenDB(cfg.DBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		return 1
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	// 探测 sqlite-vec 扩展，决定检索实现
	vecVersion, probeErr := store.ProbeVec(db)
	useVec := probeErr == nil && !cfg.ForceBrute

	// 打印启动横幅
	printBanner(cfg, vecVersion, probeErr, useVec)

	st := store.NewStore(db, cfg.EmbedModel, useVec)
	emb := embed.NewClient(cfg.OllamaURL, cfg.EmbedModel)
	// 块级进度：每完成一批向量化刷新同一行进度条，并展示本批耗时与单块平均耗时。
	emb.OnBatch = func(p embed.BatchProgress) {
		fmt.Printf("\r  向量化中 %d/%d 块 | 本批 %d 块耗时 %v | 单块均耗 %v",
			p.Done, p.Total, p.BatchSize, p.BatchCost.Round(time.Millisecond), p.AvgPerItem.Round(time.Microsecond))
	}
	embedder := func(texts []string) ([][]float32, error) {
		return emb.Embed(ctx, texts)
	}
	a := &app{cfg: cfg, st: st, embedder: embedder}

	// 子命令模式：os.Args 中第一个非 flag 参数
	args := flag.Args()
	if len(args) == 0 {
		return a.repl(ctx)
	}

	var cmdErr error
	switch args[0] {
	case "add":
		cmdErr = a.cmdAdd(ctx, args[1:])
	case "index":
		cmdErr = a.cmdIndex(ctx)
	case "ask":
		cmdErr = a.cmdAsk(ctx, args[1:])
	case "list":
		cmdErr = a.cmdList(ctx)
	case "status":
		cmdErr = a.cmdStatus(ctx)
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %q\n", args[0])
		printHelp()
		return 1
	}
	if cmdErr != nil {
		fmt.Fprintln(os.Stderr, "错误:", cmdErr)
		return 1
	}
	return 0
}

// printBanner 打印模块名、sqlite-vec 探针结果与配置摘要。
func printBanner(cfg *config.Config, vecVersion string, probeErr error, useVec bool) {
	fmt.Println("vec-poc 向量召回 POC")
	fmt.Printf("  数据库: %s\n", cfg.DBPath)
	if probeErr != nil {
		fmt.Printf("  sqlite-vec 探针: 不可用 (%v)\n", probeErr)
	} else {
		fmt.Printf("  sqlite-vec 探针: %s (可用)\n", vecVersion)
	}
	if cfg.ForceBrute {
		fmt.Println("  检索实现: pure-go-brute (--force-brute)")
	} else if useVec {
		fmt.Println("  检索实现: sqlite-vec")
	} else {
		fmt.Println("  检索实现: pure-go-brute (回退)")
	}
	fmt.Printf("  embedding: %s @ %s\n", cfg.EmbedModel, cfg.OllamaURL)
	if cfg.LLMModel != "" {
		fmt.Printf("  LLM: %s @ %s\n", cfg.LLMModel, cfg.LLMBaseURL)
	} else {
		fmt.Println("  LLM: 未配置（ask 命令不可用，可设置 --llm-model 等）")
	}
}

// cmdAdd 添加单个文件或目录（目录递归收集 .md/.txt 文本文件）。
func (a *app) cmdAdd(ctx context.Context, args []string) error {
	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("用法: add <文件或目录>")
	}
	path := args[0]
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("无法访问 %s: %w", path, err)
	}

	var files []string
	if info.IsDir() {
		err = filepath.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			lower := strings.ToLower(p)
			if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".txt") {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("遍历目录 %s 失败: %w", path, err)
		}
	} else {
		files = []string{path}
	}
	if len(files) == 0 {
		return fmt.Errorf("未找到可处理的 .md/.txt 文本文件")
	}

	total := 0
	for i, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "跳过 %s: %v\n", f, err)
			continue
		}
		chunks := chunk.ChunkDefault(string(content))
		if len(chunks) == 0 {
			fmt.Printf("跳过 %s: 无可切分内容\n", f)
			continue
		}
		start := time.Now()
		fmt.Printf("正在处理 [%d/%d] %s（%d 块）...\n", i+1, len(files), filepath.Base(f), len(chunks))
		n, err := a.st.AddDocument(ctx, filepath.Base(f), f, string(content), chunks, a.embedder)
		clearLine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "添加 %s 失败: %v\n", f, err)
			continue
		}
		fmt.Printf("  完成 %s：%d 个块（耗时 %s）\n", filepath.Base(f), n, time.Since(start).Round(time.Millisecond))
		total += n
	}
	fmt.Printf("共添加 %d 个块\n", total)
	return nil
}

// cmdIndex 清空全部 chunks 后对所有文档重建向量索引，期间展示文档级进度。
func (a *app) cmdIndex(ctx context.Context) error {
	start := time.Now()
	a.st.SetProgress(func(done, total int, msg string) {
		fmt.Printf("\r  文档 %d/%d: %s", done, total, msg)
	})
	n, err := a.st.Rebuild(ctx, a.embedder)
	if err != nil {
		return err
	}
	clearLine()
	fmt.Printf("重建完成，共 %d 个块（耗时 %s）\n", n, time.Since(start).Round(time.Millisecond))
	return nil
}

// cmdAsk 向量召回相关知识并调用 LLM 回答；召回为空时直接询问 LLM。
func (a *app) cmdAsk(ctx context.Context, args []string) error {
	if err := a.cfg.Validate(); err != nil {
		return err
	}
	question := strings.Join(args, " ")
	if question == "" {
		return fmt.Errorf("用法: ask <问题>")
	}

	// 问题向量化（复用统一 embedder 闭包）
	vecs, err := a.embedder([]string{question})
	if err != nil {
		return fmt.Errorf("问题向量化失败: %w", err)
	}
	if len(vecs) == 0 {
		return fmt.Errorf("问题向量化返回空结果")
	}

	// 向量召回
	hits, err := a.st.Search(ctx, vecs[0], a.cfg.TopK)
	if err != nil {
		return fmt.Errorf("向量检索失败: %w", err)
	}

	client := llm.NewClient(a.cfg.LLMBaseURL, a.cfg.LLMAPIKey, a.cfg.LLMModel)
	if len(hits) == 0 {
		fmt.Println("未召回相关知识")
		answer, err := client.Chat(ctx, "", question)
		if err != nil {
			return err
		}
		fmt.Println("回答:")
		fmt.Println(answer)
		return nil
	}

	// 打印召回块（来源、距离、文本前 80 字）
	fmt.Printf("召回 %d 个相关片段:\n", len(hits))
	for i, h := range hits {
		fmt.Printf("[%d] 来源=%s 块=%d 距离=%.4f\n    %s\n", i+1, h.DocName, h.ChunkIndex, h.Distance, previewText(h.Text, 80))
	}

	// 组装 system 提示并调用 LLM
	prompt := llm.BuildRecallPrompt(hits)
	answer, err := client.Chat(ctx, prompt, question)
	if err != nil {
		return err
	}
	fmt.Println("回答:")
	fmt.Println(answer)
	return nil
}

// cmdList 列出全部文档。
func (a *app) cmdList(ctx context.Context) error {
	docs, err := a.st.ListDocs(ctx)
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		fmt.Println("暂无文档，使用 add <文件> 添加。")
		return nil
	}
	fmt.Printf("共 %d 个文档:\n", len(docs))
	for _, d := range docs {
		fmt.Printf("  [%d] %s (%s)\n", d.ID, d.Name, d.SourcePath)
	}
	return nil
}

// cmdStatus 显示检索实现与文档/块统计。
func (a *app) cmdStatus(ctx context.Context) error {
	impl, docCount, chunkCount, err := a.st.Status(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("检索实现: %s\n文档数: %d\n块数: %d\n", impl, docCount, chunkCount)
	return nil
}

// repl 交互式命令行：逐行读取 stdin，命令出错打印错误但不退出；quit/EOF 退出。
func (a *app) repl(ctx context.Context) int {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("进入交互模式，输入 help 查看命令，quit 退出。")
	for {
		fmt.Print("vec-poc> ")
		if !scanner.Scan() {
			break // EOF
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// 拆出命令与剩余原始参数（问题可能含空格，需保留原文）
		cmd, rest, _ := strings.Cut(line, " ")
		rest = strings.TrimSpace(rest)

		var err error
		switch cmd {
		case "add":
			err = a.cmdAdd(ctx, []string{rest})
		case "index":
			err = a.cmdIndex(ctx)
		case "ask":
			err = a.cmdAsk(ctx, []string{rest})
		case "list":
			err = a.cmdList(ctx)
		case "status":
			err = a.cmdStatus(ctx)
		case "help":
			printHelp()
		case "quit", "exit":
			fmt.Println("再见")
			return 0
		default:
			fmt.Printf("未知命令 %q，输入 help 查看帮助\n", cmd)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
		}
	}
	fmt.Println("再见")
	return 0
}

// clearLine 清除当前行的进度条残留（\r 回车 + ANSI 清行）。
func clearLine() {
	fmt.Print("\r\033[K")
}

// previewText 将文本截取为前 n 个 rune 的预览（多字节安全）。
func previewText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

// printHelp 打印使用帮助。
func printHelp() {
	fmt.Print(`用法: vec-poc [flags] <子命令> [参数]

子命令:
  add <文件或目录>   添加文档到知识库（目录会递归收集 .md/.txt）
  index             清空并重建全部文档的向量索引
  ask <问题>         向量召回相关知识并调用 LLM 回答
  list              列出全部文档
  status            显示检索实现与文档/块统计
  help              显示本帮助
  quit              退出交互模式（仅 REPL 使用）

flags:
  --db <路径>            SQLite 数据库文件（默认 ./vec-test.db，环境变量 VEC_DB）
  --ollama-url <地址>    Ollama 服务地址（默认 http://localhost:11434，环境变量 VEC_OLLAMA_URL）
  --embed-model <模型>   embedding 模型名（默认 bge-m3，环境变量 VEC_EMBED_MODEL）
  --llm-base-url <地址>  OpenAI 兼容 LLM 服务地址（环境变量 VEC_LLM_BASE_URL）
  --llm-api-key <Key>    OpenAI 兼容 LLM API Key（环境变量 VEC_LLM_API_KEY）
  --llm-model <模型>     OpenAI 兼容 LLM 模型名（环境变量 VEC_LLM_MODEL）
  --topk <N>             召回数量（默认 5，环境变量 VEC_TOP_K）
  --force-brute          强制使用纯 Go 暴力检索实现
`)
}
