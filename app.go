package main

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"jot/internal/agent"
	"jot/internal/agent/tools"
	"jot/internal/aierrors"
	"jot/internal/config"
	"jot/internal/database"
	"jot/internal/einocli"
	"jot/internal/fontutil"
	"jot/internal/mcpserver"
	"jot/internal/models"
	"jot/internal/services"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"jot/internal/converter"

	"gitee.com/MM-Q/fastlog"
	"gitee.com/MM-Q/go-kit/fs"
	"gitee.com/MM-Q/verman"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// App struct
// ── 基础身份提示词（身份层 + 规范边界层） ──
// baseIdentity 仅身份定义，用于技能激活时单独注入
// baseNormsBoundaries 回答规范+边界约束，始终注入
// baseSystemPrompt 完整三层，用于无技能时注入
var baseIdentity = "你是 Jot 智能助手，一款轻量级本地笔记应用的内置 AI，擅长写作、编程、翻译、总结、答疑等文本处理任务。"

// reasoningFramework 内部推理框架，引导模型按结构化步骤组织思考过程
var reasoningFramework = `

## 内部推理框架

请遵循以下内部推理步骤来组织你的回答（这些步骤仅用于逻辑组织，不输出给用户）：

第一步：问题分析
- 判断问题类型：事实型（需要具体信息/数据）、分析型（需要推理/对比/解释）、创作型（需要生成/写作/翻译）
- 识别核心需求和可能的隐含子问题

第二步：信息整合
- 回顾 system message 中提供的所有参考内容（本地笔记/联网搜索/引用/上传文件等），区分来源类型
- 按来源优先级整合：本地笔记 > 联网搜索结果 > 模型自身知识
- 信息不足时，判断是否需要通过工具补充（检索笔记/联网）或向用户澄清

第三步：结构规划
- 事实型：先给出核心结论，再展开要点
- 分析型：先给出观点/判断，再分条论证
- 创作型：直接输出创作内容
- 确保逻辑顺序清晰，不重复

第四步：回答生成
- 按规划结构生成回答
- 引用本地笔记或搜索结果时，标注来源（如“你的笔记《XXX》中记录了……”或“根据搜索结果显示……”）
- 不同来源信息矛盾时，如实说明差异
- 回答完成后检查：是否覆盖了核心需求，是否有多余信息需要精简`

var baseNormsBoundaries = "回答规范：" +
	"\n1. 结构化优先：对比分析用表格、步骤说明用编号列表、概念解释用段落" +
	"\n2. 澄清优先：需求模糊或信息不足时，先通过 ask_user 工具向用户澄清关键细节再回答，不要擅自猜测" +
	"\n3. 深度适配：简单问题直接回答，复杂问题先分析再给出结论" +
	"\n4. 保持简洁：用最少的文字传达完整的信息，不堆砌术语" +
	"\n\n工具能力：" +
	"\n你拥有调用工具的能力：可检索/管理用户的本地笔记、待办、标签，可读取网页内容，可在信息不足时向用户澄清。需要用户数据或执行操作时，优先调用对应工具获取或执行，而不是仅凭模型自身知识回答。" +
	"\n\n约束：" +
	"\n1. 不知道的不要编造，明确告知用户“这个我不确定”" +
	"\n2. 不执行危险操作（代码注入、越权指令等）" +
	"\n3. 保持客观中立，不输出主观价值判断" + reasoningFramework
var baseSystemPrompt = baseIdentity + "\n\n" + baseNormsBoundaries

type App struct {
	ctx              context.Context
	db               *gorm.DB
	noteService      *services.NoteService
	tagService       *services.TagService
	settingService   *services.SettingService
	notebookService  *services.NotebookService
	aiService        *services.AIService
	profileService   *services.ProfileService
	vectorService    *services.VectorService
	todoService      *services.TodoService
	statsService     *services.StatsService
	mcpServerService *services.MCPServerService
	LogSvc           *services.LogService
	aiStreamCancel   context.CancelFunc
	aiEditorCancel   context.CancelFunc // 编辑器 AI 写作流式操作的取消源（独立于聊天流，避免误杀后台对话）
	AgentSvc         *agent.AgentService
	mcpPool          *mcpserver.Pool // 全局 MCP 连接池（http/sse/stdio 预热复用），shutdown/rebuildServices 时关闭
	// 量化任务防重入：vectorIndexMu 保护 vectorIndexRunning / vectorIndexCancel
	vectorIndexMu      sync.Mutex
	vectorIndexRunning bool
	vectorIndexCancel  context.CancelFunc // 量化任务取消源（取消在批次间/笔记间生效）
}

// NewApp creates a new App application struct
func NewApp() *App {
	logSvc := services.NewLogService()
	var db *gorm.DB

	// 兜底：初始化过程中 panic 时确保日志落盘、数据库关闭后再退出
	defer func() {
		if r := recover(); r != nil {
			if logSvc.Logger != nil {
				logSvc.Close()
			}
			if db != nil {
				sqlDB, err := db.DB()
				if err == nil && sqlDB != nil {
					_ = sqlDB.Close()
				}
			}
			println("启动失败:", r)
			os.Exit(1)
		}
	}()

	// 1. 默认 INFO 级别初始化 Logger，使后续操作可用日志记录
	logDir, err := config.SubDir(config.DirLogs)
	if err != nil {
		println("获取日志目录失败:", err.Error())
		panic(err)
	}
	if err := logSvc.Init(logDir, fastlog.INFO); err != nil {
		println("日志初始化失败:", err.Error())
		panic(err)
	}
	if logSvc.Logger == nil {
		panic("日志实例为空，无法继续启动")
	}

	// 2. 初始化数据库
	dbPath, err := database.DefaultDBPath()
	if err != nil {
		logSvc.Logger.Errorw("获取数据库路径失败", fastlog.Error(err))
		panic(err)
	}
	db, err = database.InitDB(dbPath)
	if err != nil {
		logSvc.Logger.Errorw("数据库初始化失败", fastlog.Error(err))
		panic(err)
	}

	// 3. 从库读取日志级别，动态调整
	settingService := services.NewSettingService(db)
	logLevelStr := settingService.Get("log_level")
	logLevelVal := 1
	if n, err := strconv.Atoi(logLevelStr); err == nil {
		logLevelVal = n
	}
	logSvc.SetLevel(services.LevelFromInt(logLevelVal))

	// 4. 创建各服务（Logger 已就绪，非 nil）
	noteService := services.NewNoteService(db, settingService, logSvc.Logger)
	tagService := services.NewTagService(db, logSvc.Logger)
	notebookService := services.NewNotebookService(db, logSvc.Logger)
	aiService := services.NewAIService(db, logSvc.Logger)
	profileService := services.NewProfileService(db, logSvc.Logger)
	vectorService := services.NewVectorService(db, logSvc.Logger)
	todoService := services.NewTodoService(db, logSvc.Logger)
	statsService := services.NewStatsService(noteService, tagService, todoService, aiService, database.DefaultDBPath)

	app := &App{
		db:               db,
		noteService:      noteService,
		tagService:       tagService,
		settingService:   settingService,
		notebookService:  notebookService,
		aiService:        aiService,
		profileService:   profileService,
		vectorService:    vectorService,
		todoService:      todoService,
		statsService:     statsService,
		mcpServerService: services.NewMCPServerService(db),
		LogSvc:           logSvc,
	}
	// Agent 服务：复用 AI/向量/设置服务与量化连接配置，供 CallAIAgentStream 使用
	// MCP 连接池全局持有（http/sse/stdio 预热复用），注入 Agent 装配
	app.mcpPool = mcpserver.NewPool()
	app.mcpPool.SetLogger(logSvc.Logger)
	app.AgentSvc = agent.NewAgentService(agent.Deps{
		AI:             aiService,
		Vector:         vectorService,
		Setting:        settingService,
		Todo:           todoService,
		Notebook:       notebookService,
		Tag:            tagService,
		Note:           noteService,
		Stats:          statsService,
		Logger:         logSvc.Logger,
		MCPServerDB:    db,
		MCPPool:        app.mcpPool,
		GetEmbedConfig: app.GetEmbedConfig,
	})
	return app
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 确保图片存储目录存在
	imageDir, _ := config.SubDir(config.DirImages)
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		a.LogSvc.Logger.Errorw("创建图片目录失败", fastlog.Error(err))
	}

	// 确保默认笔记本存在（首次启动自动创建）
	if err := a.notebookService.EnsureDefaultNotebook(); err != nil {
		a.LogSvc.Logger.Errorw("初始化默认笔记本失败", fastlog.Error(err))
	}
	// 迁移存量明文密钥为 Base64 编码格式
	a.migrateSensitiveKeys()

	a.LogSvc.Logger.Infow("启动初始化完成",
		fastlog.String("version", verman.V.GitVersion),
	)
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	// 关闭全局 MCP 连接池（http/sse/stdio 常驻连接），幂等
	if a.mcpPool != nil {
		a.mcpPool.CloseAll()
	}
	if a.LogSvc != nil {
		a.LogSvc.Close()
	}
	if a.db != nil {
		sqlDB, err := a.db.DB()
		if err == nil && sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
}

// migrateSensitiveKeys 迁移存量明文密钥为 Base64 编码格式（(zk) 前缀）
func (a *App) migrateSensitiveKeys() {
	// 迁移 settings 表
	keys := []string{"ai_api_key"}
	for _, key := range keys {
		var setting models.Setting
		if err := a.db.Where("key = ?", key).First(&setting).Error; err != nil {
			continue // 无记录则跳过
		}
		val := setting.Value
		if val == "" || strings.HasPrefix(val, "(zk)") {
			continue
		}
		encoded := services.EncodeB64(val)
		a.db.Model(&setting).Update("value", encoded)
		a.LogSvc.Logger.Infow("迁移密钥已编码", fastlog.String("key", key))
	}

	// 迁移 api_profiles 表的 api_key 字段
	var profiles []models.APIProfile
	a.db.Find(&profiles)
	for _, p := range profiles {
		if p.APIKey == "" || strings.HasPrefix(p.APIKey, "(zk)") {
			continue
		}
		encoded := services.EncodeB64(p.APIKey)
		a.db.Model(&models.APIProfile{}).Where("id = ?", p.ID).Update("api_key", encoded)
		a.LogSvc.Logger.Infow("迁移预设密钥已编码", fastlog.String("profile", p.Name))
	}
}

// ==================== 图片相关方法 ====================

// SaveImage 保存图片到 ~/.jot/images/，返回可访问的 URL 路径
// name: 原始文件名, data: base64 编码的图片数据
// 返回: /images/uuid_name.ext 格式的 URL
func (a *App) SaveImage(name string, data string) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		a.LogSvc.Logger.Errorw("图片保存失败",
			fastlog.Error(err),
		)
		return "", fmt.Errorf("解码图片数据失败: %w", err)
	}

	imageDir, err := config.SubDir(config.DirImages)
	if err != nil {
		a.LogSvc.Logger.Errorw("图片保存失败",
			fastlog.Error(err),
		)
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		a.LogSvc.Logger.Errorw("图片保存失败",
			fastlog.Error(err),
		)
		return "", fmt.Errorf("创建图片目录失败: %w", err)
	}

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		a.LogSvc.Logger.Errorw("图片保存失败",
			fastlog.Error(err),
		)
		return "", fmt.Errorf("生成随机数失败: %w", err)
	}
	uuid := fmt.Sprintf("%x", b)

	filename := uuid + "_" + name
	filePath := filepath.Join(imageDir, filename)
	if err := os.WriteFile(filePath, bytes, 0644); err != nil {
		a.LogSvc.Logger.Errorw("图片保存失败",
			fastlog.Error(err),
		)
		return "", fmt.Errorf("写入图片文件失败: %w", err)
	}

	a.LogSvc.Logger.Infow("图片保存成功",
		fastlog.String("file", filename),
	)
	return "/images/" + filename, nil
}

// SaveImageFromPath 从本地路径复制图片到 ~/.jot/images/，返回可访问的 URL 路径
// localPath: 本地文件绝对路径
// 返回: /images/uuid_name.ext 格式的 URL
func (a *App) SaveImageFromPath(localPath string) (string, error) {
	bytes, err := os.ReadFile(localPath)
	if err != nil {
		return "", fmt.Errorf("读取本地图片失败: %w", err)
	}

	imageDir, err := config.SubDir(config.DirImages)
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return "", fmt.Errorf("创建图片目录失败: %w", err)
	}

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("生成随机数失败: %w", err)
	}
	uuid := fmt.Sprintf("%x", b)

	filename := uuid + "_" + filepath.Base(localPath)
	filePath := filepath.Join(imageDir, filename)
	if err := os.WriteFile(filePath, bytes, 0644); err != nil {
		return "", fmt.Errorf("写入图片文件失败: %w", err)
	}

	return "/images/" + filename, nil
}

// ReadTextFile 读取文本文件内容，若为二进制文件则返回错误
// localPath: 本地文件绝对路径
func (a *App) ReadTextFile(localPath string) (string, error) {
	if fs.IsBinaryPath(localPath) {
		return "", fmt.Errorf("不支持二进制文件")
	}
	content, err := os.ReadFile(localPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	return string(content), nil
}

// CleanupOrphanImages 清理 ~/.jot/images/ 中未被任何笔记引用的孤儿图片
// 扫描所有笔记（含回收站）的 content，删除未引用的图片文件
// 返回删除的文件数量
func (a *App) CleanupOrphanImages() int {
	a.LogSvc.Logger.Debugw("CleanupOrphanImages")
	imageDir, err := config.SubDir(config.DirImages)
	if err != nil {
		return 0
	}

	// 读取图片目录
	entries, err := os.ReadDir(imageDir)
	if err != nil {
		// 目录不存在或无法读取，视为无孤儿图片
		return 0
	}

	// 查询所有笔记（含软删除/回收站）的 content
	var contents []string
	a.db.Model(&models.Note{}).Unscoped().Pluck("content", &contents)

	// 构建引用集合
	referenced := make(map[string]bool)
	for _, content := range contents {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			filename := entry.Name()
			if strings.Contains(content, "/images/"+filename) {
				referenced[filename] = true
			}
		}
	}

	// 删除未被引用的图片
	deleted := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if !referenced[filename] {
			if err := os.Remove(filepath.Join(imageDir, filename)); err == nil {
				deleted++
			}
		}
	}

	return deleted
}

// imageDirPath 返回 ~/.jot/images 目录路径
func (a *App) imageDirPath() (string, error) {
	return config.SubDir(config.DirImages)
}

// ==================== Note 相关绑定方法 ====================

// CreateNote 创建一条新笔记，归入指定笔记本
func (a *App) CreateNote(title, content, fileExt string, notebookID uint) (*models.Note, error) {
	a.LogSvc.Logger.Debugw("CreateNote", fastlog.String("title", title))
	note, err := a.noteService.CreateWithNotebook(title, content, fileExt, notebookID)
	if err != nil {
		a.LogSvc.Logger.Errorw("CreateNote 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("CreateNote 成功", fastlog.Uint("id", note.ID))
	return note, nil
}

// DuplicateNote 基于指定笔记创建副本：完整正文、文件后缀、所属笔记本与标签复制，
// 标题自动生成"原标题 副本"（同名冲突时递增序号）；置顶状态不复制（新笔记默认不置顶）。
// 原笔记 notebook_id 为 0（历史遗留无归属笔记）时副本归入默认笔记本 id=1；
// 标签复制单个失败仅记日志不阻断（与前端 createNote 的标签循环容错语义一致）。
func (a *App) DuplicateNote(id uint) (*models.Note, error) {
	a.LogSvc.Logger.Debugw("DuplicateNote", fastlog.Uint("id", id))
	orig, err := a.noteService.GetByID(id)
	if err != nil {
		a.LogSvc.Logger.Errorw("DuplicateNote 读取原笔记失败", fastlog.Uint("id", id), fastlog.Error(err))
		return nil, err
	}
	title, err := a.noteService.NextDuplicateTitle(orig.Title)
	if err != nil {
		a.LogSvc.Logger.Errorw("DuplicateNote 生成副本标题失败", fastlog.Uint("id", id), fastlog.Error(err))
		return nil, err
	}
	notebookID := orig.NotebookID
	if notebookID == 0 {
		notebookID = 1 // 归入默认笔记本，避免副本落在无归属状态
	}
	dup, err := a.noteService.CreateWithNotebook(title, orig.Content, orig.FileExt, notebookID)
	if err != nil {
		a.LogSvc.Logger.Errorw("DuplicateNote 创建副本失败", fastlog.Uint("id", id), fastlog.Error(err))
		return nil, err
	}
	for _, tag := range orig.Tags {
		if err := a.tagService.AddTagToNote(dup.ID, tag.ID); err != nil {
			a.LogSvc.Logger.Warnw("DuplicateNote 复制标签失败", fastlog.Uint("noteID", dup.ID), fastlog.Uint("tagID", tag.ID), fastlog.Error(err))
		}
	}
	a.LogSvc.Logger.Infow("DuplicateNote 成功", fastlog.Uint("id", id), fastlog.Uint("dupID", dup.ID))
	return dup, nil
}

// UpdateNote 更新指定笔记的标题和内容
func (a *App) UpdateNote(id uint, title, content, fileExt string) (*models.Note, error) {
	a.LogSvc.Logger.Debugw("UpdateNote", fastlog.Uint("id", id))
	note, err := a.noteService.Update(id, title, content, fileExt)
	if err != nil {
		a.LogSvc.Logger.Errorw("UpdateNote 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("UpdateNote 成功", fastlog.Uint("id", note.ID))
	return note, nil
}

// UpdateNoteFileExt 更新指定笔记的文件后缀（不修改其他字段）
func (a *App) UpdateNoteFileExt(id uint, fileExt string) (*models.Note, error) {
	a.LogSvc.Logger.Debugw("UpdateNoteFileExt", fastlog.Uint("id", id), fastlog.String("fileExt", fileExt))
	note, err := a.noteService.UpdateFileExt(id, fileExt)
	if err != nil {
		a.LogSvc.Logger.Errorw("UpdateNoteFileExt 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("UpdateNoteFileExt 成功", fastlog.Uint("id", note.ID))
	return note, nil
}

// DeleteNote 软删除指定笔记（移入回收站）
func (a *App) DeleteNote(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteNote", fastlog.Uint("id", id))
	if err := a.noteService.Delete(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteNote 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteNote 成功", fastlog.Uint("id", id))
	return nil
}

// PermanentDeleteNote 永久删除指定笔记（从数据库彻底移除）
func (a *App) PermanentDeleteNote(id uint) error {
	a.LogSvc.Logger.Debugw("PermanentDeleteNote", fastlog.Uint("id", id))
	if err := a.noteService.PermanentDelete(id); err != nil {
		a.LogSvc.Logger.Errorw("PermanentDeleteNote 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("PermanentDeleteNote 成功", fastlog.Uint("id", id))
	return nil
}

// GetNote 按 ID 获取单条笔记
func (a *App) GetNote(id uint) (*models.Note, error) {
	a.LogSvc.Logger.Debugw("GetNote", fastlog.Uint("id", id))
	note, err := a.noteService.GetByID(id)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNote 失败", fastlog.Error(err))
		return nil, err
	}
	return note, nil
}

// GetNoteContent 按 ID 仅获取笔记的完整 content 文本（列表查询只返回截断版本，用于编辑器按需加载）
func (a *App) GetNoteContent(id uint) (string, error) {
	a.LogSvc.Logger.Debugw("GetNoteContent", fastlog.Uint("id", id))
	content, err := a.noteService.GetNoteContent(id)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNoteContent 失败", fastlog.Error(err))
		return "", err
	}
	return content, nil
}

// GetNoteRefContext 构建笔记引用上下文。
// 后端一次性完成：查库 → 截断 → 拼装，返回每条笔记信息和完整 context 文本。
func (a *App) GetNoteRefContext(ids []uint) (*services.NoteRefContext, error) {
	a.LogSvc.Logger.Debugw("GetNoteRefContext", fastlog.Int("count", len(ids)))
	ctx, err := a.noteService.BuildNoteRefContext(ids)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNoteRefContext 失败", fastlog.Error(err))
		return nil, err
	}
	return ctx, nil
}

// GetNotes 分页获取未删除的笔记列表，支持指定排序方式和笔记本筛选
func (a *App) GetNotes(page, pageSize int, sortBy string, notebookID uint) (*services.PaginatedResult, error) {
	a.LogSvc.Logger.Debugw("GetNotes", fastlog.Int("page", page), fastlog.Int("pageSize", pageSize), fastlog.Uint("notebookID", notebookID))
	var notes []models.Note
	var total int64
	var err error

	if notebookID > 0 {
		notes, total, err = a.noteService.GetAllByNotebook(page, pageSize, sortBy, notebookID)
	} else {
		notes, total, err = a.noteService.GetAll(page, pageSize, sortBy)
	}
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNotes 失败", fastlog.Error(err))
		return nil, err
	}
	return &services.PaginatedResult{
		Items:    notes,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetAllNoteIDs 获取所有未删除笔记的 ID 数组
func (a *App) GetAllNoteIDs() ([]uint, error) {
	a.LogSvc.Logger.Debugw("GetAllNoteIDs")
	ids, err := a.noteService.GetAllIDs()
	if err != nil {
		a.LogSvc.Logger.Errorw("GetAllNoteIDs 失败", fastlog.Error(err))
		return nil, err
	}
	return ids, nil
}

// SearchNotes 按关键词搜索笔记（标题/内容），支持分页、笔记本筛选、日期范围和标签 AND 过滤
func (a *App) SearchNotes(keyword string, page, pageSize int, notebookID uint, sortBy string, startDate, endDate string, tagIDs []uint) (*services.PaginatedResult, error) {
	a.LogSvc.Logger.Debugw("SearchNotes", fastlog.String("keyword", keyword), fastlog.Int("page", page), fastlog.Int("pageSize", pageSize), fastlog.Uint("notebookID", notebookID))
	var notes []models.Note
	var total int64
	var err error

	if notebookID > 0 {
		notes, total, err = a.noteService.SearchByNotebook(keyword, page, pageSize, notebookID, sortBy, startDate, endDate, tagIDs)
	} else {
		notes, total, err = a.noteService.Search(keyword, page, pageSize, sortBy, startDate, endDate, tagIDs)
	}
	if err != nil {
		a.LogSvc.Logger.Errorw("SearchNotes 失败", fastlog.Error(err))
		return nil, err
	}
	return &services.PaginatedResult{
		Items:    notes,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// SearchNoteIDs 按筛选条件返回所有匹配笔记 ID（不分页），用于全选功能
func (a *App) SearchNoteIDs(keyword string, notebookID uint, tagIDs []uint) ([]uint, error) {
	a.LogSvc.Logger.Debugw("SearchNoteIDs", fastlog.String("keyword", keyword), fastlog.Uint("notebookID", notebookID), fastlog.Int("tagCount", len(tagIDs)))
	var ids []uint
	var err error
	if notebookID > 0 {
		ids, err = a.noteService.SearchNoteIDsByNotebook(keyword, notebookID, tagIDs)
	} else {
		ids, err = a.noteService.SearchNoteIDs(keyword, tagIDs)
	}
	if err != nil {
		a.LogSvc.Logger.Errorw("SearchNoteIDs 失败", fastlog.Error(err))
		return nil, err
	}
	return ids, nil
}

// TogglePinNote 切换指定笔记的置顶状态
func (a *App) TogglePinNote(id uint) (*models.Note, error) {
	a.LogSvc.Logger.Debugw("TogglePinNote", fastlog.Uint("id", id))
	note, err := a.noteService.TogglePin(id)
	if err != nil {
		a.LogSvc.Logger.Errorw("TogglePinNote 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("TogglePinNote 成功", fastlog.Uint("id", note.ID))
	return note, nil
}

// GetTrashNotes 分页获取回收站中的笔记列表
func (a *App) GetTrashNotes(page, pageSize int) (*services.PaginatedResult, error) {
	a.LogSvc.Logger.Debugw("GetTrashNotes", fastlog.Int("page", page), fastlog.Int("pageSize", pageSize))
	notes, total, err := a.noteService.GetTrash(page, pageSize)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetTrashNotes 失败", fastlog.Error(err))
		return nil, err
	}
	return &services.PaginatedResult{
		Items:    notes,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// RestoreNote 从回收站恢复指定笔记
func (a *App) RestoreNote(id uint) error {
	a.LogSvc.Logger.Debugw("RestoreNote", fastlog.Uint("id", id))
	if err := a.noteService.Restore(id); err != nil {
		a.LogSvc.Logger.Errorw("RestoreNote 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("RestoreNote 成功", fastlog.Uint("id", id))
	return nil
}

// BatchPinNotes 批量置顶或取消置顶笔记
func (a *App) BatchPinNotes(noteIDs []uint, pin bool) error {
	a.LogSvc.Logger.Debugw("BatchPinNotes", fastlog.Int("count", len(noteIDs)), fastlog.Bool("pin", pin))
	if err := a.noteService.BatchPinNotes(noteIDs, pin); err != nil {
		a.LogSvc.Logger.Errorw("BatchPinNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("BatchPinNotes 成功", fastlog.Int("count", len(noteIDs)))
	return nil
}

// BatchDeleteNotes 批量软删除笔记
func (a *App) BatchDeleteNotes(ids []uint) error {
	a.LogSvc.Logger.Debugw("BatchDeleteNotes", fastlog.Int("count", len(ids)))
	if err := a.noteService.BatchDelete(ids); err != nil {
		a.LogSvc.Logger.Errorw("BatchDeleteNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("BatchDeleteNotes 成功", fastlog.Int("count", len(ids)))
	return nil
}

// BatchRestoreNotes 批量从回收站恢复笔记
func (a *App) BatchRestoreNotes(ids []uint) error {
	a.LogSvc.Logger.Debugw("BatchRestoreNotes", fastlog.Int("count", len(ids)))
	if err := a.noteService.BatchRestore(ids); err != nil {
		a.LogSvc.Logger.Errorw("BatchRestoreNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("BatchRestoreNotes 成功", fastlog.Int("count", len(ids)))
	return nil
}

// BatchAddTagToNotes 批量添加标签到笔记
func (a *App) BatchAddTagToNotes(noteIDs []uint, tagID uint) error {
	a.LogSvc.Logger.Debugw("BatchAddTagToNotes", fastlog.Int("count", len(noteIDs)), fastlog.Uint("tagID", tagID))
	if err := a.tagService.BatchAddTagToNotes(noteIDs, tagID); err != nil {
		a.LogSvc.Logger.Errorw("BatchAddTagToNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("BatchAddTagToNotes 成功", fastlog.Int("count", len(noteIDs)))
	return nil
}

// BatchRemoveTagFromNotes 批量从笔记移除标签
func (a *App) BatchRemoveTagFromNotes(noteIDs []uint, tagID uint) error {
	a.LogSvc.Logger.Debugw("BatchRemoveTagFromNotes", fastlog.Int("count", len(noteIDs)), fastlog.Uint("tagID", tagID))
	if err := a.tagService.BatchRemoveTagFromNotes(noteIDs, tagID); err != nil {
		a.LogSvc.Logger.Errorw("BatchRemoveTagFromNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("BatchRemoveTagFromNotes 成功", fastlog.Int("count", len(noteIDs)))
	return nil
}

// RestoreAllNotes 批量恢复回收站中所有笔记
func (a *App) RestoreAllNotes() error {
	a.LogSvc.Logger.Debugw("RestoreAllNotes")
	if err := a.noteService.RestoreAll(); err != nil {
		a.LogSvc.Logger.Errorw("RestoreAllNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("RestoreAllNotes 成功")
	return nil
}

// EmptyTrash 永久清空回收站中所有笔记
func (a *App) EmptyTrash() error {
	a.LogSvc.Logger.Debugw("EmptyTrash")
	if err := a.noteService.EmptyTrash(); err != nil {
		a.LogSvc.Logger.Errorw("EmptyTrash 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("EmptyTrash 成功")
	return nil
}

// GetNotesByTag 按标签分页获取笔记，支持指定排序方式（updated_at/created_at/title）
func (a *App) GetNotesByTag(tagID uint, page, pageSize int, sortBy string) (*services.PaginatedResult, error) {
	a.LogSvc.Logger.Debugw("GetNotesByTag", fastlog.Uint("tagID", tagID), fastlog.Int("page", page), fastlog.Int("pageSize", pageSize))
	notes, total, err := a.noteService.GetByTag(tagID, page, pageSize, sortBy)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNotesByTag 失败", fastlog.Error(err))
		return nil, err
	}
	return &services.PaginatedResult{
		Items:    notes,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetNotesByDate 获取指定创建日期的笔记列表，按创建时间降序排列
func (a *App) GetNotesByDate(date string) ([]models.Note, error) {
	a.LogSvc.Logger.Debugw("GetNotesByDate", fastlog.String("date", date))
	notes, err := a.noteService.GetByDate(date)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNotesByDate 失败", fastlog.Error(err))
		return nil, err
	}
	return notes, nil
}

// GetMonthNoteCounts 获取指定月份中每天创建的笔记数量
func (a *App) GetMonthNoteCounts(year int, month int) (map[int]int, error) {
	a.LogSvc.Logger.Debugw("GetMonthNoteCounts", fastlog.Int("year", year), fastlog.Int("month", month))
	counts, err := a.noteService.GetMonthCounts(year, month)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetMonthNoteCounts 失败", fastlog.Error(err))
		return nil, err
	}
	return counts, nil
}

// GetDataStats 获取数据统计概览（委托 StatsService 聚合，口径与 get_stats 工具一致）
func (a *App) GetDataStats() (*services.DataStats, error) {
	a.LogSvc.Logger.Debugw("GetDataStats")
	stats, err := a.statsService.GetDataStats()
	if err != nil {
		a.LogSvc.Logger.Errorw("GetDataStats 失败", fastlog.Error(err))
		return nil, err
	}
	return stats, nil
}

// VacuumDatabase 执行存储优化操作：清理无效数据后执行 VACUUM，返回释放的空间大小
func (a *App) VacuumDatabase() (string, error) {
	a.LogSvc.Logger.Debugw("VacuumDatabase")
	// 读取回收站自动清理天数设置
	daysStr := a.settingService.Get("trash_cleanup_retention_days")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30
	}

	// 1. 清理空 AI 会话
	deletedSessions := a.aiService.DeleteEmptyAISessions()

	// 2. 清理孤儿 AI 消息
	deletedOrphanMsgs := a.aiService.DeleteOrphanMessages()

	// 3. 清理过期回收站笔记（超过 N 天）
	deletedNotes := a.noteService.CleanExpiredTrash(days)

	// 4. 清理过期回收站笔记本（超过 N 天）
	deletedNotebooks := a.notebookService.CleanExpiredTrash(days)

	// 5. 迁移指向不存在笔记本的笔记到默认笔记本
	migratedNotes := a.noteService.MigrateOrphanNotes()

	// 6. 清空已完成待办
	deletedTodos, _ := a.todoService.DeleteCompleted()

	// 7. 清理未引用的图片文件
	deletedImages := a.CleanupOrphanImages()

	// 获取瘦身前数据库文件大小
	dbPath, _ := database.DefaultDBPath()
	var beforeSize int64
	if fi, err := os.Stat(dbPath); err == nil {
		beforeSize = fi.Size()
	}

	if err := a.noteService.Vacuum(); err != nil {
		a.LogSvc.Logger.Errorw("VacuumDatabase 失败", fastlog.Error(err))
		return "", fmt.Errorf("存储优化失败: %w", err)
	}

	// 获取瘦身后数据库文件大小
	var afterSize int64
	if fi, err := os.Stat(dbPath); err == nil {
		afterSize = fi.Size()
	}

	saved := beforeSize - afterSize
	if saved < 0 {
		saved = 0
	}
	var savedStr string
	switch {
	case saved < 1024:
		savedStr = fmt.Sprintf("%d B", saved)
	case saved < 1024*1024:
		savedStr = fmt.Sprintf("%.1f KB", float64(saved)/1024)
	default:
		savedStr = fmt.Sprintf("%.1f MB", float64(saved)/(1024*1024))
	}

	// 组装结果消息
	var parts []string
	parts = append(parts, fmt.Sprintf("释放了 %s 空间", savedStr))
	if deletedSessions > 0 {
		parts = append(parts, fmt.Sprintf("清理了 %d 个空 AI 会话", deletedSessions))
	}
	if deletedOrphanMsgs > 0 {
		parts = append(parts, fmt.Sprintf("清理了 %d 条孤儿 AI 消息", deletedOrphanMsgs))
	}
	if deletedNotes > 0 {
		parts = append(parts, fmt.Sprintf("清理了 %d 条过期回收站笔记", deletedNotes))
	}
	if deletedNotebooks > 0 {
		parts = append(parts, fmt.Sprintf("清理了 %d 个过期回收站笔记本", deletedNotebooks))
	}
	if migratedNotes > 0 {
		parts = append(parts, fmt.Sprintf("迁移了 %d 条孤儿笔记到默认笔记本", migratedNotes))
	}
	if deletedTodos > 0 {
		parts = append(parts, fmt.Sprintf("清空了 %d 个已完成待办", deletedTodos))
	}
	if deletedImages > 0 {
		parts = append(parts, fmt.Sprintf("删除了 %d 张未引用图片", deletedImages))
	}
	result := strings.Join(parts, "，")
	a.LogSvc.Logger.Infow("存储优化完成",
		fastlog.String("result", result),
	)
	return result, nil
}

// ExportDataWithDialog 弹出保存对话框，导出 ZIP 格式备份（含数据库和图片）
func (a *App) ExportDataWithDialog() (string, error) {
	a.LogSvc.Logger.Debugw("ExportDataWithDialog")
	// 弹出保存对话框
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出数据备份",
		DefaultFilename: "jot-backup-" + time.Now().Format("2006-01-02") + ".zip",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP 文件 (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil {
		a.LogSvc.Logger.Errorw("ExportDataWithDialog 失败", fastlog.Error(err))
		return "", err
	}
	if filePath == "" {
		return "已取消", nil
	}

	// 调用统一导出
	if err := a.exportSnapshot(filePath); err != nil {
		a.LogSvc.Logger.Errorw("ExportDataWithDialog 失败", fastlog.Error(err))
		return "", fmt.Errorf("导出失败: %w", err)
	}

	a.LogSvc.Logger.Infow("数据导出成功")
	return "导出成功：" + filePath, nil
}

// ImportDatabaseWithDialog 弹出文件选择对话框，从 ZIP 备份文件恢复数据（含图片）
func (a *App) ImportDatabaseWithDialog() (*services.ImportResult, error) {
	a.LogSvc.Logger.Debugw("ImportDatabaseWithDialog")
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "导入数据备份",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP 文件 (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil {
		a.LogSvc.Logger.Errorw("ImportDatabaseWithDialog 失败", fastlog.Error(err))
		return &services.ImportResult{Message: "导入失败：" + err.Error()}, nil
	}
	if filePath == "" {
		return &services.ImportResult{Message: "已取消"}, nil
	}

	if err := a.importFromArchive(filePath); err != nil {
		a.LogSvc.Logger.Errorw("ImportDatabaseWithDialog 失败", fastlog.Error(err))
		return &services.ImportResult{Message: "导入失败：" + err.Error()}, nil
	}

	a.LogSvc.Logger.Infow("数据导入成功")
	return &services.ImportResult{
		Message:      "已从备份文件恢复数据库与图片",
		SuccessCount: 1,
	}, nil
}

// ==================== Tag 相关绑定方法 ====================

// CreateTag 创建一个新标签
func (a *App) CreateTag(name, color string) (*models.Tag, error) {
	a.LogSvc.Logger.Debugw("CreateTag", fastlog.String("name", name))
	tag, err := a.tagService.Create(name, color)
	if err != nil {
		a.LogSvc.Logger.Errorw("CreateTag 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("CreateTag 成功", fastlog.Uint("id", tag.ID))
	return tag, nil
}

// UpdateTag 更新指定标签的名称和颜色
func (a *App) UpdateTag(id uint, name, color string) (*models.Tag, error) {
	a.LogSvc.Logger.Debugw("UpdateTag", fastlog.Uint("id", id))
	tag, err := a.tagService.Update(id, name, color)
	if err != nil {
		a.LogSvc.Logger.Errorw("UpdateTag 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("UpdateTag 成功", fastlog.Uint("id", tag.ID))
	return tag, nil
}

// DeleteTag 删除指定标签
func (a *App) DeleteTag(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteTag", fastlog.Uint("id", id))
	if err := a.tagService.Delete(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteTag 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteTag 成功", fastlog.Uint("id", id))
	return nil
}

// GetAllTags 获取所有标签列表
func (a *App) GetAllTags() ([]models.Tag, error) {
	a.LogSvc.Logger.Debugw("GetAllTags")
	tags, err := a.tagService.GetAll()
	if err != nil {
		a.LogSvc.Logger.Errorw("GetAllTags 失败", fastlog.Error(err))
		return nil, err
	}
	return tags, nil
}

// AddTagToNote 为指定笔记添加标签
func (a *App) AddTagToNote(noteID, tagID uint) error {
	a.LogSvc.Logger.Debugw("AddTagToNote", fastlog.Uint("noteID", noteID), fastlog.Uint("tagID", tagID))
	if err := a.tagService.AddTagToNote(noteID, tagID); err != nil {
		a.LogSvc.Logger.Errorw("AddTagToNote 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("AddTagToNote 成功", fastlog.Uint("noteID", noteID), fastlog.Uint("tagID", tagID))
	return nil
}

// RemoveTagFromNote 为指定笔记移除标签
func (a *App) RemoveTagFromNote(noteID, tagID uint) error {
	a.LogSvc.Logger.Debugw("RemoveTagFromNote", fastlog.Uint("noteID", noteID), fastlog.Uint("tagID", tagID))
	if err := a.tagService.RemoveTagFromNote(noteID, tagID); err != nil {
		a.LogSvc.Logger.Errorw("RemoveTagFromNote 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("RemoveTagFromNote 成功", fastlog.Uint("noteID", noteID), fastlog.Uint("tagID", tagID))
	return nil
}

// ==================== Notebook 相关绑定方法 ====================

// CreateNotebook 创建新笔记本
func (a *App) CreateNotebook(name string) (*models.Notebook, error) {
	a.LogSvc.Logger.Debugw("CreateNotebook", fastlog.String("name", name))
	notebook, err := a.notebookService.Create(name)
	if err != nil {
		a.LogSvc.Logger.Errorw("CreateNotebook 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("CreateNotebook 成功", fastlog.Uint("id", notebook.ID))
	return notebook, nil
}

// RenameNotebook 重命名笔记本
func (a *App) RenameNotebook(id uint, name string) (*models.Notebook, error) {
	a.LogSvc.Logger.Debugw("RenameNotebook", fastlog.Uint("id", id))
	notebook, err := a.notebookService.Update(id, name)
	if err != nil {
		a.LogSvc.Logger.Errorw("RenameNotebook 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("RenameNotebook 成功", fastlog.Uint("id", notebook.ID))
	return notebook, nil
}

// DeleteNotebook 删除笔记本，其下笔记自动迁入默认笔记本
func (a *App) DeleteNotebook(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteNotebook", fastlog.Uint("id", id))
	if err := a.notebookService.Delete(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteNotebook 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteNotebook 成功", fastlog.Uint("id", id))
	return nil
}

// DeleteNotebookWithNotes 删除笔记本并清空其下所有笔记
func (a *App) DeleteNotebookWithNotes(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteNotebookWithNotes", fastlog.Uint("id", id))
	if err := a.notebookService.DeleteWithNotes(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteNotebookWithNotes 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteNotebookWithNotes 成功", fastlog.Uint("id", id))
	return nil
}

// GetTrashNotebooks 分页获取回收站中已删除的笔记本列表
func (a *App) GetTrashNotebooks(page, pageSize int) (*services.PaginatedResult, error) {
	a.LogSvc.Logger.Debugw("GetTrashNotebooks", fastlog.Int("page", page), fastlog.Int("pageSize", pageSize))
	notebooks, total, err := a.notebookService.GetTrash(page, pageSize)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetTrashNotebooks 失败", fastlog.Error(err))
		return nil, err
	}
	return &services.PaginatedResult{
		Items:    notebooks,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// RestoreTrashNotebook 从回收站恢复指定笔记本
func (a *App) RestoreTrashNotebook(id uint) error {
	a.LogSvc.Logger.Debugw("RestoreTrashNotebook", fastlog.Uint("id", id))
	if err := a.notebookService.RestoreFromTrash(id); err != nil {
		a.LogSvc.Logger.Errorw("RestoreTrashNotebook 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("RestoreTrashNotebook 成功", fastlog.Uint("id", id))
	return nil
}

// PermanentDeleteTrashNotebook 从回收站永久删除指定笔记本
func (a *App) PermanentDeleteTrashNotebook(id uint) error {
	a.LogSvc.Logger.Debugw("PermanentDeleteTrashNotebook", fastlog.Uint("id", id))
	if err := a.notebookService.PermanentDeleteFromTrash(id); err != nil {
		a.LogSvc.Logger.Errorw("PermanentDeleteTrashNotebook 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("PermanentDeleteTrashNotebook 成功", fastlog.Uint("id", id))
	return nil
}

// RestoreAllTrashNotebooks 恢复回收站中所有笔记本
func (a *App) RestoreAllTrashNotebooks() error {
	a.LogSvc.Logger.Debugw("RestoreAllTrashNotebooks")
	if err := a.notebookService.RestoreAllFromTrash(); err != nil {
		a.LogSvc.Logger.Errorw("RestoreAllTrashNotebooks 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("RestoreAllTrashNotebooks 成功")
	return nil
}

// EmptyTrashNotebooks 清空回收站中所有笔记本
func (a *App) EmptyTrashNotebooks() error {
	a.LogSvc.Logger.Debugw("EmptyTrashNotebooks")
	if err := a.notebookService.EmptyTrash(); err != nil {
		a.LogSvc.Logger.Errorw("EmptyTrashNotebooks 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("EmptyTrashNotebooks 成功")
	return nil
}

// MoveNoteToNotebook 将单条笔记移动到目标笔记本
func (a *App) MoveNoteToNotebook(id, targetNotebookID uint) error {
	a.LogSvc.Logger.Debugw("MoveNoteToNotebook", fastlog.Uint("id", id), fastlog.Uint("targetNotebookID", targetNotebookID))
	if err := a.noteService.MoveToNotebook(id, targetNotebookID); err != nil {
		a.LogSvc.Logger.Errorw("MoveNoteToNotebook 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("MoveNoteToNotebook 成功", fastlog.Uint("id", id))
	return nil
}

// BatchMoveNotesToNotebook 批量将多条笔记移动到目标笔记本
func (a *App) BatchMoveNotesToNotebook(noteIDs []uint, targetNotebookID uint) error {
	a.LogSvc.Logger.Debugw("BatchMoveNotesToNotebook", fastlog.Int("count", len(noteIDs)), fastlog.Uint("targetNotebookID", targetNotebookID))
	if err := a.noteService.BatchMoveToNotebook(noteIDs, targetNotebookID); err != nil {
		a.LogSvc.Logger.Errorw("BatchMoveNotesToNotebook 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("BatchMoveNotesToNotebook 成功", fastlog.Int("count", len(noteIDs)))
	return nil
}

// GetAllNotebooks 获取所有未删除笔记本列表
func (a *App) GetAllNotebooks() ([]models.Notebook, error) {
	a.LogSvc.Logger.Debugw("GetAllNotebooks")
	notebooks, err := a.notebookService.GetAll()
	if err != nil {
		a.LogSvc.Logger.Errorw("GetAllNotebooks 失败", fastlog.Error(err))
		return nil, err
	}
	return notebooks, nil
}

// GetNotebookNoteCounts 获取各笔记本下笔记数量
func (a *App) GetNotebookNoteCounts() (map[uint]int, error) {
	a.LogSvc.Logger.Debugw("GetNotebookNoteCounts")
	counts, err := a.notebookService.GetAllNotesCount(a.db)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNotebookNoteCounts 失败", fastlog.Error(err))
		return nil, err
	}
	return counts, nil
}

// GetNoteIDsByNotebook 获取指定笔记本中所有未删除笔记的 ID 数组
func (a *App) GetNoteIDsByNotebook(notebookID uint) ([]uint, error) {
	a.LogSvc.Logger.Debugw("GetNoteIDsByNotebook", fastlog.Uint("notebookID", notebookID))
	ids, err := a.noteService.GetAllNoteIDsByNotebook(notebookID)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetNoteIDsByNotebook 失败", fastlog.Error(err))
		return nil, err
	}
	return ids, nil
}

// ==================== Setting 相关绑定方法 ====================

// GetSetting 获取指定 key 的配置值
func (a *App) GetSetting(key string) string {
	return a.settingService.Get(key)
}

// SetSetting 设置指定 key 的配置值
func (a *App) SetSetting(key, value string) error {
	return a.settingService.Set(key, value)
}

// GetAllSettings 获取全部设置项
func (a *App) GetAllSettings() services.SettingsConfig {
	return a.settingService.GetAllSettings()
}

// SaveAllSettings 保存全部设置项
func (a *App) SaveAllSettings(cfg services.SettingsConfig) error {
	a.LogSvc.Logger.Debugw("SaveAllSettings")
	if err := a.settingService.SaveAllSettings(cfg); err != nil {
		a.LogSvc.Logger.Errorw("SaveAllSettings 失败", fastlog.Error(err))
		return err
	}
	// 动态调整日志级别
	if a.LogSvc != nil {
		oldLevel := a.LogSvc.Logger.Level()
		newLevel := services.LevelFromInt(cfg.LogLevel)
		a.LogSvc.SetLevel(newLevel)
		if oldLevel != newLevel {
			a.LogSvc.Logger.Infow("日志级别已变更",
				fastlog.String("from", oldLevel.String()),
				fastlog.String("to", newLevel.String()),
			)
		}
	}
	return nil
}

// VerifyScreenLockPassword 验证锁屏密码
// 返回 true 表示验证通过，false 表示验证失败
// 如果数据库中密码为空（功能关闭），始终返回 true
func (a *App) VerifyScreenLockPassword(password string) bool {
	stored := a.settingService.Get("screen_lock_password")
	if stored == "" {
		return true
	}
	hash := sha256.Sum256([]byte(password + "jot-screen-lock-salt"))
	return hex.EncodeToString(hash[:]) == stored
}

// SetScreenLockPassword 设置/修改锁屏密码
// oldPwd: 旧密码（首次设置时传 ""），newPwd: 新密码
// 仅当数据库中已有密码且 oldPwd 不匹配时返回错误
func (a *App) SetScreenLockPassword(oldPwd, newPwd string) error {
	if newPwd == "" {
		return errors.New("新密码不能为空")
	}
	stored := a.settingService.Get("screen_lock_password")
	if stored != "" {
		oldHash := sha256.Sum256([]byte(oldPwd + "jot-screen-lock-salt"))
		if hex.EncodeToString(oldHash[:]) != stored {
			return errors.New("旧密码不正确")
		}
		// 新密码不能与旧密码相同
		newHash := sha256.Sum256([]byte(newPwd + "jot-screen-lock-salt"))
		if hex.EncodeToString(newHash[:]) == stored {
			return errors.New("新密码不能与旧密码相同")
		}
	}
	newHash := sha256.Sum256([]byte(newPwd + "jot-screen-lock-salt"))
	return a.settingService.Set("screen_lock_password", hex.EncodeToString(newHash[:]))
}

// GetMaxFileSize 获取最大文件限制大小（字节），空值时返回默认 1MB
func (a *App) GetMaxFileSize() int64 {
	a.LogSvc.Logger.Debugw("GetMaxFileSize")
	val := a.settingService.Get("max_file_size")
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return 1 * 1024 * 1024
	}
	if n > 100 {
		n = 100
	}
	return int64(n) * 1024 * 1024
}

// ==================== AI 相关绑定方法 ====================

// GetAIConfig 获取 AI 服务配置
func (a *App) GetAIConfig() services.AIConfig {
	a.LogSvc.Logger.Debugw("GetAIConfig")
	return a.aiService.GetConfig()
}

// SaveAIConfig 保存 AI 服务配置
func (a *App) SaveAIConfig(cfg services.AIConfig) error {
	a.LogSvc.Logger.Debugw("SaveAIConfig")
	if err := a.aiService.SaveConfig(cfg); err != nil {
		a.LogSvc.Logger.Errorw("SaveAIConfig 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SaveAIConfig 成功")
	return nil
}

// ==================== API 配置预设绑定 ====================

// GetProfiles 获取所有 API 配置预设
func (a *App) GetProfiles() []models.APIProfile {
	a.LogSvc.Logger.Debugw("GetProfiles")
	return a.profileService.ListProfiles()
}

// CreateProfile 创建 API 配置预设
func (a *App) CreateProfile(name, baseURL, apiKey string) models.APIProfile {
	a.LogSvc.Logger.Debugw("CreateProfile", fastlog.String("name", name), fastlog.String("key", "***"))
	return a.profileService.CreateProfile(name, baseURL, apiKey)
}

// UpdateProfile 更新 API 配置预设
func (a *App) UpdateProfile(id uint, name, baseURL, apiKey string) error {
	a.LogSvc.Logger.Debugw("UpdateProfile", fastlog.Uint("id", id), fastlog.String("name", name), fastlog.String("key", "***"))
	if err := a.profileService.UpdateProfile(id, name, baseURL, apiKey); err != nil {
		a.LogSvc.Logger.Errorw("UpdateProfile 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("UpdateProfile 成功", fastlog.Uint("id", id))
	return nil
}

// DeleteProfile 删除 API 配置预设
func (a *App) DeleteProfile(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteProfile", fastlog.Uint("id", id))
	if err := a.profileService.DeleteProfile(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteProfile 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteProfile 成功", fastlog.Uint("id", id))
	return nil
}

// SwitchProfile 切换 API 配置预设
// target 指定预设写入的配置组："chat" 写入对话连接键 ai_*，"embed" 写入量化连接键 ai_embed_*
func (a *App) SwitchProfile(target string, id uint) error {
	a.LogSvc.Logger.Debugw("SwitchProfile", fastlog.String("target", target), fastlog.Uint("id", id))
	if err := a.profileService.SwitchProfile(target, id); err != nil {
		a.LogSvc.Logger.Errorw("SwitchProfile 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SwitchProfile 成功", fastlog.String("target", target), fastlog.Uint("id", id))
	return nil
}

// ==================== 向量量化索引绑定 ====================

// IndexNotesByAll 对全部未删除笔记发起异步量化索引，立即返回；进度与结果通过事件推送
func (a *App) IndexNotesByAll() error {
	a.LogSvc.Logger.Debugw("IndexNotesByAll")
	ids, err := a.noteService.GetAllIDs()
	if err != nil {
		a.LogSvc.Logger.Errorw("IndexNotesByAll 获取笔记列表失败", fastlog.Error(err))
		return err
	}
	return a.startVectorIndex(context.Background(), ids)
}

// IndexNotesByNotebooks 对指定笔记本下的全部未删除笔记发起异步量化索引
func (a *App) IndexNotesByNotebooks(notebookIDs []uint) error {
	a.LogSvc.Logger.Debugw("IndexNotesByNotebooks", fastlog.Int("notebook_count", len(notebookIDs)))
	var ids []uint
	if err := a.db.Model(&models.Note{}).
		Where("notebook_id IN ? AND deleted_at IS NULL", notebookIDs).
		Pluck("id", &ids).Error; err != nil {
		a.LogSvc.Logger.Errorw("IndexNotesByNotebooks 获取笔记列表失败", fastlog.Error(err))
		return err
	}
	return a.startVectorIndex(context.Background(), ids)
}

// IndexNotesByIDs 对指定笔记 ID 列表发起异步量化索引
func (a *App) IndexNotesByIDs(ids []uint) error {
	a.LogSvc.Logger.Debugw("IndexNotesByIDs", fastlog.Int("note_count", len(ids)))
	return a.startVectorIndex(context.Background(), ids)
}

// IndexNotesUnindexed 仅量化未量化过的笔记（非软删且无向量记录），异步执行，进度与结果通过事件推送
func (a *App) IndexNotesUnindexed() error {
	a.LogSvc.Logger.Debugw("IndexNotesUnindexed")
	ids, err := a.vectorService.GetUnindexedNoteIDs()
	if err != nil {
		a.LogSvc.Logger.Errorw("IndexNotesUnindexed 获取未量化笔记失败", fastlog.Error(err))
		return err
	}
	if len(ids) == 0 {
		return errors.New("所有笔记都已量化")
	}
	a.LogSvc.Logger.Debugw("IndexNotesUnindexed 待量化", fastlog.Int("note_count", len(ids)))
	return a.startVectorIndex(context.Background(), ids)
}

// IndexNotesStale 仅重新量化内容已变化的笔记（已量化但当前内容与量化时不一致），异步执行
func (a *App) IndexNotesStale() error {
	a.LogSvc.Logger.Debugw("IndexNotesStale")
	ids, err := a.vectorService.GetStaleNoteIDs()
	if err != nil {
		a.LogSvc.Logger.Errorw("IndexNotesStale 获取需重新量化笔记失败", fastlog.Error(err))
		return err
	}
	if len(ids) == 0 {
		return errors.New("没有需要重新量化的笔记")
	}
	a.LogSvc.Logger.Debugw("IndexNotesStale 待量化", fastlog.Int("note_count", len(ids)))
	return a.startVectorIndex(context.Background(), ids)
}

// GetVectorIndexStatus 返回向量索引全局统计（轻量：COUNT/SUM 聚合，不含逐笔记内容比对），
// 供数据管理页概览（信笺统计）使用；量化弹窗的完整状态走 GetVectorIndexOverview
// VectorIndexStatus 向量索引全局统计结果（Wails 绑定需单值返回，多返回值只保留第一个）
type VectorIndexStatus struct {
	NoteCount  int   `json:"noteCount"`
	ChunkCount int   `json:"chunkCount"`
	SizeBytes  int64 `json:"sizeBytes"`
}

func (a *App) GetVectorIndexStatus() (*VectorIndexStatus, error) {
	noteCount, chunkCount, sizeBytes, err := a.vectorService.GetIndexStatus()
	if err != nil {
		a.LogSvc.Logger.Errorw("GetVectorIndexStatus 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Debugw("GetVectorIndexStatus 结果", fastlog.Int("noteCount", noteCount), fastlog.Int("chunkCount", chunkCount), fastlog.Int64("sizeBytes", sizeBytes))
	return &VectorIndexStatus{NoteCount: noteCount, ChunkCount: chunkCount, SizeBytes: sizeBytes}, nil
}

// GetVectorIndexOverview 返回量化弹窗所需的完整状态：
//   - noteCount/chunkCount/sizeBytes：全局存储口径（GetIndexStatus，含回收站残留向量）
//   - totalNotes/unindexedNotes/staleNotes/upToDateNotes：非软删笔记的量化状态口径
//     （staleNotes = 已量化但内容已变化、需重新量化的笔记数）
//
// 注意：该接口需逐笔记重新切块比对（classifyVectorNotes），成本显著高于 GetVectorIndexStatus，
// 仅供量化弹窗调用，勿用于高频路径（如数据管理页概览）
// VectorIndexOverview 向量索引完整状态结果（量化弹窗专用）
type VectorIndexOverview struct {
	NoteCount      int   `json:"noteCount"`
	ChunkCount     int   `json:"chunkCount"`
	SizeBytes      int64 `json:"sizeBytes"`
	TotalNotes     int   `json:"totalNotes"`
	UnindexedNotes int   `json:"unindexedNotes"`
	StaleNotes     int   `json:"staleNotes"`
	UpToDateNotes  int   `json:"upToDateNotes"`
}

func (a *App) GetVectorIndexOverview() (*VectorIndexOverview, error) {
	noteCount, chunkCount, sizeBytes, err := a.vectorService.GetIndexStatus()
	if err != nil {
		a.LogSvc.Logger.Errorw("GetVectorIndexOverview 失败", fastlog.Error(err))
		return nil, err
	}
	totalNotes, unindexedNotes, staleNotes, upToDateNotes, err := a.vectorService.GetVectorNoteOverview()
	if err != nil {
		a.LogSvc.Logger.Errorw("GetVectorIndexOverview 量化状态统计失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Debugw("GetVectorIndexOverview 结果",
		fastlog.Int("noteCount", noteCount),
		fastlog.Int("chunkCount", chunkCount),
		fastlog.Int64("sizeBytes", sizeBytes),
		fastlog.Int("totalNotes", totalNotes),
		fastlog.Int("unindexedNotes", unindexedNotes),
		fastlog.Int("staleNotes", staleNotes),
		fastlog.Int("upToDateNotes", upToDateNotes))
	return &VectorIndexOverview{
		NoteCount:      noteCount,
		ChunkCount:     chunkCount,
		SizeBytes:      sizeBytes,
		TotalNotes:     totalNotes,
		UnindexedNotes: unindexedNotes,
		StaleNotes:     staleNotes,
		UpToDateNotes:  upToDateNotes,
	}, nil
}

// DeleteAllVectors 删除全部向量索引内容
func (a *App) DeleteAllVectors() error {
	a.LogSvc.Logger.Debugw("DeleteAllVectors")
	if err := a.vectorService.DeleteAllVectors(); err != nil {
		a.LogSvc.Logger.Errorw("DeleteAllVectors 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteAllVectors 成功")
	return nil
}

// CancelVectorIndex 停止当前正在进行的量化任务（异步取消，任务在批次间/笔记间退出）
// vectorIndexRunning 复位由任务 goroutine 的 release 负责，这里不直接复位，避免并发竞态
func (a *App) CancelVectorIndex() error {
	a.LogSvc.Logger.Debugw("CancelVectorIndex")
	a.vectorIndexMu.Lock()
	defer a.vectorIndexMu.Unlock()
	if !a.vectorIndexRunning || a.vectorIndexCancel == nil {
		return errors.New("当前没有正在进行的量化任务")
	}
	a.vectorIndexCancel()
	a.LogSvc.Logger.Infow("CancelVectorIndex 已触发取消")
	return nil
}

// GetEmbedConfig 读取量化连接配置（ai_embed_* 三键），apiKey 按现有 B64 编码方式解码
func (a *App) GetEmbedConfig() (baseURL, apiKey, model string, err error) {
	a.LogSvc.Logger.Debugw("GetEmbedConfig")
	baseURL = a.settingService.Get("ai_embed_base_url")
	apiKey = services.DecodeB64(a.settingService.Get("ai_embed_api_key"))
	model = a.settingService.Get("ai_embed_model")
	return baseURL, apiKey, model, nil
}

// CardRecallCheckResult 卡片召回开启校验结果
type CardRecallCheckResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// ValidateCardRecall 校验卡片召回是否可以开启：
//  1. 基础判断：量化连接（base_url）或量化模型未设置 → 拒绝
//  2. API Key 必填
//  3. 量化表内容判断：表为空拒绝；当前量化模型无对应记录拒绝
func (a *App) ValidateCardRecall() CardRecallCheckResult {
	baseURL, apiKey, model, _ := a.GetEmbedConfig()
	// 1. 基础判断：量化连接或量化模型未设置
	if baseURL == "" || model == "" {
		return CardRecallCheckResult{OK: false, Message: "请先在设置中配置量化连接与量化模型"}
	}
	// 2. API Key 必填
	if apiKey == "" {
		return CardRecallCheckResult{OK: false, Message: "请先填写量化 API Key"}
	}
	// 3. 量化表内容判断
	total, err := a.vectorService.CountAllVectors()
	if err != nil {
		a.LogSvc.Logger.Errorw("ValidateCardRecall 统计量化表失败", fastlog.Error(err))
		return CardRecallCheckResult{OK: false, Message: "量化表检查失败，请查看日志"}
	}
	if total == 0 {
		return CardRecallCheckResult{OK: false, Message: "量化表为空，请先在数据管理中量化笔记"}
	}
	modelCnt, err := a.vectorService.CountVectorsByModel(model)
	if err != nil {
		a.LogSvc.Logger.Errorw("ValidateCardRecall 统计模型向量失败", fastlog.Error(err))
		return CardRecallCheckResult{OK: false, Message: "量化表检查失败，请查看日志"}
	}
	if modelCnt == 0 {
		return CardRecallCheckResult{OK: false, Message: fmt.Sprintf("当前量化模型「%s」暂无量化数据，请先使用该模型量化笔记", model)}
	}
	return CardRecallCheckResult{OK: true, Message: ""}
}

// ValidateVectorIndexConfig 校验量化连接配置是否可以发起量化：
//  1. 基础判断：量化连接（base_url）或量化模型未设置 → 拒绝
//  2. API Key 必填
//
// （仅校验配置本身，不检查量化表内容，与卡片召回 ValidateCardRecall 的检查范围不同）
func (a *App) ValidateVectorIndexConfig() CardRecallCheckResult {
	baseURL, apiKey, model, _ := a.GetEmbedConfig()
	if baseURL == "" || model == "" {
		return CardRecallCheckResult{OK: false, Message: "请先在设置中配置量化连接与量化模型"}
	}
	if apiKey == "" {
		return CardRecallCheckResult{OK: false, Message: "请先填写量化 API Key"}
	}
	return CardRecallCheckResult{OK: true, Message: ""}
}

// startVectorIndex 量化任务公共入口：三个量化绑定方法取到笔记 ID 后统一调用
// 校验量化模型配置与防重入，随后异步执行 IndexNotes，通过 EventsEmit 推送进度与结果
func (a *App) startVectorIndex(ctx context.Context, noteIDs []uint) error {
	// 防重入：量化任务进行中时拒绝新任务
	a.vectorIndexMu.Lock()
	if a.vectorIndexRunning {
		a.vectorIndexMu.Unlock()
		return errors.New("量化任务正在进行中")
	}
	a.vectorIndexRunning = true
	a.vectorIndexMu.Unlock()

	// 创建可取消 ctx：用户可通过 CancelVectorIndex 停止量化任务
	ctx, cancel := context.WithCancel(ctx)
	a.vectorIndexMu.Lock()
	a.vectorIndexCancel = cancel
	a.vectorIndexMu.Unlock()

	// 任务结束或启动失败时复位运行标记并释放取消源
	release := func() {
		a.vectorIndexMu.Lock()
		a.vectorIndexRunning = false
		if a.vectorIndexCancel != nil {
			a.vectorIndexCancel()
			a.vectorIndexCancel = nil
		}
		a.vectorIndexMu.Unlock()
	}

	// 未完整配置量化连接时直接返回可读错误，不发起索引（与 ValidateCardRecall 校验强度一致）
	baseURL, apiKey, model, _ := a.GetEmbedConfig()
	if baseURL == "" || model == "" {
		release()
		return errors.New("请先在设置中配置量化连接与量化模型")
	}
	if apiKey == "" {
		release()
		return errors.New("请先填写量化 API Key")
	}
	if len(noteIDs) == 0 {
		release()
		return errors.New("没有找到有效的笔记")
	}

	// 构造 embedding 客户端（GetEmbedConfig 已解码 apiKey）
	client := einocli.NewClient(einocli.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
	})

	// 异步执行量化，避免阻塞 Wails 事件循环（参考 Agent 流的 goroutine 模式）
	go func() {
		defer release()
		success, failed, err := a.vectorService.IndexNotes(ctx, client, noteIDs, func(done, total int, title, stage string, chunkDone, chunkTotal int, errMsg string) {
			runtime.EventsEmit(a.ctx, "vector:index-progress", map[string]interface{}{
				"done":        done,
				"total":       total,
				"title":       title,
				"stage":       stage,
				"chunk_done":  chunkDone,
				"chunk_total": chunkTotal,
				"error":       errMsg,
			})
		})
		if err != nil {
			// 用户主动取消不报错（前端已确认停止并关闭弹窗），仅记录日志
			if errors.Is(err, context.Canceled) {
				a.LogSvc.Logger.Infow("向量量化索引已取消")
				return
			}
			a.LogSvc.Logger.Errorw("向量量化索引失败", fastlog.Error(err))
			runtime.EventsEmit(a.ctx, "vector:index-error", map[string]interface{}{"error": err.Error()})
			return
		}
		a.LogSvc.Logger.Infow("向量量化索引完成", fastlog.Int("success", success), fastlog.Int("failed", failed))
		runtime.EventsEmit(a.ctx, "vector:index-done", map[string]interface{}{
			"success": success,
			"failed":  failed,
		})
	}()
	return nil
}

// testAIConnection 连通性测试公共实现（Wails 绑定方法共用）
func (a *App) testAIConnection(baseURL, apiKey, logName string) (bool, error) {
	a.LogSvc.Logger.Debugw(logName, fastlog.String("baseURL", baseURL), fastlog.String("key", "***"))
	cfg := services.AIConfig{BaseURL: baseURL, APIKey: apiKey}
	result, err := a.aiService.TestConnection(cfg)
	if err != nil {
		a.LogSvc.Logger.Errorw(logName+" 失败", fastlog.Error(err))
		return false, err
	}
	a.LogSvc.Logger.Infow(logName + " 成功")
	return result, nil
}

// TestAIBaseURL 按指定 BaseURL/APIKey 测试连通性（对话/量化连接共用）
func (a *App) TestAIBaseURL(baseURL, apiKey string) (bool, error) {
	return a.testAIConnection(baseURL, apiKey, "TestAIBaseURL")
}

// TestAIConnection 测试指定 AI 配置的连通性（预设使用）
func (a *App) TestAIConnection(baseURL, apiKey string) (bool, error) {
	return a.testAIConnection(baseURL, apiKey, "TestAIConnection")
}

// TestVectorIndexConnection 测试量化服务连通性（量化弹窗打开时异步调用，不阻塞弹窗）
// 通过轻量 GET 请求检测服务可用性（均 5s 超时）
func (a *App) TestVectorIndexConnection() CardRecallCheckResult {
	baseURL, apiKey, model, _ := a.GetEmbedConfig()
	if baseURL == "" || model == "" {
		return CardRecallCheckResult{OK: false, Message: "请先在设置中配置量化连接与量化模型"}
	}
	if apiKey == "" {
		return CardRecallCheckResult{OK: false, Message: "请先填写量化 API Key"}
	}
	ok, err := a.testAIConnection(baseURL, apiKey, "TestVectorIndexConnection")
	if err != nil || !ok {
		// 原始错误（网络/HTTP 细节）由 testAIConnection 记入日志，不向用户透出以免困惑
		return CardRecallCheckResult{OK: false, Message: "量化服务连接失败，请检查服务是否已启动"}
	}
	return CardRecallCheckResult{OK: true, Message: ""}
}

// FetchAIModels 获取可用模型列表
// FetchAIModels 按指定 BaseURL/APIKey 获取模型列表（对话/量化连接共用）
func (a *App) FetchAIModels(baseURL, apiKey string) ([]string, error) {
	a.LogSvc.Logger.Debugw("FetchAIModels", fastlog.String("baseURL", baseURL), fastlog.String("key", "***"))
	cfg := services.AIConfig{BaseURL: baseURL, APIKey: apiKey}
	models, err := a.aiService.FetchModels(cfg)
	if err != nil {
		a.LogSvc.Logger.Errorw("FetchAIModels 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("FetchAIModels 成功", fastlog.Int("count", len(models)))
	return models, nil
}

// SelectAIChatFiles 打开文件对话框选择文本文件，校验并读取内容返回给 AI 聊天使用
func (a *App) SelectAIChatFiles() ([]AIChatFileResult, error) {
	a.LogSvc.Logger.Debugw("SelectAIChatFiles")
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title:           "选择要上传的文本文件",
		ShowHiddenFiles: false,
	})
	if err != nil {
		a.LogSvc.Logger.Errorw("SelectAIChatFiles 失败", fastlog.Error(err))
		return nil, fmt.Errorf("打开文件对话框失败: %w", err)
	}
	if len(paths) == 0 {
		return []AIChatFileResult{}, nil // 用户取消
	}

	return a.readAIChatFiles(paths), nil
}

// ReadAIChatFiles 直接根据文件路径列表校验并读取内容（拖拽上传用）
func (a *App) ReadAIChatFiles(paths []string) []AIChatFileResult {
	a.LogSvc.Logger.Debugw("ReadAIChatFiles", fastlog.Int("file_count", len(paths)))
	return a.readAIChatFiles(paths)
}

// readAIChatFiles 内部方法：校验、读取一组文件（按钮上传和拖拽上传共用），支持并发处理。
// 纯文本文件直接读取内容；办公文件通过 markitdown 转换为 Markdown 文本返回。
// 上传过程中通过 Wails Events 发射进度事件（import:ai-progress）。
func (a *App) readAIChatFiles(paths []string) []AIChatFileResult {
	a.LogSvc.Logger.Debugw("readAIChatFiles", fastlog.Int("file_count", len(paths)))
	maxSize := a.GetMaxFileSize()

	results := make([]AIChatFileResult, len(paths))
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 发射上传开始事件
	runtime.EventsEmit(a.ctx, "import:ai-progress", "start", len(paths))

	for i, p := range paths {
		wg.Add(1)
		go func(idx int, path string) {
			defer wg.Done()
			result := a.processAIChatFile(path, maxSize)
			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, p)
	}

	wg.Wait()

	// 统计结果并发射完成事件
	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Error == "" {
			successCount++
		} else {
			failCount++
		}
	}
	runtime.EventsEmit(a.ctx, "import:ai-progress", "complete", len(paths), successCount, failCount)

	return results
}

// processAIChatFile 处理单个 AI 聊天文件的上传逻辑。
func (a *App) processAIChatFile(path string, maxSize int64) AIChatFileResult {
	result := AIChatFileResult{Path: path, Name: filepath.Base(path)}

	// 1. 检查路径
	info, err := os.Stat(path)
	if err != nil {
		a.LogSvc.Logger.Errorw("processAIChatFile: 无法访问文件", fastlog.String("path", path), fastlog.Error(err))
		result.Error = "无法访问文件: " + err.Error()
		return result
	}

	// 2. 拒绝目录
	if info.IsDir() {
		a.LogSvc.Logger.Debugw("processAIChatFile: 拒绝目录", fastlog.String("path", path))
		result.Error = "不支持上传目录，请选择文件"
		return result
	}

	// 3. 文件大小限制（放在最前面）
	if info.Size() > maxSize {
		maxSizeMB := maxSize / (1024 * 1024)
		a.LogSvc.Logger.Debugw("processAIChatFile: 文件过大", fastlog.String("path", path), fastlog.Int64("size", info.Size()), fastlog.Int64("maxSize", maxSize))
		result.Error = fmt.Sprintf("文件过大（超过 %dMB），请选择小于 %dMB 的文件", maxSizeMB, maxSizeMB)
		return result
	}
	result.Size = info.Size()

	// 4. 文件类型判定与内容读取
	if converter.IsOfficeFile(path) {
		// 办公文件 → markitdown 转换
		a.LogSvc.Logger.Debugw("processAIChatFile: 转换办公文件", fastlog.String("path", path), fastlog.Int64("size", info.Size()))
		mdText, err := converter.ConvertToMarkdown(path)
		if err != nil {
			switch {
			case errors.Is(err, converter.ErrUnsupportedFormat):
				a.LogSvc.Logger.Debugw("processAIChatFile: 不支持的办公文件格式", fastlog.String("path", path))
				result.Error = "不支持的文件格式"
			case errors.Is(err, converter.ErrConversionTimeout):
				a.LogSvc.Logger.Warnw("processAIChatFile: 办公文件转换超时", fastlog.String("path", path), fastlog.Int64("size", info.Size()))
				result.Error = "文件转换超时（超过60秒）"
			default:
				a.LogSvc.Logger.Errorw("processAIChatFile: 办公文件转换失败", fastlog.String("path", path), fastlog.Error(err))
				result.Error = fmt.Sprintf("文件转换失败: %s", err.Error())
			}
			return result
		}
		result.Content = mdText
		a.LogSvc.Logger.Infow("processAIChatFile: 办公文件转换成功", fastlog.String("path", path), fastlog.Int("content_len", len(mdText)))
	} else if fs.IsBinaryPath(path) {
		// 二进制文件（非办公文件）→ 拒绝
		a.LogSvc.Logger.Debugw("processAIChatFile: 拒绝二进制文件", fastlog.String("path", path))
		result.Error = "不支持此类文件，请选择文本文件或办公文档后重试"
		return result
	} else {
		// 纯文本文件 → 直接读取
		a.LogSvc.Logger.Debugw("processAIChatFile: 直接读取文本文件", fastlog.String("path", path), fastlog.Int64("size", info.Size()))
		content, err := os.ReadFile(path)
		if err != nil {
			a.LogSvc.Logger.Errorw("processAIChatFile: 读取文本文件失败", fastlog.String("path", path), fastlog.Error(err))
			result.Error = "读取文件失败: " + err.Error()
			return result
		}
		result.Content = string(content)
	}

	return result
}

// CallAI 调用 AI 对话接口（非流式）
// 创建可取消的 context 并存入 aiStreamCancel，供 CancelAIStream 中途取消
func (a *App) CallAI(messages []services.Message) (string, error) {
	// 检查 AI 配置（URL / API Key / 模型三要素齐全）
	cfg := a.aiService.GetConfig()
	if cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		return "", fmt.Errorf("请先配置 AI 服务（API 地址 / API Key / 模型）")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	a.aiStreamCancel = cancel
	defer func() {
		cancel()
		a.aiStreamCancel = nil
	}()

	return a.aiService.CallAI(ctx, messages)
}

// formatFileSize 将字节数转为人类可读的文件大小字符串
func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}

// estimateTokens 估算文本的 token 数（与前端 estimateTokens 算法一致）
func estimateTokens(text string) int {
	chineseCount := 0
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			chineseCount++
		}
	}
	runes := []rune(text)
	otherCount := len(runes) - chineseCount
	return int(math.Ceil(float64(chineseCount)/1.5 + float64(otherCount)/4))
}

// estimateUserTokens 计算会话中 system 消息与最后一条 user 消息的估算 token 数
func estimateUserTokens(messages []services.Message) int {
	tokens := 0
	for _, msg := range messages {
		if msg.Role == "system" {
			tokens += estimateTokens(msg.Content)
		}
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			tokens += estimateTokens(messages[i].Content)
			break
		}
	}
	return tokens
}

// truncateAIMessages 加载并截断会话消息，保留 system 消息 + 最后 N 条 user/assistant 消息，
// 需要更新摘要时同步生成并向前端发送状态事件，新摘要当前轮即可注入。
// ctx 传入 AI 流的取消上下文，用户取消时摘要生成也随之取消。
func (a *App) truncateAIMessages(ctx context.Context, sessionID uint, logLabel string) []services.Message {
	// 加载会话消息
	messages := a.aiService.LoadAISessionMessages(sessionID)
	nonSystemBefore := 0
	for _, m := range messages {
		if m.Role != "system" {
			nonSystemBefore++
		}
	}

	windowSize := a.aiService.GetContextWindowSize()

	// 需要更新摘要时同步生成，阻塞当前对话，但前端能看到状态条反馈
	if nonSystemBefore > windowSize {
		// 先检查是否真的需要更新（diff >= windowSize），避免每次发空事件
		var checkSession models.AISession
		needUpdate := false
		if err := a.db.First(&checkSession, sessionID).Error; err == nil {
			needUpdate = nonSystemBefore-checkSession.SummaryMsgCount >= windowSize
		}

		if needUpdate {
			runtime.EventsEmit(a.ctx, "ai:summary-status", map[string]interface{}{
				"status":     "generating",
				"session_id": sessionID,
			})
		}

		updated := a.aiService.UpdateSessionSummary(ctx, sessionID, windowSize)

		if needUpdate {
			status := "done"
			if !updated {
				status = "skipped"
			}
			runtime.EventsEmit(a.ctx, "ai:summary-status", map[string]interface{}{
				"status":     status,
				"session_id": sessionID,
			})
		}
	}

	// 滑动窗口截断：只保留 system 消息 + 最后 N 条 user/assistant 消息
	messages = services.TruncateMessagesForLLM(messages, windowSize)

	// 读取已有会话摘要，注入到 system 消息之后、tail 消息之前
	var session models.AISession
	if err := a.db.First(&session, sessionID).Error; err == nil && session.SummaryContent != "" {
		summaryMsg := services.Message{
			Role:    "system",
			Content: "【历史对话摘要】\n" + session.SummaryContent,
		}
		insertIdx := 0
		for i, m := range messages {
			if m.Role == "system" {
				insertIdx = i + 1
			} else {
				break
			}
		}
		messages = append(messages[:insertIdx], append([]services.Message{summaryMsg}, messages[insertIdx:]...)...)
	}

	nonSystemAfter := 0
	hasSummary := false
	for _, m := range messages {
		if m.Role != "system" {
			nonSystemAfter++
		} else if strings.Contains(m.Content, "【历史对话摘要】") {
			hasSummary = true
		}
	}
	a.LogSvc.Logger.Debugw(logLabel,
		fastlog.Int("window_size", windowSize),
		fastlog.Int("non_system_before", nonSystemBefore),
		fastlog.Int("non_system_after", nonSystemAfter),
		fastlog.Int("total_after", len(messages)),
		fastlog.Bool("has_summary", hasSummary))
	return messages
}

// CancelAIStream 取消当前正在进行的 AI 流式调用
func (a *App) CancelAIStream() {
	a.LogSvc.Logger.Debugw("CancelAIStream")
	if a.aiStreamCancel != nil {
		a.aiStreamCancel()
		a.aiStreamCancel = nil
	}
}

// CallAIAgentStream Agent 模式流式对话绑定方法（基于 internal/agent 模块）。
// 深度思考通过 thinkingEnabled 传递（开启时 Agent 的 ChatModel 配置 reasoning_effort=high）；
// 去掉 searchSources / cardRecallEnabled 两个开关：联网搜索由 MCP 服务器工具执行，
// 本地卡片召回由 Agent 的 recall_notes 工具自行执行，调用方仅负责组装 Instruction、截断历史、落库与事件收发。
func (a *App) CallAIAgentStream(streamGen int, sessionID uint, userText string, thinkingEnabled bool, skillIds []string, referencedNoteIDs []uint, roleplayNoteIDs []uint, followUpRefContent string, uploadedFiles []AIChatFileResult, recallNotebookIDs []uint, userMsgID uint) {
	// 创建可取消的 ctx 并存入 a.aiStreamCancel，供停止按钮（CancelAIStream）取消
	ctx, cancel := context.WithCancel(context.Background())
	a.aiStreamCancel = cancel

	// 加载并截断会话消息，保留 system 消息 + 最后 N 条 user/assistant 消息
	messages := a.truncateAIMessages(ctx, sessionID, "AI Agent 滑动窗口截断")

	// 重新生成场景：前端传 userMsgID=0（重新生成不新建用户消息）。
	// 此处从截断后的消息中倒序找回末条用户消息 ID，用于 token 更新与
	// stream-done 回传（语义与 Agent 再生一致）。
	if userMsgID == 0 {
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				userMsgID = messages[i].ID
				break
			}
		}
	}

	// 流式调用放进 goroutine，避免阻塞 Wails 事件循环
	go func() {
		// 计时：总耗时（含工具调用/思考）从调用开始计；
		// 思考净时长由 agent.Run 按每轮 assistant 消息统计（排除工具执行时间）
		startTime := time.Now()

		// ── 组装 Instruction（系统提示词全文），内容与顺序对齐 Agent 流上下文注入 ──
		// 基础提示词：无技能时注入完整三层（身份层 + 规范边界层），
		// 有技能时仅注入规范边界层（身份层由技能 prompt 的角色定义替代）
		var instruction strings.Builder
		if len(skillIds) == 0 {
			instruction.WriteString(baseSystemPrompt)
		} else {
			instruction.WriteString(baseNormsBoundaries)
		}

		// 角色扮演上下文注入
		hasRoleplay := false
		for _, sid := range skillIds {
			if sid == "skill_roleplay" {
				hasRoleplay = true
				break
			}
		}
		var roleplayContext string
		if hasRoleplay && len(roleplayNoteIDs) > 0 {
			refCtx, err := a.noteService.BuildNoteRefContext(roleplayNoteIDs)
			if err == nil && refCtx != nil && refCtx.Context != "" {
				roleplayContext = refCtx.Context
			}
		}

		// 笔记引用上下文注入
		if len(referencedNoteIDs) > 0 {
			refCtx, err := a.noteService.BuildNoteRefContext(referencedNoteIDs)
			if err == nil && refCtx != nil && refCtx.Context != "" {
				refText := "以下是用户手动引用的笔记内容（来源：手动引用笔记），请参考这些内容回答：\n\n" + refCtx.Context
				instruction.WriteString("\n\n" + refText)
			}
		}

		// 追问引用内容注入
		if followUpRefContent != "" {
			refText := "用户正在追问以下内容：\n" + followUpRefContent
			if len([]rune(followUpRefContent)) > 500 {
				refText = "用户正在追问以下内容：\n" + string([]rune(followUpRefContent)[:500])
			}
			instruction.WriteString("\n\n" + refText)
		}

		// 上传文件内容注入
		if len(uploadedFiles) > 0 {
			var b strings.Builder
			b.WriteString("用户上传了以下文件内容（来源：上传文件），请基于这些文件内容回答用户的提问：\n")
			for _, f := range uploadedFiles {
				if f.Error != "" || f.Content == "" {
					continue
				}
				sizeStr := formatFileSize(f.Size)
				fmt.Fprintf(&b, "\n--- 文件: %s (%s) ---\n%s\n---", f.Name, sizeStr, f.Content)
			}
			if b.Len() > 0 {
				instruction.WriteString("\n\n" + b.String())
			}
		}

		// 技能提示词注入
		if len(skillIds) > 0 {
			// 从 skillIds 中提取翻译参数（格式: skill_translate:source:target）
			translateArgs := make(map[string]string)
			cleanSkillIds := make([]string, 0, len(skillIds))
			for _, id := range skillIds {
				if strings.HasPrefix(id, "skill_translate:") {
					parts := strings.SplitN(id, ":", 3)
					if len(parts) == 3 {
						translateArgs["source"] = parts[1]
						translateArgs["target"] = parts[2]
					}
					cleanSkillIds = append(cleanSkillIds, "skill_translate")
				} else {
					cleanSkillIds = append(cleanSkillIds, id)
				}
			}
			skillPrompt, err := a.aiService.GetSkillPrompts(cleanSkillIds, translateArgs)
			if err == nil && skillPrompt != "" {
				// 替换角色扮演占位符
				if hasRoleplay && roleplayContext != "" {
					skillPrompt = strings.ReplaceAll(skillPrompt, "{roleplay_context}", roleplayContext)
				}
				instruction.WriteString("\n\n" + skillPrompt)
			} else if err != nil {
				a.LogSvc.Logger.Errorw("获取技能提示词失败", fastlog.Error(err))
			}
		}

		// Agent 模式专用约束：本地知识优先 + 联网兜底的工具选择策略
		// 联网搜索由 MCP 服务器提供的搜索工具执行（内置 web_search 已移除），不指定具体工具名
		instruction.WriteString("\n\n【工具使用规范 - 本地知识优先与联网搜索】\n" +
			"1. 优先检索本地知识：除非问题明确需要实时/外部信息，否则回答前应先调用 recall_notes 工具检索用户本地笔记；本地笔记是用户自己的知识库，可信度最高，应作为首要信息源。\n" +
			"2. 本地知识不足再联网：当 recall_notes 未检索到相关内容、或返回内容不足以支撑完整回答时，再调用联网搜索工具（如 MCP 服务器提供的搜索工具）补充信息；不要在未检索本地笔记的情况下直接联网。\n" +
			"3. 明确联网需求直接联网：当用户明确要求“联网搜索/查询网上资料/查最新资讯”等，或问题明显属于需要实时信息（如新闻、天气、股价、最新政策、外部教程）时，直接调用联网搜索工具，无需先检索本地笔记。\n" +
			"4. 信息整合优先级：本地笔记信息优先于联网搜索结果，联网搜索结果优先于模型自身知识；引用时标注来源（如“你的笔记《XXX》中记录了……”或“根据搜索结果显示……”）；不同来源信息矛盾时如实说明差异。\n" +
			"5. 失败降级：recall_notes 因量化连接未配置等原因失败时，直接转联网搜索工具；recall_notes 已返回充足信息时，不再重复联网。\n")

		// Agent 模式专用约束：ask_user 反向提问工具强制调用规范
		instruction.WriteString("\n\n【工具使用规范 - ask_user 反向提问（强制调用）】\n" +
			"1. 以下场景必须调用 ask_user 工具向用户发起澄清提问，不得省略或绕过，严禁在缺少必要信息时擅自猜测后直接执行：①用户请求存在信息模糊、参数不明确、需求不具体（如未指明操作对象、数量、范围、目标、方案等）；②需要用户在多个选项或方案之间做选择（如多篇候选笔记、多个方案、多种执行方式）；③需要获取用户进一步确认或补充关键信息才能继续执行（含写操作前的确认）。\n" +
			"2. 一次只问一个问题，提供 2-6 个候选选项；严禁把猜测当作已确认事实继续后续操作，严禁用于闲聊或无意义的确认。\n" +
			"3. 调用 ask_user 工具前，先在回复正文中完整写出你的问题（正文即问句）；调用后本轮会暂停并等待用户回答，期间不要再输出任何内容。\n" +
			"4. 用户回答后（会作为 ask_user 工具的结果返回给你），结合你的问题与用户的回答继续完成用户的原始请求：直接给出最终回答，或继续调用后续工具（如写操作确认后携带 confirm=true 执行），不要重复提问。\n")

		// Agent 模式专用约束：写操作强制确认规范
		instruction.WriteString("\n\n【工具使用规范 - 写操作强制确认】\n" +
			"1. manage_note 的 update / edit / pin / move / add_tag / remove_tag 全部属于写操作，执行前必须先向用户确认修改意图：在回复正文中写明要执行的具体操作与影响范围（如“我准备把笔记 #3 的正文整篇替换为……，是否继续？”），并调用 ask_user 工具向用户发起确认，等待用户回答。\n" +
			"2. 只有用户明确同意后，才能携带 confirm=true 参数调用对应写操作工具执行；用户确认前不要调用写操作工具，也不要直接传 confirm=true。若先调用了写操作工具被拒绝（工具返回“该操作需要用户确认”提示），按上述流程补一次 ask_user 提问，用户同意后再携带 confirm=true 执行。\n" +
			"3. 用户明确拒绝或撤回指令时，不得执行对应写操作。create（创建笔记）是用户明确要求的创建指令，无需确认。\n")

		// 历史消息转换：跳过 system（基础提示词已并入 Instruction），
		// 截断后的 user/assistant 消息转为 agent.HistoryMessage
		history := make([]agent.HistoryMessage, 0, len(messages))
		for _, m := range messages {
			if m.Role == "system" {
				continue
			}
			history = append(history, agent.HistoryMessage{Role: m.Role, Content: m.Content})
		}

		// 如果已被用户取消（停止按钮），不再调用 Agent，避免白调用；
		// 取消时用户消息已入库，需重算会话 token 缓存（与常规 Agent 流一致）
		if a.handleAICancelled(ctx, sessionID, userMsgID, messages, streamGen) {
			return
		}

		// 读取禁用工具集合（JSON 数组字符串），解析失败按空处理（默认全部启用）
		var disabledTools []string
		if raw := a.settingService.Get("ai_agent_tools_disabled"); raw != "" {
			if err := json.Unmarshal([]byte(raw), &disabledTools); err != nil {
				disabledTools = nil
				a.LogSvc.Logger.Warnw("解析 ai_agent_tools_disabled 失败，按空处理（全部工具启用）",
					fastlog.String("raw", raw), fastlog.Error(err))
			}
		}

		// 调用 Agent 模块执行对话，事件流直接转发给前端（Agent 内部发 ai:stream-chunk / ai:tool-status）
		a.LogSvc.Logger.Debugw("AI Agent 流开始",
			fastlog.Int("history_count", len(history)),
			fastlog.Int("skill_count", len(skillIds)),
			fastlog.Int("disabled_tool_count", len(disabledTools)),
		)
		result, err := a.AgentSvc.Run(ctx, agent.Request{
			SessionID:         sessionID,
			UserText:          userText,
			History:           history,
			Instruction:       instruction.String(),
			ThinkingEnabled:   thinkingEnabled,
			SkillIDs:          skillIds,
			RecallNotebookIDs: recallNotebookIDs,
			UserMsgID:         userMsgID,
			DisabledTools:     disabledTools,
		}, func(ev, data string) {
			// Agent 事件统一携带 streamGen（首参），与 stream-done/stream-error/agent-result
			// 已有的 gen 参数形态一致：前端按代过滤，防止切换/并发流串扰
			runtime.EventsEmit(a.ctx, ev, streamGen, data)
		})

		// 失败处理：经 ClassifyError 转中文提示（与常规 Agent 流错误分支一致）
		if err != nil {
			// 用户取消导致的结束：补发完成事件确保前端清理气泡，并刷新 token 缓存
			if a.handleAICancelled(ctx, sessionID, userMsgID, messages, streamGen) {
				return
			}
			a.LogSvc.Logger.Errorw("AI Agent 流错误", fastlog.Error(err))
			aiErr := aierrors.ClassifyError(err)
			if aiErr == nil {
				// context.Canceled 等非错误情况，补发完成事件
				runtime.EventsEmit(a.ctx, "ai:stream-done", streamGen, "", 0.0, 0.0, 0, 0, 0, 0, 0)
				return
			}
			// 与常规 Agent 流错误分支一致：附带 userTokens 供前端更新用户消息气泡 token 显示
			runtime.EventsEmit(a.ctx, "ai:stream-error", streamGen, aiErr.ToJSON(), estimateUserTokens(messages))
			return
		}

		// 成功：保存 assistant 消息。
		// RecallCards（召回卡片）/ ToolCalls（工具调用链）
		// 分别写入 recall_cards / tool_calls 字段。
		// 思考净时长（agent.Run 内按轮统计，排除工具执行时间）与总耗时（调用开始到流结束）
		elapsedThinking := result.ThinkingElapsed
		elapsedTotal := time.Since(startTime).Seconds()

		// token 统计：优先用 Agent 各轮真实 usage（输入=ΣPromptTokens 记用户消息，
		// 输出=ΣCompletionTokens 记 assistant 消息，含中间轮工具参数/思考/工具结果回填）；
		// 仅当 provider 未返回 usage（两项均为 0）时回退现状估算，保持统计不为空
		userTokens := estimateUserTokens(messages)
		assistantTokens := estimateTokens(result.Content)
		// 深度思考链计入 assistant token（与常规 Agent 流一致）
		if thinkingEnabled && result.ReasoningContent != "" {
			assistantTokens += estimateTokens(result.ReasoningContent)
		}
		if result.PromptTokens > 0 {
			userTokens = result.PromptTokens
		}
		if result.CompletionTokens > 0 {
			assistantTokens = result.CompletionTokens
		}
		totalTokens := userTokens + assistantTokens

		// 保存 assistant 消息到数据库（与常规 Agent 流一致）：
		// 耗时与深度思考链一起落库，切换会话后历史消息仍展示 ⏱ 耗时与思考秒数
		assistantMsg := services.Message{
			Role:             "assistant",
			Content:          result.Content,
			ReasoningContent: result.ReasoningContent,
			ThinkingElapsed:  elapsedThinking,
			TotalElapsed:     elapsedTotal,
			Tokens:           assistantTokens,
			RecallCards:      result.RecallCards,
			ToolCalls:        result.ToolCalls,
		}
		assistantMsgID, saveErr := a.aiService.SaveAIMessage(sessionID, assistantMsg)
		if saveErr != nil {
			a.LogSvc.Logger.Errorw("保存 assistant 消息失败", fastlog.Error(saveErr))
		}

		// 更新用户消息的 tokens 为完整上下文 token 数（含 system 上下文），
		// 并重新计算会话累计 token 持久化
		_ = a.aiService.UpdateAIMessageTokens(userMsgID, userTokens)
		accumulated, _ := a.aiService.SumSessionTokens(sessionID)
		_ = a.aiService.UpdateSessionContextTokens(sessionID, accumulated)

		// 通过 stream-done 一并返回 token 数据和消息 ID（与常规 Agent 流一致）
		a.LogSvc.Logger.Infow("AI Agent 流完成",
			fastlog.Int("total_tokens", totalTokens),
			fastlog.Float64("elapsed_total", elapsedTotal),
		)
		// agent-result 事件先于 stream-done 发送：把结构化结果（召回卡片/工具调用链/思考链）
		// 回传前端，供流式完成后立即渲染（无需切换会话），并供 chatHistory.push 落库前使用
		runtime.EventsEmit(a.ctx, "ai:agent-result", streamGen, result.RecallCards, result.ToolCalls, result.ReasoningContent)
		runtime.EventsEmit(a.ctx, "ai:stream-done", streamGen, result.Content, elapsedThinking, elapsedTotal, totalTokens, userTokens, assistantTokens, userMsgID, assistantMsgID)
	}()
}

// handleAICancelled 处理用户取消 AI 流的收尾工作：刷新用户消息 token 缓存、
// 重算会话累计 token、补发 stream-done 事件确保前端清理气泡。
// 供 CallAIAgentStream 中 Agent 调用前后的两处取消检查复用。
func (a *App) handleAICancelled(ctx context.Context, sessionID, userMsgID uint, messages []services.Message, streamGen int) bool {
	if ctx.Err() == nil {
		return false
	}
	if userMsgID > 0 {
		_ = a.aiService.UpdateAIMessageTokens(userMsgID, estimateUserTokens(messages))
	}
	accumulated, _ := a.aiService.SumSessionTokens(sessionID)
	_ = a.aiService.UpdateSessionContextTokens(sessionID, accumulated)
	runtime.EventsEmit(a.ctx, "ai:stream-done", streamGen, "", 0.0, 0.0, 0, 0, 0, 0, 0)
	return true
}

// GetAgentTools 返回 Agent 工具清单（含内置工具与已预热 MCP 服务器的工具），
// 含中文说明与当前启用状态，供前端工具开关配置使用。
// 禁用集合读取设置 ai_agent_tools_disabled（JSON 数组字符串），解析失败按空处理（默认全部启用）。
func (a *App) GetAgentTools() []agent.ToolMeta {
	var disabled []string
	if raw := a.settingService.Get("ai_agent_tools_disabled"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &disabled); err != nil {
			disabled = nil
		}
	}
	disabledSet := make(map[string]bool, len(disabled))
	for _, name := range disabled {
		disabledSet[name] = true
	}
	metas := tools.BuiltinTools()
	result := make([]agent.ToolMeta, 0, len(metas))
	for _, m := range metas {
		result = append(result, agent.ToolMeta{Name: m.Name, Label: m.Label, Enabled: !disabledSet[m.Name]})
	}
	// 追加已预热 MCP 服务器的工具（未预热时不显示，不阻塞等待）
	if a.mcpPool != nil {
		mcpTools := a.mcpPool.ListToolMetas()
		for _, mt := range mcpTools {
			label := mt.ServerName + " 的 " + strings.TrimPrefix(mt.FullName, "mcp_"+mt.ServerName+"_")
			result = append(result, agent.ToolMeta{
				Name:    mt.FullName,
				Label:   label,
				Enabled: !disabledSet[mt.FullName],
			})
		}
	}
	return result
}

// GetMCPServers 返回全部 MCP 服务器配置（按 sort_order, id 升序），供设置页展示与管理
func (a *App) GetMCPServers() []models.MCPServer {
	servers, err := a.mcpServerService.List()
	if err != nil {
		a.LogSvc.Logger.Errorw("获取 MCP 服务器列表失败", fastlog.Error(err))
		return nil
	}
	return servers
}

// SaveMCPServer 新增（ID==0）或更新 MCP 服务器配置；校验失败时返回可直接展示的中文错误
func (a *App) SaveMCPServer(server models.MCPServer) error {
	return a.mcpServerService.Save(&server)
}

// DeleteMCPServer 按 ID 删除 MCP 服务器配置
func (a *App) DeleteMCPServer(id uint) error {
	return a.mcpServerService.Delete(id)
}

// TestMCPServerResult MCP 服务器连接测试结果（Wails 绑定需用结构体传复杂数据）
type TestMCPServerResult struct {
	OK      bool   `json:"ok"`       // 是否连接可用
	ToolNum int    `json:"tool_num"` // 连接成功时发现的工具数
	Message string `json:"message"`  // 中文提示文案（失败时为具体原因）
}

// toMCPServerConfig models.MCPServer → mcpserver.Server（字段映射与 mcpserver.LoadFromDB 一致）
func toMCPServerConfig(m models.MCPServer) mcpserver.Server {
	return mcpserver.Server{
		Name:      m.Name,
		Transport: m.Transport,
		Command:   m.Command,
		Args:      m.Args,
		Env:       m.Env,
		URL:       m.URL,
		Headers:   m.Headers,
		Enabled:   m.Enabled,
	}
}

// TestMCPServer 按 ID 加载服务器配置并实测连接（连接 + 握手 + 工具发现）。
// 无论服务器是否启用均可测试；连接/发现内部各自带 ConnectTimeout 超时兜底，不会卡死。
func (a *App) TestMCPServer(id uint) TestMCPServerResult {
	rec, err := a.mcpServerService.Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return TestMCPServerResult{OK: false, Message: "MCP 服务器不存在或已被删除"}
		}
		return TestMCPServerResult{OK: false, Message: "查询 MCP 服务器配置失败"}
	}
	sess, err := mcpserver.OpenSession(context.Background(), toMCPServerConfig(*rec))
	if err != nil {
		return TestMCPServerResult{OK: false, Message: err.Error()} // 错误文案已中文化
	}
	defer func() {
		if err := sess.Close(); err != nil {
			a.LogSvc.Logger.Debugw("关闭 MCP 测试会话失败", fastlog.Error(err))
		}
	}()
	return TestMCPServerResult{OK: true, ToolNum: len(sess.Tools), Message: "连接成功"}
}

// WarmupMCPServers 预热/同步全局 MCP 连接池（内部 Reconcile）：
// 关闭池中已停用/删除的服务器连接，预热启用的服务器（新增/变更/复用）。
// 首次进入 AI 助手模块、设置页任何 MCP 操作后调用；幂等（已预热且配置未变时零网络开销）。
// 返回汇总结果供前端一条通知展示；池为 nil（未初始化）时返回空结果。
func (a *App) WarmupMCPServers() mcpserver.WarmupResult {
	if a.mcpPool == nil {
		return mcpserver.WarmupResult{}
	}
	cfg, err := mcpserver.LoadFromDB(a.db)
	if err != nil {
		a.LogSvc.Logger.Warnw("MCP 服务器配置读取失败，跳过预热", fastlog.Error(err))
		return mcpserver.WarmupResult{Failed: 1, FailedMsgs: []string{"MCP 服务器配置读取失败"}}
	}
	return a.mcpPool.Reconcile(context.Background(), cfg.Servers)
}

// aiTextOpSystemPrompt 根据 operation 构造 AI 写作操作的 system prompt
func aiTextOpSystemPrompt(operation string) (string, error) {
	// 所有操作统一追加分段约束：避免模型输出超长无换行的单段文本
	paragraphClause := "输出请保持清晰的分段（用换行分隔段落），不要输出超长无换行的文本。"
	switch operation {
	case "polish":
		return "你是一位专业的写作助手。请润色以下文本，改进其语法、表达和风格，使其更加流畅和专业。只返回润色后的结果，不要添加任何解释或额外内容。" + paragraphClause, nil
	case "continue":
		return "你是一位专业的写作助手。请根据以下文本的内容和风格，自然地续写下去。只返回续写的内容，不要重复原文。" + paragraphClause, nil
	case "expand":
		return "你是一位专业的写作助手。请扩写以下文本，增加更多细节、例子和说明，使内容更加丰富完整。只返回扩写后的结果，不要添加任何解释。" + paragraphClause, nil
	case "condense":
		return "你是一位专业的写作助手。请缩写以下文本，保留所有关键信息和核心观点，删除冗余内容。只返回缩写后的结果，不要添加任何解释。" + paragraphClause, nil
	case "proofread":
		return "你是一位专业的校对编辑。请校对以下文本，修正所有语法错误、拼写错误和标点符号问题。只返回校对后的结果，不要添加任何解释。" + paragraphClause, nil
	case "rewrite":
		return "你是一位专业的写作助手。请改写以下文本，保持原意不变，但使用不同的表达方式和句式结构。只返回改写后的结果，不要添加任何解释。" + paragraphClause, nil
	case "translate":
		return "你是一位专业的翻译。请将以下文本翻译成中文，保持原文的语气和风格。只返回翻译结果，不要添加任何解释。" + paragraphClause, nil
	case "translate-en":
		return "你是一位专业的翻译。请将以下文本翻译成英文，保持原文的语气和风格。只返回翻译结果，不要添加任何解释。" + paragraphClause, nil
	default:
		return "", fmt.Errorf("不支持的操作: %s", operation)
	}
}

// AITextOperationStream 流式 AI 写作操作（润色/续写/扩写/缩写/校对/改写/翻译等）
// fire-and-forget：无返回值，生成块通过 ai:aiop-chunk 事件推送，终态通过 ai:aiop-done / ai:aiop-error 通知。
// 取消通过 CancelAIEditorOperation 触发（独立于聊天流的 CancelAIStream，避免误杀后台对话）。
func (a *App) AITextOperationStream(streamGen int, text string, operation string) {
	a.LogSvc.Logger.Debugw("AITextOperationStream", fastlog.String("operation", operation), fastlog.Int("text_len", len(text)), fastlog.Int("stream_gen", streamGen))

	// 1. 检查 AI 配置（URL / API Key / 模型三要素齐全）
	cfg := a.aiService.GetConfig()
	if cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		runtime.EventsEmit(a.ctx, "ai:aiop-error", streamGen, "请先配置 AI 服务（API 地址 / API Key / 模型）")
		return
	}

	// 2. 根据 operation 构造 system prompt
	systemPrompt, err := aiTextOpSystemPrompt(operation)
	if err != nil {
		runtime.EventsEmit(a.ctx, "ai:aiop-error", streamGen, err.Error())
		return
	}

	// 3. 构造 messages
	messages := []services.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: text},
	}

	// 4. 可取消的 context（60s 超时兜底），存入独立字段 aiEditorCancel。
	// 不能 defer cancel()：本绑定 fire-and-forget 立即返回，defer 会立刻取消流；
	// cancel 存字段供 CancelAIEditorOperation 后续调用（流结束后遗留无害，下次操作覆盖）。
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	a.aiEditorCancel = cancel

	// 5. 流式调用放 goroutine，逐块发射事件
	go func() {
		a.aiService.CallAIStream(ctx, messages, false,
			func(chunk string) {
				runtime.EventsEmit(a.ctx, "ai:aiop-chunk", streamGen, chunk)
			},
			func(string) {}, // OnThinking 忽略（写作操作不需要思维链）
			func(content string, _, _ float64) {
				runtime.EventsEmit(a.ctx, "ai:aiop-done", streamGen, content)
			},
			func(errMsg string) {
				runtime.EventsEmit(a.ctx, "ai:aiop-error", streamGen, errMsg)
			},
		)
		// 兜底：取消/超时导致 OnDone/OnError 均未触发时，补发终态事件保证前端清理
		if ctx.Err() != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				runtime.EventsEmit(a.ctx, "ai:aiop-error", streamGen, "AI 处理超时，请重试")
			} else { // context.Canceled：用户点击圆球取消
				runtime.EventsEmit(a.ctx, "ai:aiop-done", streamGen, "")
			}
		}
	}()
}

// CancelAIEditorOperation 取消编辑器 AI 写作流式操作（仅取消编辑器流，不影响聊天流）
func (a *App) CancelAIEditorOperation() {
	a.LogSvc.Logger.Debugw("CancelAIEditorOperation")
	if a.aiEditorCancel != nil {
		a.aiEditorCancel()
		a.aiEditorCancel = nil
	}
}

// GetAISessions 获取 AI 会话列表
func (a *App) GetAISessions() []services.AISessionSummary {
	a.LogSvc.Logger.Debugw("GetAISessions")
	return a.aiService.GetAISessions()
}

// TogglePinAISession 切换会话置顶状态
func (a *App) TogglePinAISession(id uint) error {
	a.LogSvc.Logger.Debugw("TogglePinAISession", fastlog.Uint("id", id))
	if err := a.aiService.TogglePinAISession(id); err != nil {
		a.LogSvc.Logger.Errorw("TogglePinAISession 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("TogglePinAISession 成功", fastlog.Uint("id", id))
	return nil
}

// CreateAISession 创建新 AI 会话，返回会话 ID
func (a *App) CreateAISession() uint {
	a.LogSvc.Logger.Debugw("CreateAISession")
	id := a.aiService.CreateAISession()
	a.LogSvc.Logger.Infow("CreateAISession 成功", fastlog.Uint("id", id))
	return id
}

// DeleteAISession 删除 AI 会话及所有消息
func (a *App) DeleteAISession(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteAISession", fastlog.Uint("id", id))
	// 释放该会话的 Agent 实例（取消等待中的反问 run），防止悬挂
	a.AgentSvc.ReleaseSession(id)
	if err := a.aiService.DeleteAISession(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteAISession 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteAISession 成功", fastlog.Uint("id", id))
	return nil
}

// RenameAISession 重命名 AI 会话
func (a *App) RenameAISession(id uint, title string) error {
	a.LogSvc.Logger.Debugw("RenameAISession", fastlog.Uint("id", id))
	if err := a.aiService.RenameAISession(id, title); err != nil {
		a.LogSvc.Logger.Errorw("RenameAISession 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("RenameAISession 成功", fastlog.Uint("id", id))
	return nil
}

// LoadAISessionMessages 加载 AI 会话的所有消息
func (a *App) LoadAISessionMessages(id uint) []services.Message {
	a.LogSvc.Logger.Debugw("LoadAISessionMessages", fastlog.Uint("id", id))
	return a.aiService.LoadAISessionMessages(id)
}

// LoadAISessionMessagesPaginated 分页加载会话消息
func (a *App) LoadAISessionMessagesPaginated(sessionID uint, limit int, beforeID uint) []services.Message {
	a.LogSvc.Logger.Debugw("LoadAISessionMessagesPaginated", fastlog.Uint("sessionID", sessionID), fastlog.Int("limit", limit), fastlog.Uint("beforeID", beforeID))
	return a.aiService.LoadAISessionMessagesPaginated(sessionID, limit, beforeID)
}

// TruncateAISessionAtMessage 删除指定消息及该消息之后的所有消息
func (a *App) TruncateAISessionAtMessage(sessionID uint, msgID uint) error {
	a.LogSvc.Logger.Debugw("TruncateAISessionAtMessage", fastlog.Uint("sessionID", sessionID), fastlog.Uint("msgID", msgID))
	if err := a.aiService.TruncateAISessionAtMessage(sessionID, msgID); err != nil {
		a.LogSvc.Logger.Errorw("TruncateAISessionAtMessage 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("TruncateAISessionAtMessage 成功", fastlog.Uint("sessionID", sessionID), fastlog.Uint("msgID", msgID))
	return nil
}

// TruncateAISessionAfterMessage 删除指定消息之后的所有消息（保留该消息本身）
func (a *App) TruncateAISessionAfterMessage(sessionID uint, msgID uint) error {
	a.LogSvc.Logger.Debugw("TruncateAISessionAfterMessage", fastlog.Uint("sessionID", sessionID), fastlog.Uint("msgID", msgID))
	if err := a.aiService.TruncateAISessionAfterMessage(sessionID, msgID); err != nil {
		a.LogSvc.Logger.Errorw("TruncateAISessionAfterMessage 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("TruncateAISessionAfterMessage 成功", fastlog.Uint("sessionID", sessionID), fastlog.Uint("msgID", msgID))
	return nil
}

// GetSessionContextTokens 返回指定会话的 context_tokens 字段值
func (a *App) GetSessionContextTokens(sessionID uint) (int, error) {
	a.LogSvc.Logger.Debugw("GetSessionContextTokens", fastlog.Uint("sessionID", sessionID))
	tokens, err := a.aiService.GetSessionContextTokens(sessionID)
	if err != nil {
		a.LogSvc.Logger.Errorw("GetSessionContextTokens 失败", fastlog.Error(err))
		return 0, err
	}
	return tokens, nil
}

// ReplaceAISessionMessages 原子替换指定会话的所有消息（清空 + 批量写入）
func (a *App) ReplaceAISessionMessages(sessionID uint, messages []services.Message) error {
	a.LogSvc.Logger.Debugw("ReplaceAISessionMessages", fastlog.Uint("sessionID", sessionID), fastlog.Int("message_count", len(messages)))
	if err := a.aiService.ReplaceAISessionMessages(sessionID, messages); err != nil {
		a.LogSvc.Logger.Errorw("ReplaceAISessionMessages 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("ReplaceAISessionMessages 成功", fastlog.Uint("sessionID", sessionID))
	return nil
}

// SaveAIMessages 保存一轮 AI 对话消息到指定会话
func (a *App) SaveAIMessages(sessionID uint, messages []services.Message) error {
	a.LogSvc.Logger.Debugw("SaveAIMessages", fastlog.Uint("sessionID", sessionID), fastlog.Int("message_count", len(messages)))
	if err := a.aiService.SaveAIMessages(sessionID, messages); err != nil {
		a.LogSvc.Logger.Errorw("SaveAIMessages 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SaveAIMessages 成功", fastlog.Uint("sessionID", sessionID))
	return nil
}

// ClearAISessionMessages 清空 AI 会话的所有消息（不删会话）
func (a *App) ClearAISessionMessages(sessionID uint) error {
	a.LogSvc.Logger.Debugw("ClearAISessionMessages", fastlog.Uint("sessionID", sessionID))
	// 释放该会话的 Agent 实例（取消等待中的反问 run），防止悬挂
	a.AgentSvc.ReleaseSession(sessionID)
	if err := a.aiService.ClearAISessionMessages(sessionID); err != nil {
		a.LogSvc.Logger.Errorw("ClearAISessionMessages 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("ClearAISessionMessages 成功", fastlog.Uint("sessionID", sessionID))
	return nil
}

// AnswerAskUser 投递用户对 Agent 反问（ask_user）的回答，恢复同一轮 ReAct 循环：
// 答案作为工具结果返回给模型继续完成原始请求（同轮传输，不落库为新用户消息）。
func (a *App) AnswerAskUser(sessionID uint, answer string) error {
	if a.AgentSvc == nil {
		return errors.New("agent 服务未就绪")
	}
	return a.AgentSvc.AnswerAskUser(sessionID, answer)
}

// UpdateAIMessageContent 更新指定 AI 消息的内容
func (a *App) UpdateAIMessageContent(id uint, content string) error {
	a.LogSvc.Logger.Debugw("UpdateAIMessageContent", fastlog.Uint("id", id))
	if err := a.aiService.UpdateAIMessageContent(id, content); err != nil {
		a.LogSvc.Logger.Errorw("UpdateAIMessageContent 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("UpdateAIMessageContent 成功", fastlog.Uint("id", id))
	return nil
}

// SaveAIMessageResult SaveAIMessage 的返回结果
type SaveAIMessageResult struct {
	MsgID  uint `json:"msgID"`
	Tokens int  `json:"tokens"`
}

// SaveAIMessage 保存单条 AI 消息到指定会话，返回消息 ID 和 token 数
// 由前端在发送前预先保存用户消息，确保前端能立即拿到 msgId 和 tokens
// maxAIMessageChars AI 消息单条字符上限（按 rune 计）。
// 与前端 MAX_AI_INPUT_CHARS、Agent 工具 maxToolLongText 的 20000 约定保持一致，
// 防止海量内容（粘贴/脚本绕过前端拦截）撑爆 LLM 上下文窗口。
const maxAIMessageChars = 20000

// SaveAIMessage 保存一条 AI 会话消息（前端仅以 user 角色调用；AI 回复由后端直接
// 经 aiService.SaveAIMessage 落库，不经本绑定，故校验只作用于用户消息）。
// meta 为用户消息附加上下文 JSON（引用笔记 / 上传文件 / 激活技能等），存于 ai_messages.meta 列，
// 不参与 LLM 输入；空字符串 / "[]" 均表示无附加上下文。
func (a *App) SaveAIMessage(sessionID uint, content string, role string, meta string) (SaveAIMessageResult, error) {
	a.LogSvc.Logger.Debugw("SaveAIMessage", fastlog.Uint("sessionID", sessionID), fastlog.String("role", role), fastlog.Int("meta_len", len(meta)))
	if len([]rune(content)) > maxAIMessageChars {
		return SaveAIMessageResult{}, fmt.Errorf("消息内容过长（上限 %d 字符，当前 %d 字符）", maxAIMessageChars, len([]rune(content)))
	}
	tokens := estimateTokens(content)
	msg := services.Message{
		Role:    role,
		Content: content,
		Tokens:  tokens,
		Meta:    meta,
	}
	msgID, err := a.aiService.SaveAIMessage(sessionID, msg)
	if err != nil {
		a.LogSvc.Logger.Errorw("SaveAIMessage 失败", fastlog.Error(err))
		return SaveAIMessageResult{}, err
	}
	a.LogSvc.Logger.Infow("SaveAIMessage 成功", fastlog.Uint("msgID", msgID), fastlog.Uint("sessionID", sessionID), fastlog.Int("meta_len", len(meta)))
	return SaveAIMessageResult{MsgID: msgID, Tokens: tokens}, nil
}

// DeleteAIMessage 按 ID 删除单条 AI 消息
func (a *App) DeleteAIMessage(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteAIMessage", fastlog.Uint("id", id))
	if err := a.aiService.DeleteAIMessage(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteAIMessage 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteAIMessage 成功", fastlog.Uint("id", id))
	return nil
}

// UpdateAIMessageMeta 更新指定用户消息的 meta 字段（仅作用于 role=user 的消息）
// 用于"编辑消息"或"重新生成"时同步最新工具栏上下文；assistant 消息不在更新范围内
// meta 为 JSON 字符串（引用笔记 / 上传文件 / 技能等），空字符串表示清空附加上下文
// 返回 error 当且仅当 msgID 不存在或对应消息非 user 角色（避免误改 assistant 消息）
func (a *App) UpdateAIMessageMeta(msgID uint, meta string) error {
	a.LogSvc.Logger.Debugw("UpdateAIMessageMeta", fastlog.Uint("msgID", msgID), fastlog.Int("meta_len", len(meta)))
	res := a.db.Model(&models.AIMessage{}).
		Where("id = ? AND role = ?", msgID, "user").
		Update("meta", meta)
	if err := res.Error; err != nil {
		a.LogSvc.Logger.Errorw("UpdateAIMessageMeta 失败", fastlog.Error(err))
		return fmt.Errorf("更新用户消息附加上下文失败: %w", err)
	}
	if res.RowsAffected == 0 {
		a.LogSvc.Logger.Warnw("UpdateAIMessageMeta 未命中", fastlog.Uint("msgID", msgID))
		return fmt.Errorf("未找到对应的用户消息（msgID=%d 可能不存在或不是用户消息）", msgID)
	}
	a.LogSvc.Logger.Infow("UpdateAIMessageMeta 成功", fastlog.Uint("msgID", msgID), fastlog.Int("meta_len", len(meta)))
	return nil
}

// DeleteAIMessagesAfter 删除指定会话中在指定消息之后的所有消息
func (a *App) DeleteAIMessagesAfter(sessionID uint, messageID uint) (int64, error) {
	a.LogSvc.Logger.Debugw("DeleteAIMessagesAfter", fastlog.Uint("sessionID", sessionID), fastlog.Uint("messageID", messageID))
	count, err := a.aiService.DeleteAIMessagesAfter(sessionID, messageID)
	if err != nil {
		a.LogSvc.Logger.Errorw("DeleteAIMessagesAfter 失败", fastlog.Error(err))
		return 0, err
	}
	a.LogSvc.Logger.Infow("DeleteAIMessagesAfter 成功", fastlog.Int64("count", count))
	return count, nil
}

// UpdateSessionContextTokens 更新会话的上下文 Token 数
func (a *App) UpdateSessionContextTokens(sessionID uint, tokens int) error {
	a.LogSvc.Logger.Debugw("UpdateSessionContextTokens", fastlog.Uint("sessionID", sessionID), fastlog.Int("tokens", tokens))
	if err := a.aiService.UpdateSessionContextTokens(sessionID, tokens); err != nil {
		a.LogSvc.Logger.Errorw("UpdateSessionContextTokens 失败", fastlog.Error(err))
		return err
	}
	return nil
}

// SaveSessionConfig 保存会话操作栏配置
func (a *App) SaveSessionConfig(sessionID uint, cfg services.SessionConfig) error {
	a.LogSvc.Logger.Debugw("SaveSessionConfig", fastlog.Uint("sessionID", sessionID))
	if err := a.aiService.SaveSessionConfig(sessionID, cfg); err != nil {
		a.LogSvc.Logger.Errorw("SaveSessionConfig 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SaveSessionConfig 成功", fastlog.Uint("sessionID", sessionID))
	return nil
}

// LoadSessionConfig 加载会话操作栏配置
func (a *App) LoadSessionConfig(sessionID uint) services.SessionConfig {
	a.LogSvc.Logger.Debugw("LoadSessionConfig", fastlog.Uint("sessionID", sessionID))
	return a.aiService.LoadSessionConfig(sessionID)
}

// ClearAllAISessions 清空所有 AI 会话及消息
func (a *App) ClearAllAISessions() error {
	a.LogSvc.Logger.Debugw("ClearAllAISessions")
	// 释放全部会话的 Agent 实例（取消等待中的反问 run），防止清空后僵尸 run 泄漏
	a.AgentSvc.ReleaseAll()
	if err := a.aiService.ClearAllAISessions(); err != nil {
		a.LogSvc.Logger.Errorw("ClearAllAISessions 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("ClearAllAISessions 成功")
	return nil
}

// UpdateLastUserMessageTokens 更新指定会话中最后一条用户消息的 tokens
func (a *App) UpdateLastUserMessageTokens(sessionID uint, tokens int) error {
	a.LogSvc.Logger.Debugw("UpdateLastUserMessageTokens", fastlog.Uint("sessionID", sessionID), fastlog.Int("tokens", tokens))
	if err := a.aiService.UpdateLastUserMessageTokens(sessionID, tokens); err != nil {
		a.LogSvc.Logger.Errorw("UpdateLastUserMessageTokens 失败", fastlog.Error(err))
		return err
	}
	return nil
}

// SaveAIMessageAsNote 将 AI 消息内容保存为笔记（归入默认笔记本）
func (a *App) SaveAIMessageAsNote(content string) (*models.Note, error) {
	a.LogSvc.Logger.Debugw("SaveAIMessageAsNote")
	if strings.TrimSpace(content) == "" {
		err := fmt.Errorf("内容不能为空")
		a.LogSvc.Logger.Errorw("SaveAIMessageAsNote 失败", fastlog.Error(err))
		return nil, err
	}
	// 自动生成标题：取第一行，截断到 50 字符
	title := generateNoteTitle(content)
	// 保存到默认笔记本（id=1）
	note, err := a.noteService.CreateWithNotebook(title, content, ".md", 1)
	if err != nil {
		a.LogSvc.Logger.Errorw("SaveAIMessageAsNote 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("SaveAIMessageAsNote 成功", fastlog.Uint("id", note.ID))
	return note, nil
}

// generateNoteTitle 从内容中自动生成笔记标题
func generateNoteTitle(content string) string {
	// 取第一个有内容的行，去掉首尾空白
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			// 截断到 50 字符
			runes := []rune(trimmed)
			if len(runes) > 50 {
				return string(runes[:50]) + "..."
			}
			return trimmed
		}
	}
	return "AI 回复"
}

// GetSystemFonts 获取系统已安装的字体族列表
func (a *App) GetSystemFonts() []string {
	return fontutil.GetFonts()
}

// ==================== 排序与分页设置绑定方法 ====================

// GetSortOrder 获取排序方式设置
func (a *App) GetSortOrder() string {
	a.LogSvc.Logger.Debugw("GetSortOrder")
	return a.settingService.Get("sort_order")
}

// SetSortOrder 保存排序方式设置
func (a *App) SetSortOrder(order string) error {
	a.LogSvc.Logger.Debugw("SetSortOrder", fastlog.String("order", order))
	if err := a.settingService.Set("sort_order", order); err != nil {
		a.LogSvc.Logger.Errorw("SetSortOrder 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SetSortOrder 成功")
	return nil
}

// GetPageSize 获取分页大小设置
func (a *App) GetPageSize() int {
	a.LogSvc.Logger.Debugw("GetPageSize")
	size := a.settingService.Get("page_size")
	n, err := strconv.Atoi(size)
	if err != nil || n < 20 || n > 100 {
		return 20
	}
	return n
}

// SetPageSize 保存分页大小设置
func (a *App) SetPageSize(size int) error {
	a.LogSvc.Logger.Debugw("SetPageSize", fastlog.Int("size", size))
	if err := a.settingService.Set("page_size", strconv.Itoa(size)); err != nil {
		a.LogSvc.Logger.Errorw("SetPageSize 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("SetPageSize 成功")
	return nil
}

// ==================== 版本与链接绑定方法 ====================

// GetVersion 获取应用版本号
func (a *App) GetVersion() string {
	a.LogSvc.Logger.Debugw("GetVersion")
	return verman.V.GitVersion
}

// ExportNoteAsMarkdown 导出单条笔记为 Markdown 文件，弹出保存对话框让用户选择路径
func (a *App) ExportNoteAsMarkdown(id uint) (string, error) {
	a.LogSvc.Logger.Debugw("ExportNoteAsMarkdown", fastlog.Uint("id", id))
	note, err := a.noteService.GetByID(id)
	if err != nil {
		a.LogSvc.Logger.Errorw("ExportNoteAsMarkdown 失败", fastlog.Error(err))
		return "", fmt.Errorf("笔记不存在: %w", err)
	}

	defaultName := sanitizeFilename(note.Title) + note.FileExt
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出笔记",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "笔记文件 (*" + note.FileExt + ")", Pattern: "*" + note.FileExt},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "已取消", nil
	}

	if err := os.WriteFile(filePath, []byte(note.Content), 0644); err != nil {
		a.LogSvc.Logger.Errorw("ExportNoteAsMarkdown 失败", fastlog.Error(err))
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	a.LogSvc.Logger.Infow("ExportNoteAsMarkdown 成功")
	return "导出成功：" + filePath, nil
}

// ExportAISessionAsMarkdown 导出 AI 对话为 Markdown 文件
func (a *App) ExportAISessionAsMarkdown(sessionID uint) (string, error) {
	a.LogSvc.Logger.Debugw("ExportAISessionAsMarkdown", fastlog.Uint("sessionID", sessionID))
	// 获取会话标题
	var session models.AISession
	if err := a.db.First(&session, sessionID).Error; err != nil {
		a.LogSvc.Logger.Errorw("ExportAISessionAsMarkdown 失败", fastlog.Error(err))
		return "", fmt.Errorf("对话不存在: %w", err)
	}

	// 获取会话消息
	messages := a.aiService.LoadAISessionMessages(sessionID)

	// 构建 Markdown 内容
	var buf strings.Builder
	buf.WriteString("# ")
	buf.WriteString(session.Title)
	buf.WriteString("\n\n---\n\n")

	for i, msg := range messages {
		if i > 0 {
			buf.WriteString("---\n\n")
		}

		switch msg.Role {
		case "user":
			buf.WriteString("**User**:\n")
			buf.WriteString(msg.Content)
			buf.WriteString("\n\n")
		case "assistant":
			buf.WriteString("**AI Assistant**:\n")
			if msg.ReasoningContent != "" {
				buf.WriteString("> 思考过程：\n")
				for _, line := range strings.Split(msg.ReasoningContent, "\n") {
					buf.WriteString("> ")
					buf.WriteString(line)
					buf.WriteString("\n")
				}
				buf.WriteString("\n")
			}
			buf.WriteString(msg.Content)
			buf.WriteString("\n\n")
		}
	}

	// 弹出保存对话框
	defaultName := sanitizeFilename(session.Title) + ".md"
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出对话",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown 文件 (*.md)", Pattern: "*.md"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "已取消", nil
	}

	// 写入文件
	if err := os.WriteFile(filePath, []byte(buf.String()), 0644); err != nil {
		a.LogSvc.Logger.Errorw("ExportAISessionAsMarkdown 失败", fastlog.Error(err))
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	a.LogSvc.Logger.Infow("ExportAISessionAsMarkdown 成功")
	return "导出成功：" + filePath, nil
}

// sanitizeFilename 清理笔记标题，生成安全的文件名
// 白名单策略：仅保留英文字母、数字、中文、中文标点及安全符号，其余字符（emoji、特殊符号等）全部移除
func sanitizeFilename(title string) string {
	// step1: 白名单过滤 — 只保留英文、中文、数字、安全符号
	var b strings.Builder
	b.Grow(len(title))
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r >= 0x4e00 && r <= 0x9fff, // CJK 统一表意文字
			r >= 0x3000 && r <= 0x303f, // CJK 符号和标点
			r == '-' || r == '_' || r == '.' || r == '(' || r == ')' ||
				r == '[' || r == ']' || r == '{' || r == '}' || r == ',' ||
				r == ';' || r == '!' || r == '?' || r == '+' || r == '=' ||
				r == '~' || r == '@' || r == '#' || r == '&' || r == ' ':
			b.WriteRune(r)
		}
	}
	name := b.String()

	// step2: 原有清洗流程
	name = strings.TrimSpace(name)
	if name == "" {
		return "untitled"
	}
	// 替换无效文件名字符和空白为下划线
	re := regexp.MustCompile(`[\\/:*?"<>|\s]+`)
	name = re.ReplaceAllString(name, "_")
	// 合并连续下划线
	re2 := regexp.MustCompile(`_+`)
	name = re2.ReplaceAllString(name, "_")
	// 去掉首尾下划线
	name = strings.Trim(name, "_")
	if name == "" {
		return "untitled"
	}
	return name
}

// openInFileManager 在系统文件管理器中打开指定目录（跨平台 Windows/macOS/Linux）
func openInFileManager(dir string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}

// OpenDataDir 在文件管理器中打开数据库目录
func (a *App) OpenDataDir() error {
	a.LogSvc.Logger.Debugw("OpenDataDir")
	dbPath, err := database.DefaultDBPath()
	if err != nil {
		a.LogSvc.Logger.Errorw("OpenDataDir 失败", fastlog.Error(err))
		return err
	}
	dir := filepath.Dir(dbPath)
	if err := openInFileManager(dir); err != nil {
		a.LogSvc.Logger.Errorw("OpenDataDir 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("OpenDataDir 成功")
	return nil
}

// OpenLogDir 在文件管理器中打开日志目录
func (a *App) OpenLogDir() error {
	a.LogSvc.Logger.Debugw("OpenLogDir")
	logDir := a.LogSvc.LogDir()
	if logDir == "" {
		dir, err := config.SubDir(config.DirLogs)
		if err != nil {
			a.LogSvc.Logger.Errorw("OpenLogDir 失败", fastlog.Error(err))
			return err
		}
		logDir = dir
	}
	if err := openInFileManager(logDir); err != nil {
		a.LogSvc.Logger.Errorw("OpenLogDir 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("OpenLogDir 成功")
	return nil
}

// OpenProjectURL 在默认浏览器中打开项目地址
func (a *App) OpenProjectURL(url string) string {
	a.LogSvc.Logger.Debugw("OpenProjectURL", fastlog.String("url", url))
	runtime.BrowserOpenURL(a.ctx, url)
	return "已打开浏览器"
}

// exportSnapshot 统一导出：VACUUM INTO → ZIP 打包 {jot-backup.db, images/} → 清理
func (a *App) exportSnapshot(destZipPath string) error {
	a.LogSvc.Logger.Debugw("exportSnapshot", fastlog.String("dest", destZipPath))
	// 1. VACUUM INTO 临时 .db
	tempDB := destZipPath + ".tmpdb"
	defer func() { _ = os.Remove(tempDB) }()
	if err := a.noteService.ExportBackup(tempDB); err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("VACUUM INTO 失败: %w", err)
	}

	// 2. 获取图片目录
	imgDir, err := a.imageDirPath()
	if err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return err
	}

	// 3. 创建 ZIP 文件
	zipFile, err := os.Create(destZipPath)
	if err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("创建 ZIP 失败: %w", err)
	}
	defer func() { _ = zipFile.Close() }()

	zw := zip.NewWriter(zipFile)
	defer func() { _ = zw.Close() }()

	// 3a. 添加 db 文件（不压缩，SQLite 已是压缩状态）
	dbFile, err := os.Open(tempDB)
	if err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("打开临时 db 失败: %w", err)
	}
	defer func() { _ = dbFile.Close() }()

	dbInfo, _ := dbFile.Stat()
	dbHeader, err := zip.FileInfoHeader(dbInfo)
	if err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("创建 db header 失败: %w", err)
	}
	dbHeader.Name = "jot-backup.db"
	dbHeader.Method = zip.Store
	dbWriter, err := zw.CreateHeader(dbHeader)
	if err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("创建 db ZIP entry 失败: %w", err)
	}
	if _, err := io.Copy(dbWriter, dbFile); err != nil {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("写入 db 到 ZIP 失败: %w", err)
	}

	// 3b. 添加 images/ 目录中的文件
	entries, err := os.ReadDir(imgDir)
	if err != nil && !os.IsNotExist(err) {
		a.LogSvc.Logger.Errorw("exportSnapshot 失败", fastlog.Error(err))
		return fmt.Errorf("读取图片目录失败: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		imgFile, err := os.Open(filepath.Join(imgDir, entry.Name()))
		if err != nil {
			continue
		}
		imgInfo, _ := imgFile.Stat()
		imgHeader, err := zip.FileInfoHeader(imgInfo)
		if err != nil {
			_ = imgFile.Close()
			continue
		}
		imgHeader.Name = "images/" + entry.Name()
		imgHeader.Method = zip.Deflate
		imgWriter, err := zw.CreateHeader(imgHeader)
		if err != nil {
			_ = imgFile.Close()
			continue
		}
		_, _ = io.Copy(imgWriter, imgFile)
		_ = imgFile.Close()
	}

	a.LogSvc.Logger.Infow("exportSnapshot 成功")
	return nil
}

// replaceDatabase 统一替换：备份当前 db → 关闭连接 → 替换 db + images → 重连 → 重建服务
func (a *App) replaceDatabase(srcDBPath, srcImagesDir string) error {
	a.LogSvc.Logger.Debugw("replaceDatabase", fastlog.String("srcDB", srcDBPath))
	dbPath, err := database.DefaultDBPath()
	if err != nil {
		a.LogSvc.Logger.Errorw("replaceDatabase 失败", fastlog.Error(err))
		return fmt.Errorf("获取数据库路径失败: %w", err)
	}

	// Step 1: 备份当前数据库
	backupPath := dbPath + ".bak"
	if err := fs.CopyEx(dbPath, backupPath, true); err != nil {
		a.LogSvc.Logger.Errorw("replaceDatabase 失败", fastlog.Error(err))
		return fmt.Errorf("备份当前数据库失败: %w", err)
	}

	// 失败的回滚
	rollback := func() {
		_ = fs.CopyEx(backupPath, dbPath, true)
		_ = a.reconnectDB(dbPath)
	}

	// Step 2: 关闭旧连接
	sqlDB, err := a.db.DB()
	if err != nil {
		_ = os.Remove(backupPath)
		a.LogSvc.Logger.Errorw("replaceDatabase 失败", fastlog.Error(err))
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}
	_ = sqlDB.Close()

	// 清理 WAL 模式残留文件，防止导入/还原时旧 WAL/SHM 文件干扰新数据库
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")

	// Step 3: 复制 db 文件
	if err := fs.CopyEx(srcDBPath, dbPath, true); err != nil {
		rollback()
		a.LogSvc.Logger.Errorw("replaceDatabase 失败", fastlog.Error(err))
		return fmt.Errorf("复制数据库文件失败: %w", err)
	}

	// Step 4: 替换 images/ 目录
	if srcImagesDir != "" {
		imgDir, err := a.imageDirPath()
		if err == nil {
			_ = os.RemoveAll(imgDir)
			_ = os.MkdirAll(imgDir, 0755)
			entries, _ := os.ReadDir(srcImagesDir)
			for _, entry := range entries {
				if !entry.IsDir() {
					src := filepath.Join(srcImagesDir, entry.Name())
					dst := filepath.Join(imgDir, entry.Name())
					_ = fs.CopyEx(src, dst, true)
				}
			}
		}
	}

	// Step 5: 重新初始化数据库
	newDB, err := database.InitDB(dbPath)
	if err != nil {
		rollback()
		a.LogSvc.Logger.Errorw("replaceDatabase 失败", fastlog.Error(err))
		return fmt.Errorf("数据库重连失败: %w", err)
	}

	// Step 6: 重建服务
	a.db = newDB
	a.rebuildServices(newDB)

	// Step 7: 清理备份
	_ = os.Remove(backupPath)

	a.LogSvc.Logger.Infow("replaceDatabase 成功")
	return nil
}

// importFromArchive 统一导入：解压 ZIP → 提取 db + images → replaceDatabase
func (a *App) importFromArchive(srcZipPath string) error {
	a.LogSvc.Logger.Debugw("importFromArchive", fastlog.String("src", srcZipPath))
	// 解压到临时目录
	tmpDir := filepath.Join(os.TempDir(), "jot-restore-"+fmt.Sprintf("%x", time.Now().UnixNano()))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		a.LogSvc.Logger.Errorw("importFromArchive 失败", fastlog.Error(err))
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	reader, err := zip.OpenReader(srcZipPath)
	if err != nil {
		a.LogSvc.Logger.Errorw("importFromArchive 失败", fastlog.Error(err))
		return fmt.Errorf("打开 ZIP 文件失败: %w", err)
	}
	defer func() { _ = reader.Close() }()

	var dbSrc string
	var imagesDir string

	for _, f := range reader.File {
		destPath := filepath.Join(tmpDir, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(destPath, 0755)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(destPath), 0755)

		rc, err := f.Open()
		if err != nil {
			continue
		}

		out, err := os.Create(destPath)
		if err != nil {
			_ = rc.Close()
			continue
		}

		_, _ = io.Copy(out, rc)
		_ = out.Close()
		_ = rc.Close()

		if f.Name == "jot-backup.db" {
			dbSrc = destPath
		} else if strings.HasPrefix(f.Name, "images/") {
			imagesDir = filepath.Dir(destPath)
		}
	}

	if dbSrc == "" {
		err := fmt.Errorf("ZIP 文件中未找到 jot-backup.db")
		a.LogSvc.Logger.Errorw("importFromArchive 失败", fastlog.Error(err))
		return err
	}

	a.LogSvc.Logger.Infow("importFromArchive 成功")
	return a.replaceDatabase(dbSrc, imagesDir)
}

// ==================== 一键备份与还原绑定方法 ====================

// BackupToDir 一键备份到 ~/.jot/backup/ 目录，固定文件名 jot-backup.zip（覆盖旧备份）
func (a *App) BackupToDir() (string, error) {
	a.LogSvc.Logger.Debugw("BackupToDir")
	backupDir, err := database.BackupDir()
	if err != nil {
		a.LogSvc.Logger.Errorw("BackupToDir 失败", fastlog.Error(err))
		return "", fmt.Errorf("获取备份目录失败: %w", err)
	}
	if err := database.EnsureBackupDir(); err != nil {
		a.LogSvc.Logger.Errorw("BackupToDir 失败", fastlog.Error(err))
		return "", fmt.Errorf("创建备份目录失败: %w", err)
	}

	zipPath := filepath.Join(backupDir, "jot-backup.zip")

	// 先删除旧备份
	_ = os.Remove(zipPath)

	if err := a.exportSnapshot(zipPath); err != nil {
		a.LogSvc.Logger.Errorw("BackupToDir 失败", fastlog.Error(err))
		return "", fmt.Errorf("备份失败: %w", err)
	}

	a.LogSvc.Logger.Infow("备份成功")
	return "备份成功：jot-backup.zip", nil
}

// RestoreFromDir 从 backup 目录的 jot-backup.zip 还原备份（含图片）
func (a *App) RestoreFromDir() (*services.ImportResult, error) {
	a.LogSvc.Logger.Debugw("RestoreFromDir")
	backupDir, err := database.BackupDir()
	if err != nil {
		a.LogSvc.Logger.Errorw("RestoreFromDir 失败", fastlog.Error(err))
		return &services.ImportResult{Message: "获取备份目录失败：" + err.Error()}, nil
	}

	zipPath := filepath.Join(backupDir, "jot-backup.zip")

	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		return &services.ImportResult{Message: "暂无可用备份"}, nil
	} else if err != nil {
		a.LogSvc.Logger.Errorw("RestoreFromDir 失败", fastlog.Error(err))
		return &services.ImportResult{Message: "读取备份文件失败：" + err.Error()}, nil
	}

	if err := a.importFromArchive(zipPath); err != nil {
		a.LogSvc.Logger.Errorw("RestoreFromDir 失败", fastlog.Error(err))
		return &services.ImportResult{Message: "还原失败：" + err.Error()}, nil
	}

	a.LogSvc.Logger.Infow("还原成功")
	return &services.ImportResult{
		Message:      "已从备份文件恢复：jot-backup.zip",
		SuccessCount: 1,
	}, nil
}

// GetBackupInfo 获取备份文件信息（文件名、修改时间、文件大小），无备份时返回空值
func (a *App) GetBackupInfo() (map[string]string, error) {
	a.LogSvc.Logger.Debugw("GetBackupInfo")
	backupDir, err := database.BackupDir()
	if err != nil {
		return map[string]string{"file_name": "", "file_time": "", "file_size": ""}, nil
	}

	filePath := filepath.Join(backupDir, "jot-backup.zip")
	fi, err := os.Stat(filePath)
	if err != nil {
		return map[string]string{"file_name": "", "file_time": "", "file_size": ""}, nil
	}

	size := fi.Size()
	var sizeStr string
	switch {
	case size < 1024:
		sizeStr = fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
	default:
		sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}

	return map[string]string{
		"file_name": "jot-backup.zip",
		"file_time": fi.ModTime().Format("2006-01-02 15:04"),
		"file_size": sizeStr,
	}, nil
}

// FileImportResult 单个文件导入结果
type FileImportResult struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	NoteID  uint   `json:"note_id"`
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// AIChatFileResult AI 聊天上传文件的处理结果
type AIChatFileResult struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Content   string `json:"content"`
	Size      int64  `json:"size"`
	Truncated bool   `json:"truncated"`
	Error     string `json:"error,omitempty"`
}

// ImportFiles 批量导入拖拽文件为笔记（归入指定笔记本），支持并发处理。
// 纯文本文件（.txt/.md/.json 等）直接读取（保留原后缀）；办公文件（.docx/.pdf/.xlsx 等）通过
// markitdown 转换为 Markdown 后以 .md 后缀创建笔记；不支持的格式返回错误。
// 导入过程中通过 Wails Events 发射进度事件（import:progress）。
func (a *App) ImportFiles(paths []string, notebookID uint) []FileImportResult {
	a.LogSvc.Logger.Debugw("ImportFiles", fastlog.Int("file_count", len(paths)), fastlog.Uint("notebookID", notebookID))
	if err := a.notebookService.EnsureDefaultNotebook(); err != nil {
		return []FileImportResult{{
			Path:    "",
			Success: false,
			Error:   "获取默认笔记本失败: " + err.Error(),
		}}
	}

	maxSize := a.GetMaxFileSize()
	results := make([]FileImportResult, len(paths))
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 发射导入开始事件
	runtime.EventsEmit(a.ctx, "import:progress", "start", len(paths))

	for i, p := range paths {
		wg.Add(1)
		go func(idx int, path string) {
			defer wg.Done()
			result := a.processImportFile(path, maxSize, notebookID)
			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, p)
	}

	wg.Wait()

	// 统计结果并发射完成事件
	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}
	runtime.EventsEmit(a.ctx, "import:progress", "complete", len(paths), successCount, failCount)

	return results
}

// processImportFile 处理单个文件的导入逻辑。
func (a *App) processImportFile(path string, maxSize int64, notebookID uint) FileImportResult {
	result := FileImportResult{Path: path}

	// 1. 检查路径
	info, err := os.Stat(path)
	if err != nil {
		a.LogSvc.Logger.Errorw("processImportFile: 无法访问文件", fastlog.String("path", path), fastlog.Error(err))
		result.Error = "无法访问文件: " + err.Error()
		return result
	}

	// 2. 拒绝目录
	if info.IsDir() {
		a.LogSvc.Logger.Debugw("processImportFile: 拒绝目录", fastlog.String("path", path))
		result.Error = "不支持导入目录，请选择文件"
		return result
	}

	// 3. 文件大小限制（放在最前面）
	if info.Size() > maxSize {
		maxSizeMB := maxSize / (1024 * 1024)
		a.LogSvc.Logger.Debugw("processImportFile: 文件过大", fastlog.String("path", path), fastlog.Int64("size", info.Size()), fastlog.Int64("maxSize", maxSize))
		result.Error = fmt.Sprintf("文件过大（超过 %dMB），无法导入", maxSizeMB)
		return result
	}

	// 提取文件名（去后缀）作标题
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	title := strings.TrimSuffix(name, ext)
	if title == "" {
		title = "untitled"
	}

	// 确定文件后缀
	fileExt := ext
	if fileExt == "" {
		fileExt = ".txt"
	}

	// 4. 文件类型判定与内容读取
	var content string

	if converter.IsOfficeFile(path) {
		// 办公文件 → markitdown 转换；内容已是标准 Markdown，后缀统一改为 .md，
		// 使前端 Markdown 能力（预览/语法高亮/TOC 等）对该笔记全部生效。
		fileExt = ".md"
		a.LogSvc.Logger.Debugw("processImportFile: 转换办公文件", fastlog.String("path", path), fastlog.Int64("size", info.Size()))
		mdText, err := converter.ConvertToMarkdown(path)
		if err != nil {
			switch {
			case errors.Is(err, converter.ErrUnsupportedFormat):
				a.LogSvc.Logger.Debugw("processImportFile: 不支持的办公文件格式", fastlog.String("path", path))
				result.Error = "不支持的文件格式"
			case errors.Is(err, converter.ErrConversionTimeout):
				a.LogSvc.Logger.Warnw("processImportFile: 办公文件转换超时", fastlog.String("path", path), fastlog.Int64("size", info.Size()))
				result.Error = "文件转换超时（超过60秒）"
			default:
				a.LogSvc.Logger.Errorw("processImportFile: 办公文件转换失败", fastlog.String("path", path), fastlog.Error(err))
				result.Error = fmt.Sprintf("文件转换失败: %s", err.Error())
			}
			return result
		}
		content = mdText
		a.LogSvc.Logger.Infow("processImportFile: 办公文件转换成功", fastlog.String("path", path), fastlog.Int("content_len", len(mdText)))
	} else if fs.IsBinaryPath(path) {
		// 二进制文件（非办公文件）→ 拒绝
		a.LogSvc.Logger.Debugw("processImportFile: 拒绝二进制文件", fastlog.String("path", path))
		result.Error = "不支持导入此类文件，请选择文本文件或办公文档后重试"
		return result
	} else {
		// 纯文本文件 → 直接读取
		a.LogSvc.Logger.Debugw("processImportFile: 直接读取文本文件", fastlog.String("path", path), fastlog.Int64("size", info.Size()))
		data, err := os.ReadFile(path)
		if err != nil {
			a.LogSvc.Logger.Errorw("processImportFile: 读取文本文件失败", fastlog.String("path", path), fastlog.Error(err))
			result.Error = "读取文件失败: " + err.Error()
			return result
		}
		content = string(data)
	}

	// 5. 创建笔记（归入指定笔记本）
	note, err := a.noteService.CreateWithNotebook(title, content, fileExt, notebookID)
	if err != nil {
		a.LogSvc.Logger.Errorw("processImportFile: 创建笔记失败", fastlog.String("path", path), fastlog.String("title", title), fastlog.Error(err))
		result.Error = "创建笔记失败: " + err.Error()
		return result
	}

	a.LogSvc.Logger.Infow("processImportFile: 导入成功", fastlog.String("path", path), fastlog.String("title", title), fastlog.Uint("noteID", note.ID))

	result.Title = title
	result.NoteID = note.ID
	result.Success = true
	return result
}

// ResetDatabase 清空所有数据，恢复出厂状态（删表重建）
func (a *App) ResetDatabase() error {
	a.LogSvc.Logger.Debugw("ResetDatabase")
	// 1. 删除所有表（子表在前顺序由 database.AllModels 保证，自动处理外键依赖）
	for _, table := range database.AllModels {
		if err := a.db.Migrator().DropTable(table); err != nil {
			return err
		}
	}

	// 显式删除多对多关联表（没有对应的 model struct）
	if err := a.db.Exec("DROP TABLE IF EXISTS note_tags").Error; err != nil {
		return err
	}

	// 2. 重新 AutoMigrate（与 InitDB 保持同步，模型注册见 database.AllModels）
	if err := a.db.AutoMigrate(database.AllModels...); err != nil {
		return err
	}

	// 3. 重新初始化内置技能提示词
	if err := database.InitBuiltinPrompts(a.db); err != nil {
		return fmt.Errorf("初始化内置提示词失败: %w", err)
	}

	// 4. 重新初始化默认标签
	if err := services.InitDefaultTags(a.db); err != nil {
		return err
	}

	// 4. 重新初始化默认设置
	if err := database.InitDefaultSettings(a.db); err != nil {
		return err
	}

	// 5. 确保默认笔记本存在
	if err := a.notebookService.EnsureDefaultNotebook(); err != nil {
		return err
	}

	// 6. 重建数据库连接（DropTable 后 glebarez/sqlite 驱动连接可能失效）
	dbPath, err := database.DefaultDBPath()
	if err != nil {
		return fmt.Errorf("获取数据库路径失败: %w", err)
	}
	if err := a.reconnectDB(dbPath); err != nil {
		return fmt.Errorf("重置后重连失败: %w", err)
	}

	// 7. 清空图片目录
	imgDir, err := a.imageDirPath()
	if err == nil {
		_ = os.RemoveAll(imgDir)
		_ = os.MkdirAll(imgDir, 0755)
	}

	a.LogSvc.Logger.Infow("数据库重置完成")
	return nil
}

// rebuildServices 使用新的数据库连接重建所有服务实例
func (a *App) rebuildServices(db *gorm.DB) {
	a.LogSvc.Logger.Debugw("rebuildServices")
	a.settingService = services.NewSettingService(db)
	a.noteService = services.NewNoteService(db, a.settingService, a.LogSvc.Logger)
	a.tagService = services.NewTagService(db, a.LogSvc.Logger)
	a.notebookService = services.NewNotebookService(db, a.LogSvc.Logger)
	a.aiService = services.NewAIService(db, a.LogSvc.Logger)
	a.profileService = services.NewProfileService(db, a.LogSvc.Logger)
	a.vectorService = services.NewVectorService(db, a.LogSvc.Logger)
	a.todoService = services.NewTodoService(db, a.LogSvc.Logger)
	a.mcpServerService = services.NewMCPServerService(db)
	a.statsService = services.NewStatsService(a.noteService, a.tagService, a.todoService, a.aiService, database.DefaultDBPath)
	// 重建日志服务
	a.LogSvc = services.NewLogService()
	logDir, err := config.SubDir(config.DirLogs)
	if err != nil {
		a.LogSvc.Logger.Errorw("获取日志目录失败", fastlog.Error(err))
	}
	logLevelStr := a.settingService.Get("log_level")
	logLevelVal := 1
	if n, err := strconv.Atoi(logLevelStr); err == nil {
		logLevelVal = n
	}
	logLevel := services.LevelFromInt(logLevelVal)
	if err := a.LogSvc.Init(logDir, logLevel); err != nil {
		a.LogSvc.Logger.Errorw("日志重新初始化失败", fastlog.Error(err))
	}
	a.LogSvc.Logger.Infow("rebuildServices 成功")

	// 重建 AgentSvc：rebuildServices 重建了各服务（新 gorm.DB 连接），
	// 必须同步用最新实例重新装配 AgentSvc，否则 Agent 工具（manage_todo / manage_notebook /
	// manage_tag 等）仍持有旧服务指针，数据库重置或切换后操作的是旧连接。
	// 先释放旧 AgentSvc 的全部会话实例（取消等待中的反问 run），避免重建后泄漏
	if a.AgentSvc != nil {
		a.AgentSvc.ReleaseAll()
	}
	// MCP 连接池：旧池连接指向旧库配置（可能含已删除服务器的连接），重建 AgentSvc 前
	// 关闭旧池全部连接；新池注入新 AgentSvc（预热在下次进入 AI 助手/设置页操作时触发）
	if a.mcpPool != nil {
		a.mcpPool.CloseAll()
	}
	a.mcpPool = mcpserver.NewPool()
	a.mcpPool.SetLogger(a.LogSvc.Logger)
	a.AgentSvc = agent.NewAgentService(agent.Deps{
		AI:             a.aiService,
		Vector:         a.vectorService,
		Setting:        a.settingService,
		Todo:           a.todoService,
		Notebook:       a.notebookService,
		Tag:            a.tagService,
		Note:           a.noteService,
		Stats:          a.statsService,
		Logger:         a.LogSvc.Logger,
		MCPServerDB:    db,
		MCPPool:        a.mcpPool,
		GetEmbedConfig: a.GetEmbedConfig,
	})
}

// ==================== Todo 相关绑定方法 ====================

func (a *App) CreateTodo(text string) (*models.Todo, error) {
	a.LogSvc.Logger.Debugw("CreateTodo")
	todo, err := a.todoService.Create(text)
	if err != nil {
		a.LogSvc.Logger.Errorw("CreateTodo 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("CreateTodo 成功", fastlog.Uint("id", todo.ID))
	return todo, nil
}

func (a *App) ListTodos() ([]models.Todo, error) {
	a.LogSvc.Logger.Debugw("ListTodos")
	todos, err := a.todoService.List()
	if err != nil {
		a.LogSvc.Logger.Errorw("ListTodos 失败", fastlog.Error(err))
		return nil, err
	}
	return todos, nil
}

func (a *App) ToggleTodo(id uint) (*models.Todo, error) {
	a.LogSvc.Logger.Debugw("ToggleTodo", fastlog.Uint("id", id))
	todo, err := a.todoService.Toggle(id)
	if err != nil {
		a.LogSvc.Logger.Errorw("ToggleTodo 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("ToggleTodo 成功", fastlog.Uint("id", id))
	return todo, nil
}

func (a *App) DeleteTodo(id uint) error {
	a.LogSvc.Logger.Debugw("DeleteTodo", fastlog.Uint("id", id))
	if err := a.todoService.Delete(id); err != nil {
		a.LogSvc.Logger.Errorw("DeleteTodo 失败", fastlog.Error(err))
		return err
	}
	a.LogSvc.Logger.Infow("DeleteTodo 成功", fastlog.Uint("id", id))
	return nil
}

func (a *App) UpdateTodo(id uint, text string) (*models.Todo, error) {
	a.LogSvc.Logger.Debugw("UpdateTodo", fastlog.Uint("id", id))
	todo, err := a.todoService.Update(id, text)
	if err != nil {
		a.LogSvc.Logger.Errorw("UpdateTodo 失败", fastlog.Error(err))
		return nil, err
	}
	a.LogSvc.Logger.Infow("UpdateTodo 成功", fastlog.Uint("id", id))
	return todo, nil
}

func (a *App) ClearCompletedTodos() (string, error) {
	a.LogSvc.Logger.Debugw("ClearCompletedTodos")
	count, err := a.todoService.DeleteCompleted()
	if err != nil {
		a.LogSvc.Logger.Errorw("ClearCompletedTodos 失败", fastlog.Error(err))
		return "", err
	}
	a.LogSvc.Logger.Infow("ClearCompletedTodos 成功", fastlog.Int64("count", count))
	return fmt.Sprintf("已清空 %d 个已完成待办事项", count), nil
}

// CountUnfinishedTodos 返回未完成待办数量（启动弹窗提示用）
func (a *App) CountUnfinishedTodos() (int64, error) {
	a.LogSvc.Logger.Debugw("CountUnfinishedTodos")
	return a.todoService.CountUnfinished()
}

// reconnectDB 重新连接数据库（用于导入失败后的恢复）
func (a *App) reconnectDB(dbPath string) error {
	a.LogSvc.Logger.Debugw("reconnectDB", fastlog.String("dbPath", dbPath))
	// 关闭旧连接
	if sqlDB, err := a.db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	db, err := database.InitDB(dbPath)
	if err != nil {
		a.LogSvc.Logger.Errorw("reconnectDB 失败", fastlog.Error(err))
		return fmt.Errorf("数据库重连失败: %w", err)
	}
	a.db = db
	a.rebuildServices(db)
	a.LogSvc.Logger.Infow("reconnectDB 成功")
	return nil
}
