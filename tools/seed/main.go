package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"jot/internal/database"
	"jot/internal/models"

	"gorm.io/gorm"
)

func main() {
	// 默认数据库路径
	dbPath := ""
	if len(os.Args) >= 2 {
		dbPath = os.Args[1]
	} else {
		var err error
		dbPath, err = database.DefaultDBPath()
		if err != nil {
			panic(err)
		}
	}

	// 初始化数据库连接
	fmt.Printf("连接数据库: %s\n", dbPath)
	db, err := database.InitDB(dbPath)
	if err != nil {
		panic(err)
	}

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
		Title    string
		Content  string
		Pinned   bool
		DaysAgo  int // 创建时间偏移（天）
		NoteType string
	}{
		{
			Title:    "📌 项目计划 - Q3 里程碑",
			Content:  "## Q3 关键目标\n\n1. **完成核心模块重构**\n   - 数据库层迁移到 GORM\n   - 新增 RESTful API 接口\n2. **性能优化**\n   - 页面加载时间 < 1s\n   - API 响应时间 < 200ms\n3. **测试覆盖**\n   - 单元测试覆盖率 > 80%\n   - E2E 测试覆盖核心流程",
			Pinned:   true,
			DaysAgo:  0,
			NoteType: "markdown",
		},
		{
			Title:    "🛒 购物清单",
			Content:  "周末去超市采购：\n\n- [ ] 牛奶 2 瓶\n- [ ] 面包 1 袋\n- [ ] 鸡蛋 1 板\n- [ ] 水果（苹果、香蕉）\n- [ ] 蔬菜（西兰花、番茄）\n- [ ] 纸巾 2 包\n- [ ] 垃圾袋 1 卷",
			Pinned:   false,
			DaysAgo:  1,
			NoteType: "text",
		},
		{
			Title:    "💡 产品创意 - AI 笔记助手",
			Content:  "### 想法记录\n\n**AI 笔记助手**\n- 自动标签推荐\n- 内容摘要生成\n- 关联笔记推荐\n\n**跨平台同步**\n- 手机端阅读\n- 桌面端编辑\n- 云端实时同步",
			Pinned:   false,
			DaysAgo:  2,
			NoteType: "markdown",
		},
		{
			Title:    "📚 Go 并发编程要点",
			Content:  "## Go 并发编程要点\n\n### Goroutine\n- 轻量级线程，由 Go 运行时管理\n- 使用 `go` 关键字启动\n- 栈空间初始仅 2KB，可动态增长\n\n### Channel\n```go\nch := make(chan int, 10)\ngo func() {\n    ch <- 42\n}()\nvalue := <-ch\n```\n\n### Select\n- 多路复用 channel\n- 配合 for 循环实现超时控制",
			Pinned:   false,
			DaysAgo:  3,
			NoteType: "markdown",
		},
		{
			Title:    "🏠 装修注意事项",
			Content:  "### 水电改造\n- [x] 插座位置确认\n- [ ] 水管打压测试\n- [ ] 弱电布线方案\n\n### 瓦木工程\n- [ ] 瓷砖进场验收\n- [ ] 木工交底\n- [ ] 地面找平检查\n\n### 油漆阶段\n- [ ] 墙面基层处理\n- [ ] 底漆 1 遍 + 面漆 2 遍",
			Pinned:   false,
			DaysAgo:  5,
			NoteType: "text",
		},
		{
			Title:    "🎬 近期观影清单",
			Content:  "### 想看\n- 《奥本海默》- 传记/历史\n- 《蜘蛛侠：纵横宇宙》- 动画\n- 《流浪地球 3》- 科幻\n\n### 已看\n- [x] 《肖申克的救赎》⭐⭐⭐⭐⭐\n- [x] 《盗梦空间》⭐⭐⭐⭐⭐\n- [x] 《星际穿越》⭐⭐⭐⭐",
			Pinned:   false,
			DaysAgo:  7,
			NoteType: "text",
		},
		{
			Title:    "⚙️ 开发环境配置指南",
			Content:  "### Windows 开发环境\n\n**Go 工具链**\n1. 安装 Go 1.26\n2. 配置 GOPATH\n3. 安装 golangci-lint\n\n**Wails**\n1. `wails doctor` 检查环境\n2. `wails dev` 启动开发模式\n\n**数据库**\n- SQLite (纯 Go 驱动，免 CGO)\n- 数据库路径: `~/.jot/data/jot.db`",
			Pinned:   false,
			DaysAgo:  10,
			NoteType: "markdown",
		},
		{
			Title:    "📝 周报模板",
			Content:  "## 本周工作\n\n### 完成事项\n1. \n2. \n3. \n\n### 进行中\n1. \n2. \n\n### 遇到的问题\n1. \n2. \n\n### 下周计划\n1. \n2.",
			Pinned:   false,
			DaysAgo:  14,
			NoteType: "markdown",
		},
		{
			Title:    "🍳 家常菜谱汇总",
			Content:  "### 红烧肉\n- 五花肉 500g，冰糖 30g\n- 八角 2 个，桂皮 1 段\n- 生抽 3 勺，老抽 1 勺\n\n### 番茄炒蛋\n- 番茄 2 个，鸡蛋 3 个\n- 盐、糖适量\n\n### 清炒时蔬\n- 西兰花、蒜末\n- 蚝油 1 勺",
			Pinned:   false,
			DaysAgo:  4,
			NoteType: "text",
		},
		{
			Title:    "💻 常用 Git 命令速查",
			Content:  "### 日常操作\n```\ngit add -p          # 交互式暂存\ngit commit -v       # 查看 diff 后提交\ngit log --oneline --graph  # 历史树\n```\n### 分支操作\n```\ngit switch -c <branch>     # 新建并切换\ngit merge --no-ff <branch> # 合并不快进\ngit rebase -i HEAD~3       # 交互式变基\n```\n### 回退\n```\ngit reset HEAD~1    # 撤消提交保留修改\ngit restore <file>  # 丢弃工作区修改\n```",
			Pinned:   false,
			DaysAgo:  6,
			NoteType: "markdown",
		},
		{
			Title:    "🧘 每日冥想记录",
			Content:  "### 第 1 周\n- [x] 周一 10min - 专注力提升\n- [x] 周三 15min - 呼吸练习\n- [ ] 周五 15min\n\n### 感受\n冥想后头脑更清晰，注意力更集中。\n\n### Tips\n- 早晨起床后最佳\n- 保持背部挺直\n- 关注呼吸节奏",
			Pinned:   false,
			DaysAgo:  8,
			NoteType: "text",
		},
		{
			Title:    "📖 读书笔记 - 《设计模式》",
			Content:  "## 创建型模式\n\n### 单例模式\n保证一个类仅有一个实例，并提供一个全局访问点。\n\n### 工厂模式\n定义一个创建对象的接口，让子类决定实例化哪个类。\n\n## 结构型模式\n\n### 适配器模式\n将一个类的接口转换成客户期望的另一个接口。\n\n### 观察者模式\n定义对象间一对多的依赖关系，当一个对象状态变化时，所有依赖对象自动收到通知。",
			Pinned:   false,
			DaysAgo:  9,
			NoteType: "markdown",
		},
		{
			Title:    "🏋️ 健身计划",
			Content:  "### 周一：胸+三头\n- 杠铃卧推 4×10\n- 哑铃飞鸟 3×12\n- 绳索下压 3×12\n\n### 周三：背+二头\n- 引体向上 4×8\n- 杠铃划船 4×10\n- 哑铃弯举 3×12\n\n### 周五：腿+肩\n- 深蹲 5×5\n- 推举 4×10\n- 侧平举 3×15",
			Pinned:   false,
			DaysAgo:  11,
			NoteType: "text",
		},
		{
			Title:    "🌐 个人网站优化清单",
			Content:  "- [ ] 启用 Brotli 压缩\n- [x] 添加 CDN 缓存\n- [ ] 图片转 WebP 格式\n- [ ] 减少 CSS/JS 打包体积\n- [ ] 添加骨架屏加载\n- [ ] 实现懒加载\n- [ ] 优化 LCP 指标",
			Pinned:   false,
			DaysAgo:  12,
			NoteType: "text",
		},
		{
			Title:    "🎯 2026 年度目标",
			Content:  "## 大目标\n1. **完成开源项目 Jot** — 发布 v1.0\n2. **学习 Rust 语言** — 完成一个小项目\n3. **阅读 24 本书** — 每月 2 本\n4. **健身达标** — 体脂率降到 18%\n\n## 季度拆解\n| 季度 | 目标 |\n|------|------|\n| Q1 | 基础搭建 |\n| Q2 | 功能完善 |\n| Q3 | 性能优化 |\n| Q4 | 发布迭代 |",
			Pinned:   true,
			DaysAgo:  13,
			NoteType: "markdown",
		},
		{
			Title:    "🐳 Docker 常用命令",
			Content:  "### 容器管理\n```\ndocker ps -a                # 查看所有容器\ndocker start/stop <name>    # 启停容器\ndocker logs -f <name>       # 查看日志\n```\n### 镜像操作\n```\ndocker images               # 查看镜像列表\ndocker pull <image>:<tag>   # 拉取镜像\ndocker rmi <image>          # 删除镜像\n```\n### Compose\n```\ndocker compose up -d        # 启动服务\ndocker compose down         # 停止服务\n```",
			Pinned:   false,
			DaysAgo:  15,
			NoteType: "markdown",
		},
		{
			Title:    "🎵 近期歌单推荐",
			Content:  "### 循环中\n- 《起风了》- 买辣椒也用券\n- 《路过人间》- 郁可唯\n- 《山丘》- 李宗盛\n- 《平凡之路》- 朴树\n\n### 经典\n- 《Hotel California》- Eagles\n- 《Bohemian Rhapsody》- Queen\n- 《Imagine》- John Lennon",
			Pinned:   false,
			DaysAgo:  16,
			NoteType: "text",
		},
		{
			Title:    "📊 数据分析笔记",
			Content:  "## Pandas 常用操作\n\n### 数据读取\n```python\ndf = pd.read_csv('data.csv')\ndf.info()  # 查看基本信息\n```\n\n### 数据清洗\n- `dropna()` — 删除缺失值\n- `fillna(value)` — 填充缺失值\n- `drop_duplicates()` — 去重\n\n### 可视化\n```python\nimport matplotlib.pyplot as plt\ndf.plot(kind='bar')\nplt.show()\n```",
			Pinned:   false,
			DaysAgo:  18,
			NoteType: "markdown",
		},
		{
			Title:    "✈️ 旅行计划 - 云南",
			Content:  "## 行程安排\n\n### Day 1-2: 大理\n- 洱海骑行\n- 古城漫步\n- 苍山索道\n\n### Day 3-4: 丽江\n- 玉龙雪山\n- 束河古镇\n- 泸沽湖\n\n### Day 5: 昆明\n- 滇池\n- 斗南花市\n\n**预算**: 5000 元",
			Pinned:   false,
			DaysAgo:  20,
			NoteType: "text",
		},
		{
			Title:    "🔐 密码安全备忘",
			Content:  "### 好习惯\n- 每个网站使用不同密码\n- 密码长度至少 12 位\n- 包含大小写字母+数字+符号\n- 启用两步验证（2FA）\n\n### 推荐工具\n- **Bitwarden** — 开源密码管理器\n- **Authy** — 两步验证码 app\n\n### 绝不\n- ❌ 不要重复使用密码\n- ❌ 不要用生日/姓名\n- ❌ 不要把密码存明文",
			Pinned:   false,
			DaysAgo:  22,
			NoteType: "markdown",
		},
		{
			Title:    "📱 应用推广思路",
			Content:  "## 推广渠道\n\n### 线上\n- **V2EX** — 发帖分享开发经历\n- **GitHub** — 开源吸引 Star\n- **即刻** — 产品向社区\n- **少数派** — 投稿评测\n\n### 线下\n- 参加技术 meetup\n- 程序员朋友推荐\n\n### 增长策略\n1. 免费增值模式\n2. 邀请奖励机制\n3. 优质内容运营",
			Pinned:   false,
			DaysAgo:  25,
			NoteType: "text",
		},
		{
			Title:    "🎄 圣诞节礼物清单",
			Content:  "- [ ] 给妈妈 — 羊绒围巾\n- [ ] 给爸爸 — 茶叶礼盒\n- [ ] 给对象 — 她最近想要的那本书\n- [ ] 给朋友 — 手工烘焙饼干\n- [ ] 给自己 — 一个新键盘",
			Pinned:   false,
			DaysAgo:  28,
			NoteType: "text",
		},
		{
			Title:    "🚀 效率工具推荐",
			Content:  "## 日常使用\n| 工具 | 用途 |\n|------|------|\n| VS Code | 代码编辑器 |\n| Obsidian | 知识管理 |\n| Snipaste | 截图工具 |\n| Ditto | 剪贴板增强 |\n| Everything | 文件搜索 |\n\n## 终端\n- **Windows Terminal** — 现代终端\n- **Oh My Posh** — 提示符美化\n- **fzf** — 模糊搜索",
			Pinned:   false,
			DaysAgo:  30,
			NoteType: "markdown",
		},
	}

	var notes []models.Note
	for _, nd := range noteData {
		createdAt := now.AddDate(0, 0, -nd.DaysAgo).Add(-time.Duration(rand.Intn(86400)) * time.Second)
		updatedAt := createdAt.Add(time.Duration(rand.Intn(7200)) * time.Second)

		note := models.Note{
			Title:     nd.Title,
			Content:   nd.Content,
			NoteType:  nd.NoteType,
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
