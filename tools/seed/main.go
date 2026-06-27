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

	// 注入笔记本
	fmt.Println("正在注入笔记本...")
	notebooks := seedNotebooks(db)

	// 注入笔记
	fmt.Println("正在注入笔记...")
	notes := seedNotes(db, tags, notebooks)

	// 建立笔记-标签关联
	fmt.Println("正在关联标签...")
	seedNoteTags(db, notes, tags)

	fmt.Println("\n✅ 测试数据注入完成!")
	fmt.Printf("   笔记本: %d 个\n", len(notebooks))
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
		{Name: "技术", Color: "#06b6d4"},
		{Name: "随笔", Color: "#ec4899"},
	}

	var tags []models.Tag
	for _, t := range tagData {
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

// seedNotebooks 创建测试笔记本
func seedNotebooks(db *gorm.DB) []models.Notebook {
	notebookData := []struct {
		Name      string
		SortOrder int
	}{
		{Name: "工作", SortOrder: 1},
		{Name: "学习", SortOrder: 2},
		{Name: "生活", SortOrder: 3},
		{Name: "个人项目", SortOrder: 4},
		{Name: "随笔", SortOrder: 5},
	}

	// 先确保默认笔记本存在（id=1）
	var defaultNb models.Notebook
	result := db.First(&defaultNb, 1)
	if result.Error != nil {
		defaultNb = models.Notebook{Name: "默认笔记本", SortOrder: 0}
		db.Create(&defaultNb)
	}

	var notebooks []models.Notebook
	notebooks = append(notebooks, defaultNb)

	for _, nd := range notebookData {
		var existing models.Notebook
		if db.Where("name = ?", nd.Name).First(&existing).Error == nil {
			notebooks = append(notebooks, existing)
			continue
		}

		nb := models.Notebook{Name: nd.Name, SortOrder: nd.SortOrder}
		if err := db.Create(&nb).Error; err != nil {
			fmt.Printf("   ⚠️  创建笔记本失败 %q: %v\n", nd.Name, err)
			continue
		}
		notebooks = append(notebooks, nb)
	}
	return notebooks
}

// seedNotes 创建测试笔记
func seedNotes(db *gorm.DB, tags []models.Tag, notebooks []models.Notebook) []models.Note {
	now := time.Now()

	// 笔记本 ID 映射
	notebookMap := make(map[string]uint)
	for _, nb := range notebooks {
		notebookMap[nb.Name] = nb.ID
	}

	type noteDef struct {
		Title        string
		Content      string
		Pinned       bool
		DaysAgo      int
		FileExt      string
		NotebookName string
	}

	noteData := []noteDef{
		// ========== 默认笔记本 ==========
		{
			Title: "欢迎使用 Jot 🎉",
			Content: `## 欢迎来到 Jot！

Jot 是一个温暖、精致的桌面端卡片式笔记应用。

### 快速上手
- **新建笔记** — 点击左上角"+"按钮或按 Ctrl+N
- **Markdown 支持** — 支持完整的 Markdown 语法
- **分类管理** — 用笔记本和标签组织你的笔记

### 快捷键
| 快捷键 | 功能 |
|--------|------|
| Ctrl+N | 新建笔记 |
| Ctrl+F | 搜索 |
| Ctrl+E | 编辑笔记 |

> 小贴士：试试写一篇笔记吧！
`,
			Pinned:       true,
			DaysAgo:      0,
			FileExt:      ".md",
			NotebookName: "默认笔记本",
		},

		// ========== 工作笔记本 ==========
		{
			Title: "📌 Q4 季度工作总结",
			Content: `## Q4 工作总结

### 核心成果
1. **完成支付模块重构** — 重构了旧版支付接口，统一收银台体验
2. **性能优化上线** — 首页加载速度从 2.3s 降到 0.8s
3. **团队建设** — 新人导师制落地，3 名新同学完成 onboarding

### 数据指标
| 指标 | Q3 | Q4 | 提升 |
|------|----|----|------|
| PV | 120w | 156w | +30% |
| 转化率 | 3.2% | 4.1% | +28% |
| 崩溃率 | 0.8% | 0.3% | -62% |

### 不足
- 需求评审环节耗时过长
- 部分技术债仍需消化

### 下季度计划
1. 微服务拆分第二阶段
2. CI/CD 流水线优化
`,
			Pinned:       true,
			DaysAgo:      1,
			FileExt:      ".md",
			NotebookName: "工作",
		},
		{
			Title: "📅 本周站会记录 (12/16-12/20)",
			Content: `### 周一
- 处理线上告警：用户反馈支付回调延迟
- 定位原因：第三方回调接口超时，已加重试机制

### 周二
- 完成支付模块单元测试补充（覆盖率 65%→82%）
- 下午 Code Review：支付回调 PR

### 周三
- 配合 QA 回归测试支付相关 case
- 修复两个边界条件 bug

### 周四
- 开始设计新结算系统方案
- 输出技术方案初稿

### 周五
- 组内分享：《Go 并发编程最佳实践》
- 下午团建
`,
			Pinned:       false,
			DaysAgo:      3,
			FileExt:      ".md",
			NotebookName: "工作",
		},
		{
			Title: "技术方案：结算系统重构",
			Content: `## 结算系统重构方案

### 背景
现有结算系统耦合严重，扩展性差，难以支持新的分账需求。

### 目标
1. 支持多商户分账
2. 结算周期可配置
3. 全链路可追踪

### 架构设计

~~~
结算服务 → 结算引擎 → 分账模块 → 通知服务
    ↑           ↑            ↑
 结算配置    规则引擎     渠道适配
~~~

### 数据库设计

~~~sql
CREATE TABLE settlement_config (
    id          BIGINT PRIMARY KEY,
    merchant_id BIGINT NOT NULL,
    cycle_type  VARCHAR(20), -- daily/weekly/monthly
    rule_json   TEXT,
    status      TINYINT DEFAULT 1
);

CREATE TABLE settlement_order (
    id            BIGINT PRIMARY KEY,
    config_id     BIGINT,
    amount        DECIMAL(20,2),
    status        VARCHAR(20),
    execute_time  DATETIME,
    created_at    DATETIME
);
~~~

### 风险点
- 历史数据迁移方案尚不明确
- 涉及多团队协调，排期风险
`,
			Pinned:       false,
			DaysAgo:      5,
			FileExt:      ".md",
			NotebookName: "工作",
		},
		{
			Title: "会议纪要：年终述职评审",
			Content: `## 年终述职评审会

**时间**：12月18日 14:00-16:00
**参与人**：张老师 / 李老师 / 王老师 / 我

### 评审要点
1. **项目完成度** — 核心目标全部达成，额外完成安全整改
2. **技术成长** — 引入新架构模式，团队技术分享 4 次
3. **待改进** — 跨团队沟通效率需提升

### 反馈记录
> "今年的产出超出预期，尤其是在系统稳定性方面做出了显著贡献。" — 张老师
> "建议明年在技术影响力方面多投入，可以尝试做内部技术分享。" — 李老师

### 明年计划
- P1: 微服务架构升级
- P2: 建立团队技术规范
- P3: 新人培养体系
`,
			Pinned:       false,
			DaysAgo:      7,
			FileExt:      ".md",
			NotebookName: "工作",
		},
		{
			Title: "代码审查清单",
			Content: `## Code Review Checklist

### 功能性
- [ ] 所有边界条件都覆盖了吗？
- [ ] 错误处理是否完整？不要吞错误
- [ ] 并发安全吗？有没有 race condition？

### 性能
- [ ] 有没有不必要的数据库查询？
- [ ] N+1 问题是否已避免？
- [ ] 内存分配是否合理？

### 可维护性
- [ ] 命名清晰，符合团队规范
- [ ] 函数长度是否合理？（建议 < 50 行）
- [ ] 注释是否有用而不是废话？

### 安全性
- [ ] SQL 注入防护
- [ ] 用户输入校验
- [ ] 敏感信息不记录日志
`,
			Pinned:       false,
			DaysAgo:      10,
			FileExt:      ".md",
			NotebookName: "工作",
		},
		{
			Title: "📊 项目进度追踪",
			Content: `## 支付中台项目进度

| 模块 | 负责人 | 进度 | 状态 |
|------|--------|------|------|
| 收银台 | 小明 | 100% | ✅ 已完成 |
| 路由引擎 | 小红 | 85% | 🚧 联调中 |
| 结算系统 | 我 | 60% | 🚧 开发中 |
| 对账模块 | 小刚 | 30% | ⏳ 排期中 |

### 本周阻塞项
1. 结算系统依赖的商户中心接口延迟交付 🚫
2. 测试环境不稳定，影响联调进度
3. 需要和风控团队对齐新接口规范

### 需协调资源
- 申请增加一台测试服务器
- 需要 QA 提前介入结算模块测试
`,
			Pinned:       false,
			DaysAgo:      12,
			FileExt:      ".md",
			NotebookName: "工作",
		},
		{
			Title: "Go 1.23 新特性学习笔记",
			Content: `## Go 1.23 值得关注的新特性

### Range-over-Func
~~~go
func Backward(s []int) func(func(int) bool) {
    return func(yield func(int) bool) {
        for i := len(s)-1; i >= 0; i-- {
            if !yield(s[i]) {
                return
            }
        }
    }
}
~~~

### 迭代器支持
标准库增加了对自定义迭代器的支持，可以用 range 遍历自定义类型。

### 改进的标准库
- math/rand/v2 性能提升明显
- slices 包新增更多便捷函数
- maps 包新增操作函数

### 实际应用
在支付路由模块中可以用迭代器重构费率计算逻辑，代码会更简洁。
`,
			Pinned:       false,
			DaysAgo:      14,
			FileExt:      ".md",
			NotebookName: "工作",
		},
		{
			Title: "线上故障复盘：支付超时",
			Content: `## 故障复盘报告

**时间**：2026-12-10 14:23 - 14:51
**影响**：支付超时率 12%，影响用户约 3000 人
**等级**：P2

### 时间线
1. 14:23 监控告警：支付超时率突增
2. 14:25 确认问题，开始排查
3. 14:32 定位根因：第三方支付网关 DNS 解析异常
4. 14:40 切换备用线路
5. 14:51 系统恢复，超时率归零

### 根因分析
第三方支付网关的 DNS 服务器短暂不可用，导致我们的请求无法解析。

### 改进措施
- [x] 增加备用 DNS 配置
- [ ] 实现多网关自动切换
- [ ] 增加第三方健康度监测
- [ ] 完善故障演练流程
`,
			Pinned:       false,
			DaysAgo:      16,
			FileExt:      ".md",
			NotebookName: "工作",
		},
		{
			Title: "OKR 设定：2026 Q1",
			Content: `## 2026 Q1 OKR

### O1：提升系统稳定性，可用性达 99.99%
- KR1: 核心链路延迟 P99 < 200ms（当前 350ms）
- KR2: 故障恢复时间 < 5min（当前 ~15min）
- KR3: 自动化测试覆盖率达到 85%

### O2：推进技术架构升级
- KR1: 完成 3 个核心模块微服务化
- KR2: 引入服务网格，逐步替换现有方案
- KR3: 输出架构升级技术文档

### O3：团队建设
- KR1: 组织 6 次内部技术分享
- KR2: 建立 onboarding 文档体系
- KR3: 引入 OKR + 周报组合工作法
`,
			Pinned:       false,
			DaysAgo:      19,
			FileExt:      ".md",
			NotebookName: "工作",
		},

		// ========== 学习笔记本 ==========
		{
			Title: "📖 Rust 入门笔记",
			Content: `## Rust 学习笔记 - 第一章

### 所有权系统

Rust 的核心特性：每个值在任意时刻只有一个所有者。

~~~rust
fn main() {
    let s1 = String::from("hello");
    let s2 = s1; // s1 被移动，不再有效
    // println!("{}", s1); // ❌ 编译错误
    println!("{}", s2); // ✅
}
~~~

### 借用与引用

~~~rust
fn calculate_length(s: &String) -> usize {
    s.len()
}

fn main() {
    let s = String::from("hello");
    let len = calculate_length(&s);
    println!("{}的长度是{}", s, len);
}
~~~

### 生命周期

~~~rust
fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() > y.len() { x } else { y }
}
~~~
`,
			Pinned:       true,
			DaysAgo:      1,
			FileExt:      ".md",
			NotebookName: "学习",
		},
		{
			Title: "📝 算法题：动态规划总结",
			Content: `## 动态规划解题模板

### 核心思路
1. 定义状态
2. 写出状态转移方程
3. 确定初始条件
4. 确定遍历顺序

### 经典题目

#### 背包问题

~~~go
// 0-1 背包
func knapsack(weights, values []int, capacity int) int {
    dp := make([]int, capacity+1)
    for i := 0; i < len(weights); i++ {
        for w := capacity; w >= weights[i]; w-- {
            dp[w] = max(dp[w], dp[w-weights[i]] + values[i])
        }
    }
    return dp[capacity]
}
~~~

#### 最长递增子序列

~~~go
func lengthOfLIS(nums []int) int {
    tails := make([]int, 0)
    for _, num := range nums {
        idx := sort.SearchInts(tails, num)
        if idx == len(tails) {
            tails = append(tails, num)
        } else {
            tails[idx] = num
        }
    }
    return len(tails)
}
~~~

### 刷题建议
- 先暴力再优化
- 重点理解状态定义
- 多画 DP 表格
`,
			Pinned:       false,
			DaysAgo:      4,
			FileExt:      ".md",
			NotebookName: "学习",
		},
		{
			Title: "🎓 计算机组成原理复习",
			Content: `## 计算机组成原理重点

### 存储层次
| 层级 | 容量 | 速度 | 价格 |
|------|------|------|------|
| 寄存器 | ~1KB | 0.3ns | 最高 |
| L1 Cache | ~32KB | 1ns | ↑ |
| L2 Cache | ~256KB | 3ns | ↑ |
| L3 Cache | ~8MB | 12ns | ↑ |
| 内存 | ~16GB | 50ns | ↓ |
| SSD | ~512GB | 10μs | 低 |

### CPU 流水线
- IF（取指）→ ID（译码）→ EX（执行）→ MEM（访存）→ WB（写回）
- 冒险类型：结构冒险 / 数据冒险 / 控制冒险
- 解决方案：转发（Forwarding）+ 分支预测

### 中断机制
- 中断响应过程：关中断 → 保存断点 → 中断服务
- 中断优先级：硬件故障 > 时钟 > I/O
`,
			Pinned:       false,
			DaysAgo:      6,
			FileExt:      ".md",
			NotebookName: "学习",
		},
		{
			Title: "🐳 Kubernetes 学习路线",
			Content: `## K8s 学习路线图

### 基础概念
- [x] Pod、Service、Deployment
- [x] ConfigMap、Secret
- [ ] Ingress、PV/PVC
- [ ] Namespace、RBAC

### 进阶内容
- [ ] Operator 模式
- [ ] 自定义 CRD
- [ ] Service Mesh（Istio）
- [ ] Helm 包管理

### 实践项目
1. 部署一个 Go Web 应用到 K8s ✅
2. 配置自动扩缩容 HPA 🚧
3. 实现蓝绿部署 🚧
4. 搭建监控体系（Prometheus + Grafana）📅

### 常用命令

~~~
kubectl get pods -o wide
kubectl describe pod <name>
kubectl logs -f <pod> -c <container>
kubectl exec -it <pod> -- sh
kubectl port-forward svc/<svc> 8080:80
~~~
`,
			Pinned:       false,
			DaysAgo:      8,
			FileExt:      ".md",
			NotebookName: "学习",
		},
		{
			Title: "📚 《系统设计面试》读书笔记",
			Content: `## 《System Design Interview》笔记

### 设计 Twitter 时间线
- **需求**：推文发布、时间线查看、关注/取关
- **核心挑战**：写扩散（Fanout）vs 读扩散
- **方案**：混合策略 — 大 V 用读扩散，普通用户用写扩散

### 设计短视频平台
- **写路径**：视频上传 → 转码 → CDN 分发
- **读路径**：CDN 就近分发 → 客户端解码 → 播放
- **关键组件**：分片存储、转码队列、CDN 预热

### 设计聊天系统
- 长连接管理：WebSocket + 心跳保活
- 消息顺序：时钟戳 + 序列号
- 离线消息：消息队列 + 持久化

### 通用技巧
1. 先明确需求，不要直接设计方案
2. 估算 QPS 和数据量
3. 从简到繁，逐步优化
`,
			Pinned:       false,
			DaysAgo:      11,
			FileExt:      ".md",
			NotebookName: "学习",
		},
		{
			Title: "🎯 LeetCode 刷题进度",
			Content: `## 刷题追踪

### 已完成
| # | 题目 | 难度 | 类型 |
|---|------|------|------|
| 1 | 两数之和 | 🟢 | 哈希表 |
| 15 | 三数之和 | 🟡 | 双指针 |
| 42 | 接雨水 | 🔴 | 单调栈 |
| 53 | 最大子数组和 | 🟡 | DP |
| 121 | 买卖股票最佳时机 | 🟢 | DP |
| 200 | 岛屿数量 | 🟡 | DFS |
| 206 | 反转链表 | 🟢 | 链表 |
| 215 | 数组第K大 | 🟡 | 快排 |

### 计划
- 本周：二叉树专题（5 道）
- 下周：图论专题（3 道）
- 本月目标：累计 50 道
`,
			Pinned:       false,
			DaysAgo:      13,
			FileExt:      ".md",
			NotebookName: "学习",
		},
		{
			Title: "🌐 HTTP/3 与 QUIC 协议学习",
			Content: `## HTTP/3 & QUIC 笔记

### 为什么需要 QUIC？
- TCP 队头阻塞（Head-of-Line Blocking）
- 连接建立需要 1.5 个 RTT
- 网络切换时需要重新建立连接

### QUIC 核心特性

1. **0-RTT 连接** — 缓存会话票据，首次即可发送数据
2. **多路复用无阻塞** — 一个流丢包不影响其他流
3. **连接迁移** — 连接标识符不依赖 IP，切换网络不中断
4. **内置加密** — TLS 1.3 原生集成

### HTTP/3 帧结构
~~~
QUIC Packet
├── QUIC Header（公共头部）
└── QUIC Frame
    ├── STREAM（数据流）
    ├── ACK（确认）
    └── ...
~~~

### 实际收益
- 弱网环境下性能提升显著（20-30%）
- 移动端网络切换不断连
`,
			Pinned:       false,
			DaysAgo:      15,
			FileExt:      ".md",
			NotebookName: "学习",
		},
		{
			Title: "🎬 英语听力练习记录",
			Content: `## 英语听力练习日志

### 本周练习
- Day 1: TED Talk - "The Power of Vulnerability" ✓
- Day 2: BBC News 15min ✓
- Day 3: 播客 "Techmeme Ride Home" ✓
- Day 4: 美剧片段听写练习
- Day 5: 复习本周生词

### 难点
- 连读（Connected Speech）："gonna"、"wanna"
- 弱读：介词和冠词在快速语流中几乎消失
- 不同口音：印度英语、英式英语

### 工具推荐
- **YouGlish** — 查单词在不同口音中的发音
- **Ororo.tv** — 双语字幕看美剧
- **Anki** — 间隔重复记忆生词
`,
			Pinned:       false,
			DaysAgo:      18,
			FileExt:      ".txt",
			NotebookName: "学习",
		},

		// ========== 生活笔记本 ==========
		{
			Title: "🍳 周末烘焙记录",
			Content: `## 第一次做戚风蛋糕

### 配方
- 鸡蛋 5 个
- 低筋面粉 85g
- 细砂糖 70g（蛋白用）+ 30g（蛋黄用）
- 牛奶 40g
- 玉米油 40g
- 柠檬汁 几滴

### 步骤
1. 蛋黄蛋清分离，蛋清放冰箱冷藏
2. 蛋黄 + 糖搅匀，加油和牛奶
3. 筛入低筋面粉，Z 字形搅拌
4. 蛋白打至硬性发泡
5. 混合蛋白霜和蛋黄糊
6. 150°C 烘烤 50min

### 结果
✅ 成功！组织细腻，回弹很好。下次可以试试抹茶口味。
`,
			Pinned:       false,
			DaysAgo:      2,
			FileExt:      ".md",
			NotebookName: "生活",
		},
		{
			Title: "🏃 运动打卡记录",
			Content: `## 本周运动记录

| 日期 | 项目 | 时长 | 消耗 |
|------|------|------|------|
| 周一 | 跑步 | 5km / 28min | 320kcal |
| 周三 | 游泳 | 1h | 450kcal |
| 周五 | 力量训练 | 45min | 280kcal |
| 周日 | 羽毛球 | 1.5h | 500kcal |

### 本月目标
- [x] 跑步 50km（已完成 42km）
- [ ] 游泳 4 次（已完成 2 次）
- [ ] 体脂率降到 20%（目前 21.5%）

### 感受
坚持运动一个月，睡眠质量明显改善，白天精力更充沛了。继续加油 💪
`,
			Pinned:       false,
			DaysAgo:      4,
			FileExt:      ".md",
			NotebookName: "生活",
		},
		{
			Title: "📺 近期追剧清单",
			Content: `## 正在追

### 《漫长的季节》⭐9.4
> 一部让人笑着笑着就哭了的剧。范伟的演技封神，剧本扎实，每个角色都立体鲜活。

### 《繁花》⭐8.7
王家卫的首部电视剧，光影美学拉满，故事节奏稍慢但后劲十足。

## 已看完
- 《我的解放日志》⭐9.0 — 治愈系韩剧天花板
- 《黑暗荣耀》⭐8.9 — 复仇爽剧，金恩淑编剧功力深厚
- 《Signal》⭐9.2 — 穿越时空的推理神剧

## 想看
- [ ] 《三体》（Netflix 版）
- [ ] 《The Last of Us》S2
`,
			Pinned:       false,
			DaysAgo:      6,
			FileExt:      ".md",
			NotebookName: "生活",
		},
		{
			Title: "✈️ 春节出行计划",
			Content: `## 春节行程安排

### Day 1 (除夕) — 回家
- 早上 9:00 高铁 G1234
- 下午到家，帮爸妈准备年夜饭
- 晚上看春晚、守岁 🎆

### Day 2-3 (初一初二) — 走亲戚
- 初一：大伯家拜年
- 初二：外婆家聚餐

### Day 4-5 — 短途旅行
- 想去附近的温泉度假村
- 自驾 2h 车程
- 住一晚，泡温泉 ♨️

### 预算
| 项目 | 费用 |
|------|------|
| 交通 | 800 |
| 礼物 | 1500 |
| 旅行 | 2000 |
| 其他 | 700 |
| **合计** | **5000** |
`,
			Pinned:       false,
			DaysAgo:      9,
			FileExt:      ".md",
			NotebookName: "生活",
		},
		{
			Title: "🌿 阳台盆栽记录",
			Content: `## 阳台小花园日志

### 多肉们 🌵
- 玉露：状态不错，叶片晶莹剔透 ✅
- 熊童子：掉了几片叶子，可能水浇多了 ⚠️
- 吉娃莲：开始出状态了，叶尖变红 ✅

### 绿植 🪴
- 龟背竹：新叶展开中，长得很快
- 绿萝：剪了几枝水培，已经生根了
- 薄荷：长势疯狂，每周都能摘一次泡水喝

### 注意事项
- 多肉控水，两周浇一次
- 绿植每周施肥一次（薄肥勤施）
- 定期检查有无病虫害

> "养植物最能培养耐心了" —— 我妈说的
`,
			Pinned:       false,
			DaysAgo:      11,
			FileExt:      ".txt",
			NotebookName: "生活",
		},
		{
			Title: "🎮 游戏通关记录",
			Content: `## 2026 年游戏记录

### 已通关 ✅
| 游戏 | 平台 | 时长 | 评分 |
|------|------|------|------|
| 黑神话：悟空 | PC | 60h | ⭐⭐⭐⭐⭐ |
| 艾尔登法环 | PC | 120h | ⭐⭐⭐⭐⭐ |
| 博德之门 3 | PC | 90h | ⭐⭐⭐⭐½ |

### 进行中 🎮
- **赛博朋克 2077：往日之影** — 20h，剧情渐入佳境
- **Hades II** — 还在 Early Access，等正式版

### 心愿单
- [ ] 《丝之歌》（什么时候出啊...）
- [ ] 《荒野大镖客 3》（如果出了的话）
- [ ] 《GTA 6》
`,
			Pinned:       false,
			DaysAgo:      13,
			FileExt:      ".md",
			NotebookName: "生活",
		},
		{
			Title: "💊 年度体检报告摘要",
			Content: `## 2026 年度体检结果

### 正常 ✅
- 血压：118/76 mmHg
- 心率：68 bpm
- 血常规：各项指标正常
- 肝功能：正常
- 肾功能：正常

### 需关注 ⚠️
- 视力：左眼 5.0 / 右眼 4.6（轻度加深）
- 总胆固醇：5.8（略偏高，建议控制饮食）
- BMI：23.5（正常范围上限）

### 医生建议
1. 减少久坐，每小时起来活动一下
2. 控制高脂食物摄入
3. 每年复查一次视力
4. 保持规律运动

### 行动
- [x] 换了一副新眼镜
- [x] 买了一个升降桌
- [ ] 开始控制饮食
`,
			Pinned:       false,
			DaysAgo:      16,
			FileExt:      ".md",
			NotebookName: "生活",
		},
		{
			Title: "🎵 2026 年度歌单",
			Content: `## 我的年度歌曲 Top 10

1. 《鲜花》— 回春丹
2. 《向云端》— 黄绮珊
3. 《大梦》— 瓦依那 / 任素汐
4. 《生活倒影》— 苏运莹
5. 《有没有》— 韦礼安
6. 《山雀》— 万能青年旅店
7. 《凄美地》— 郭顶
8. 《一般的一天》— 汪苏泷
9. 《蚂蚁筑起高塔》— 逃跑计划
10. 《我记得》— 赵雷

> 今年听了 2800+ 首歌，最爱的还是民谣和独立摇滚。
`,
			Pinned:       false,
			DaysAgo:      19,
			FileExt:      ".txt",
			NotebookName: "生活",
		},

		// ========== 个人项目笔记本 ==========
		{
			Title: "🎯 Jot 项目开发计划",
			Content: `## Jot 笔记应用开发路线图

### v0.1 ✅ 已完成
- [x] 基础笔记 CRUD
- [x] Markdown 编辑与预览
- [x] 笔记本分类
- [x] 标签系统

### v0.2 🚧 进行中
- [x] 搜索功能
- [ ] 笔记导出
- [ ] 回收站
- [x] 设置页面

### v0.3 📅 规划中
- [ ] 笔记附件
- [ ] 批量操作
- [ ] 数据统计
- [ ] 自动保存

### v1.0 🌟 未来
- [ ] 暗色主题完善
- [ ] 性能优化
- [ ] 跨平台适配
- [ ] 插件系统
`,
			Pinned:       true,
			DaysAgo:      0,
			FileExt:      ".md",
			NotebookName: "个人项目",
		},
		{
			Title: "🐛 Bug 追踪：搜索中文分词",
			Content: `## Bug: 中文搜索分词不准确

**优先级**：P2
**状态**：修复中

### 问题描述
搜索中文关键词时，部分相关笔记无法匹配到。例如搜索"笔记"时，标题包含"笔记"的笔记都能搜到，但内容中包含"笔记本"的笔记却匹配不上。

### 根因分析
当前使用 SQLite 的 LIKE 查询，对中文没有分词支持。LIKE '%笔记%' 无法匹配到"笔记本"（虽然可以，但反过来不行）。更严重的是，搜索"学习笔记"时无法匹配到"Go 学习笔记"以外的内容。

### 解决方案

#### 方案 A：SQLite FTS5（推荐）
~~~sql
CREATE VIRTUAL TABLE notes_fts USING FTS5(title, content);
~~~
- 支持中文分词（需额外配置）
- 全文索引，性能好
- 支持排名和片段高亮

#### 方案 B：LIKE 优化
- 简单但功能有限
- 无需迁移

### 下一步
尝试集成 FTS5，评估效果和性能开销。
`,
			Pinned:       false,
			DaysAgo:      3,
			FileExt:      ".md",
			NotebookName: "个人项目",
		},
		{
			Title: "💡 功能提案：笔记模板",
			Content: `## 功能提案：笔记模板系统

### 需求背景
写周报、会议纪要等固定格式笔记时，每次都手动复制模板很麻烦。

### 设计方案

#### 模板管理
1. 内置模板：周报、会议纪要、读书笔记、TODO
2. 自定义模板：用户可创建自己的模板
3. 模板市场：分享和下载社区模板

#### 技术实现
~~~typescript
interface NoteTemplate {
    id: string;
    name: string;
    content: string;      // Markdown
    category: string;     // built-in / custom
    tags?: string[];
    created_at: Date;
}
~~~

#### UI 交互
- 新建笔记时可选择模板
- 支持预览模板内容
- 使用模板后仍可自由编辑

### 优先级
高 — 能显著提升日常使用效率。
`,
			Pinned:       false,
			DaysAgo:      6,
			FileExt:      ".md",
			NotebookName: "个人项目",
		},
		{
			Title: "📊 数据库表结构设计",
			Content: `## Jot 数据库 Schema

### 核心表

~~~sql
-- 笔记表
CREATE TABLE notes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL DEFAULT '',
    note_type  TEXT NOT NULL DEFAULT 'markdown',  -- markdown / text
    notebook_id INTEGER REFERENCES notebooks(id),
    pinned     BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

-- 笔记本表
CREATE TABLE notebooks (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    sort_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

-- 标签表
CREATE TABLE tags (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    color      TEXT DEFAULT '#3b82f6',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 笔记-标签关联表
CREATE TABLE note_tags (
    note_id INTEGER REFERENCES notes(id),
    tag_id  INTEGER REFERENCES tags(id),
    PRIMARY KEY (note_id, tag_id)
);
~~~

### 索引设计
- notes: (notebook_id), (deleted_at), (updated_at)
- notebook_tags: (notebook_id)
`,
			Pinned:       false,
			DaysAgo:      9,
			FileExt:      ".md",
			NotebookName: "个人项目",
		},
		{
			Title: "📦 CI/CD 配置记录",
			Content: `## GitHub Actions 工作流配置

### 构建配置

~~~yaml
name: Build

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Build
        run: |
          go build -v ./...
          cd frontend && npm ci && npm run build

      - name: Test
        run: go test -v ./...
~~~

### 发布配置
- 在 GitHub Release 创建时自动触发
- 使用 goreleaser 跨平台构建
- 自动上传构建产物到 Release
`,
			Pinned:       false,
			DaysAgo:      12,
			FileExt:      ".md",
			NotebookName: "个人项目",
		},
		{
			Title: "🎨 UI 设计灵感收集",
			Content: `## 界面设计参考

### 配色方案
- 主色调：琥珀色 (#D97706)
- 背景：暖白 (#FDFBF7)
- 卡片：纯白 (#FFFFFF)
- 文字：深灰 (#1F2937)

### 交互参考
- **Linear** — 极简的项目管理，微交互动效很优雅
- **Notion** — 灵活的块编辑器，设计规范统一
- **Obsidian** — 知识图谱和双向链接

### 想实现的动效
1. 笔记卡片 hover 上浮 + 阴影加深
2. 侧栏展开/收起平滑过渡
3. 删除笔记时微缩消失
4. 新建笔记时淡入

### Design Tokens

~~~css
:root {
    --accent: #D97706;
    --bg: #FDFBF7;
    --card-bg: #FFFFFF;
    --text-primary: #1F2937;
    --radius-sm: 6px;
    --radius-md: 8px;
    --radius-lg: 12px;
}
~~~
`,
			Pinned:       false,
			DaysAgo:      15,
			FileExt:      ".md",
			NotebookName: "个人项目",
		},
		{
			Title: "🧪 性能优化记录",
			Content: `## Jot 性能优化日志

### 当前瓶颈分析

| 指标 | 当前值 | 目标值 | 状态 |
|------|--------|--------|------|
| 启动时间 | 1.2s | < 500ms | 🚧 |
| 笔记列表加载 | 280ms | < 100ms | 🚧 |
| 搜索响应 | 150ms | < 50ms | 📅 |
| 编辑器启动 | 400ms | < 200ms | ✅ |

### 已做的优化
1. **懒加载** — 编辑器组件按需加载 ✅
2. **虚拟列表** — 长列表只渲染可视区域 📅
3. **数据库索引** — 增加 updated_at 索引 ✅
4. **缓存** — 笔记本列表内存缓存 ✅

### 下一步
- 使用 Web Worker 做 Markdown 渲染
- 图片懒加载
- 笔记内容分页加载
`,
			Pinned:       false,
			DaysAgo:      18,
			FileExt:      ".md",
			NotebookName: "个人项目",
		},
		{
			Title: "📝 开源许可证选择",
			Content: `## 开源许可证对比

###  contenders
| 许可证 | 商用 | 修改后闭源 | 需注明出处 | 备注 |
|--------|------|------------|------------|------|
| MIT | ✅ | ✅ | ✅ | 最宽松，推荐 |
| Apache 2.0 | ✅ | ✅ | ✅ | 含专利授权 |
| GPL v3 | ✅ | ❌ | ✅ | 强传染性 |
| AGPL v3 | ✅ | ❌ | ✅ | 网络使用也传染 |

### 选择
Jot 选择 **MIT** 许可证 — 简单、宽松、对用户友好。

### LICENSE 文件
~~~
MIT License

Copyright (c) 2026 Jot

Permission is hereby granted...
~~~
`,
			Pinned:       false,
			DaysAgo:      21,
			FileExt:      ".md",
			NotebookName: "个人项目",
		},

		// ========== 随笔笔记本 ==========
		{
			Title: "🌅 关于慢下来的思考",
			Content: `## 慢下来

最近读了一篇文章，说现代人最大的问题不是效率不够，而是"停不下来"。

想想确实是这样：

- 吃饭要倍速看视频
- 通勤要听播客"学习"
- 连休假都要安排得满满的

**慢下来的好处：**
1. 注意力更集中
2. 思考更深
3. 体验更丰富
4. 焦虑更少

### 今天的小尝试
下班后散步 30 分钟，不带耳机，不看手机。
看看路边的树，听听鸟叫，感受晚风。
意外的很放松。

> "人生不是赛跑，而是旅程。" —— 虽然老套，但说得对。
`,
			Pinned:       false,
			DaysAgo:      2,
			FileExt:      ".md",
			NotebookName: "随笔",
		},
		{
			Title: "📖 读《百年孤独》有感",
			Content: `## 初读《百年孤独》

马尔克斯的文字像是有魔力。

"多年以后，面对行刑队，奥雷里亚诺·布恩迪亚上校将会回想起父亲带他去见识冰块的那个遥远的下午。"

这个开头被无数人分析过，但读到书末才真正理解它的精妙：
- 过去："那个遥远的下午"
- 现在："将会回想起"
- 未来："面对行刑队"

一句话把三个时间维度折叠在一起，像预言，又像回忆。

### 一些摘抄
> "一个人不是在该死的时候死，而是在能死的时候死。"

> "无论走到哪里，都应该记住，过去都是假的。"

### 读完的感觉
像是做了一场漫长的梦。梦醒了，但梦中那些人和事的样子还在脑海里挥之不去。

准备过段时间二刷。
`,
			Pinned:       false,
			DaysAgo:      5,
			FileExt:      ".md",
			NotebookName: "随笔",
		},
		{
			Title: "☕️ 咖啡馆探店 #3：社区小店",
			Content: `## 藏在社区里的宝藏咖啡馆

路过一家藏在老小区里的咖啡馆，门面很小，差点错过。

### 环境
- 只有 4 张桌子，老板一个人在打理
- 装修很 vintage，放的是爵士乐
- 窗户正对着小区花园，很安静

### 咖啡
点了一杯 Dirty，老板说是用深烘豆做的。
- 入口层次感分明
- 油脂很饱满
- 性价比高（才 22 块）

### 感受
比起连锁咖啡店，这种社区小店更有"人情味"。
老板会记得熟客的口味，会和你聊咖啡豆的产地故事。

> 开一家这样的小店，似乎是很多人的梦想。但真正去做的，都是勇敢的人。

### 收集到的咖啡店清单
- [x] #1 叁咖啡（市中心）
- [x] #2 小满咖啡（大学路）
- [x] #3 此刻咖啡（老小区）
- [ ] #4 山丘咖啡
- [ ] #5 河岸咖啡
`,
			Pinned:       false,
			DaysAgo:      7,
			FileExt:      ".md",
			NotebookName: "随笔",
		},
		{
			Title: "🎬 电影《好东西》观后感",
			Content: `## 关于《好东西》

看完点映，心情很复杂。

### 短评
一部"女性主义"电影，但又不只是"女性主义"。
它讲的是：一个女性如何在不完美的世界里，努力做自己。

### 喜欢的镜头
小女孩茉莉在录音室打鼓的那场戏。
鼓声震耳欲聋，但她的表情是自由的。

### 对话摘录
> "我正直勇敢有阅读量，我有什么好可怜的？"

> "正是因为我们足够乐观和自信，我们才能直面悲剧。"

### 评分
⭐⭐⭐⭐½

### 推荐
推荐所有女生去看，也推荐所有男生去看。
这是一部能让人思考的电影，不是消费焦虑，而是给出力量。
`,
			Pinned:       false,
			DaysAgo:      10,
			FileExt:      ".md",
			NotebookName: "随笔",
		},
		{
			Title: "🌙 深夜随笔：关于习惯",
			Content: `## 关于习惯的一些想法

最近在尝试养成几个小习惯：

### 早晨
1. 起床先喝一杯温水 ✅
2. 冥想 10 分钟 ✅
3. 写晨间日记 ✅

### 晚间
1. 23:00 前放下手机 ⚠️ 还需要努力
2. 读 20 页书 ✅
3. 写明日计划 ✅

### 感悟
养成习惯最难的不是"开始"，而是"坚持"。
但坚持的关键，不是靠"意志力"，而是靠"降低门槛"。

想把习惯坚持下去，就要让它:
- **简单**到无法拒绝
- **具体**到不用思考
- **可见**到能追踪进度

> 种一棵树最好的时间是十年前，其次是现在。
`,
			Pinned:       false,
			DaysAgo:      14,
			FileExt:      ".txt",
			NotebookName: "随笔",
		},
		{
			Title: "📸 摄影入门学习记录",
			Content: `## 摄影学习日志

### 曝光三要素

| 要素 | 作用 | 常用值 |
|------|------|--------|
| 光圈 (Aperture) | 控制进光量 + 景深 | f/2.8 虚化, f/8 风景 |
| 快门 (Shutter) | 控制曝光时间 + 动态 | 1/125s 日常, 1/500s 运动 |
| 感光度 (ISO) | 控制传感器灵敏度 | ISO 100 最佳, ISO 3200 噪点多 |

### 本周练习
- 主题：街头摄影
- 设备：富士 X-T5 + 35mm f/2
- 拍了大概 200 张，选出了 5 张满意的

### 学到的东西
1. 光线比器材重要
2. 构图比参数重要
3. 多拍多看是唯一捷径

### 想拍的题材
- [ ] 城市夜景长曝光
- [ ] 人像写真
- [ ] 星空
- [ ] 街头的决定性瞬间
`,
			Pinned:       false,
			DaysAgo:      17,
			FileExt:      ".md",
			NotebookName: "随笔",
		},
		{
			Title: "🎄 十二月，一年将尽",
			Content: `## 十二月记

十二月了，一年又要过去了。

### 翻翻相册
- 三月：和朋友去了趟大理，洱海的风现在还记忆犹新
- 六月：换了新工作，适应了新环境
- 九月：开始学 Rust，写了第一个能跑的程序
- 十一月：看了最喜欢的乐队的现场

### 年末三问
**今年最开心的三件事？**
1. 大理旅行
2. 新工作带来的成长
3. 开始规律运动

**今年最大的遗憾？**
没有多陪陪家人。

**明年最想做的事？**
做一个自己的开源项目。

### 给明年的自己
希望你能保持好奇，保持热情。
少焦虑，多行动。
少熬夜，多睡觉。
少想，多做。
`,
			Pinned:       false,
			DaysAgo:      20,
			FileExt:      ".txt",
			NotebookName: "随笔",
		},
	}

	var notes []models.Note
	for _, nd := range noteData {
		createdAt := now.AddDate(0, 0, -nd.DaysAgo).Add(-time.Duration(rand.Intn(86400)) * time.Second)
		updatedAt := createdAt.Add(time.Duration(rand.Intn(7200)) * time.Second)

		notebookID := notebookMap[nd.NotebookName]
		if notebookID == 0 {
			notebookID = notebookMap["默认笔记本"]
		}

		note := models.Note{
			Title:      nd.Title,
			Content:    nd.Content,
			FileExt:    nd.FileExt,
			Pinned:     nd.Pinned,
			NotebookID: notebookID,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
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
	tagMap := make(map[string]models.Tag)
	for _, tag := range tags {
		tagMap[tag.Name] = tag
	}

	for i, note := range notes {
		// 为每条笔记分配 1~3 个随机标签
		numTags := 1 + rand.Intn(3)
		if numTags > len(tags) {
			numTags = len(tags)
		}

		perm := rand.Perm(len(tags))
		for j := 0; j < numTags; j++ {
			tag := tags[perm[j]]
			if err := db.Model(&note).Association("Tags").Append(&tag); err != nil {
				fmt.Printf("   ⚠️  关联标签失败 (笔记#%d, 标签#%d): %v\n", note.ID, tag.ID, err)
			}
		}

		// 特定笔记额外关联"待办"标签
		if i < 2 {
			if tag, ok := tagMap["待办"]; ok {
				_ = db.Model(&note).Association("Tags").Append(&tag)
			}
		}
	}
}
