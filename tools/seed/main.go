package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"jot/database"
	"jot/models"

	"gorm.io/gorm"
)

func main() {
	// 解析命令行参数：数据库文件路径
	if len(os.Args) < 2 {
		fmt.Println("用法: go run cmd/seed/main.go <db_path>")
		fmt.Println("示例: go run cmd/seed/main.go data/jot.db")
		os.Exit(1)
	}
	dbPath := os.Args[1]

	// 初始化数据库连接
	fmt.Printf("连接数据库: %s\n", dbPath)
	db := database.InitDB(dbPath)

	// 注入标签
	fmt.Println("正在注入标签...")
	tags := seedTags(db)

	// 注入笔记
	fmt.Println("正在注入笔记...")
	notes := seedNotes(db, tags)

	// 建立笔记-标签关联
	fmt.Println("正在关联标签...")
	seedNoteTags(db, notes, tags)

	fmt.Println("\n✅ 测试数据注入完成!")
	fmt.Printf("   标签: %d 个\n", len(tags))
	fmt.Printf("   笔记: %d 条\n", len(notes))
}

// seedTags 创建测试标签
func seedTags(db *gorm.DB) []models.Tag {
	tagData := []struct {
		Name  string
		Color string
	}{
		{Name: "工作", Color: "#ef4444"},
		{Name: "学习", Color: "#3b82f6"},
		{Name: "生活", Color: "#10b981"},
		{Name: "灵感", Color: "#8b5cf6"},
		{Name: "待办", Color: "#f59e0b"},
	}

	var tags []models.Tag
	for _, t := range tagData {
		// 跳过已存在的同名标签
		var existing models.Tag
		if db.Where("name = ?", t.Name).First(&existing).Error == nil {
			tags = append(tags, existing)
			continue
		}

		tag := models.Tag{Name: t.Name, Color: t.Color}
		if err := db.Create(&tag).Error; err != nil {
			fmt.Printf("   ⚠️  创建标签失败 %s: %v\n", t.Name, err)
			continue
		}
		tags = append(tags, tag)
	}
	return tags
}

// seedNotes 创建测试笔记
func seedNotes(db *gorm.DB, tags []models.Tag) []models.Note {
	now := time.Now()

	noteData := []struct {
		Title   string
		Content string
		Pinned  bool
		DaysAgo int // 创建时间偏移（天）
	}{
		{
			Title:   "📌 项目计划 - Q3 里程碑",
			Content: "## Q3 关键目标\n\n1. **完成核心模块重构**\n   - 数据库层迁移到 GORM\n   - 新增 RESTful API 接口\n2. **性能优化**\n   - 页面加载时间 < 1s\n   - API 响应时间 < 200ms\n3. **测试覆盖**\n   - 单元测试覆盖率 > 80%\n   - E2E 测试覆盖核心流程",
			Pinned:  true,
			DaysAgo: 0,
		},
		{
			Title:   "🛒 购物清单",
			Content: "周末去超市采购：\n\n- [ ] 牛奶 2 瓶\n- [ ] 面包 1 袋\n- [ ] 鸡蛋 1 板\n- [ ] 水果（苹果、香蕉）\n- [ ] 蔬菜（西兰花、番茄）\n- [ ] 纸巾 2 包\n- [ ] 垃圾袋 1 卷",
			Pinned:  false,
			DaysAgo: 1,
		},
		{
			Title:   "💡 产品创意",
			Content: "### 想法记录\n\n**AI 笔记助手**\n- 自动标签推荐\n- 内容摘要生成\n- 关联笔记推荐\n\n**跨平台同步**\n- 手机端阅读\n- 桌面端编辑\n- 云端实时同步",
			Pinned:  false,
			DaysAgo: 2,
		},
		{
			Title:   "📚 Go 语言学习笔记",
			Content: "## Go 并发编程要点\n\n### Goroutine\n- 轻量级线程，由 Go 运行时管理\n- 使用 `go` 关键字启动\n- 栈空间初始仅 2KB，可动态增长\n\n### Channel\n```go\nch := make(chan int, 10)\ngo func() {\n    ch <- 42\n}()\nvalue := <-ch\n```\n\n### Select\n- 多路复用 channel\n- 配合 for 循环实现超时控制",
			Pinned:  false,
			DaysAgo: 3,
		},
		{
			Title:   "🏠 装修注意事项",
			Content: "### 水电改造\n- [x] 插座位置确认\n- [ ] 水管打压测试\n- [ ] 弱电布线方案\n\n### 瓦木工程\n- [ ] 瓷砖进场验收\n- [ ] 木工交底\n- [ ] 地面找平检查\n\n### 油漆阶段\n- [ ] 墙面基层处理\n- [ ] 底漆 1 遍 + 面漆 2 遍",
			Pinned:  false,
			DaysAgo: 5,
		},
		{
			Title:   "🎬 近期观影清单",
			Content: "### 想看\n- 《奥本海默》- 传记/历史\n- 《蜘蛛侠：纵横宇宙》- 动画\n- 《流浪地球 3》- 科幻\n\n### 已看\n- [x] 《肖申克的救赎》⭐⭐⭐⭐⭐\n- [x] 《盗梦空间》⭐⭐⭐⭐⭐\n- [x] 《星际穿越》⭐⭐⭐⭐",
			Pinned:  false,
			DaysAgo: 7,
		},
		{
			Title:   "⚙️ 开发环境配置",
			Content: "### Windows 开发环境\n\n**Go 工具链**\n1. 安装 Go 1.26\n2. 配置 GOPATH\n3. 安装 golangci-lint\n\n**Wails**\n1. `wails doctor` 检查环境\n2. `wails dev` 启动开发模式\n\n**数据库**\n- SQLite (纯 Go 驱动，免 CGO)\n- 数据库路径: `data/jot.db`",
			Pinned:  false,
			DaysAgo: 10,
		},
		{
			Title:   "📝 周报模板",
			Content: "## 本周工作\n\n### 完成事项\n1. \n2. \n3. \n\n### 进行中\n1. \n2. \n\n### 遇到的问题\n1. \n2. \n\n### 下周计划\n1. \n2.",
			Pinned:  false,
			DaysAgo: 14,
		},
	}

	var notes []models.Note
	for _, nd := range noteData {
		createdAt := now.AddDate(0, 0, -nd.DaysAgo).Add(-time.Duration(rand.Intn(86400)) * time.Second)
		updatedAt := createdAt.Add(time.Duration(rand.Intn(7200)) * time.Second)

		note := models.Note{
			Title:     nd.Title,
			Content:   nd.Content,
			Pinned:    nd.Pinned,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
		if err := db.Create(&note).Error; err != nil {
			fmt.Printf("   ⚠️  创建笔记失败 %q: %v\n", nd.Title, err)
			continue
		}
		notes = append(notes, note)
	}
	return notes
}

// seedNoteTags 为笔记关联标签
func seedNoteTags(db *gorm.DB, notes []models.Note, tags []models.Tag) {
	for i, note := range notes {
		// 每条笔记关联 1~3 个随机标签
		numTags := 1 + rand.Intn(3)
		if numTags > len(tags) {
			numTags = len(tags)
		}

		// 随机打乱标签顺序取前 numTags 个
		perm := rand.Perm(len(tags))
		for j := 0; j < numTags; j++ {
			tag := tags[perm[j]]
			if err := db.Model(&note).Association("Tags").Append(&tag); err != nil {
				fmt.Printf("   ⚠️  关联标签失败 (笔记#%d, 标签#%d): %v\n", note.ID, tag.ID, err)
			}
		}

		// 前 3 条笔记额外关联"工作"标签
		if i < 3 {
			for _, tag := range tags {
				if tag.Name == "工作" {
					_ = db.Model(&note).Association("Tags").Append(&tag)
				}
			}
		}
	}
}
