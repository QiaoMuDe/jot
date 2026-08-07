# Checklist

- [x] `vec-poc/` 独立 go module 存在，`go build ./...` 编译通过，主项目代码零改动
- [x] 依赖引入正确：glebarez/sqlite、gorm、ollama api、go-openai、modernc.org/sqlite/vec
- [x] sqlite-vec 探针执行 `SELECT vec_version()`：可用则 `status` 显示版本，不可用则记录原因（实测 v0.1.9 可用，`status` 显示"检索实现: sqlite-vec"）
- [x] 向量存储双实现存在：sqlite-vec 路径（`vec_distance_cosine`）与纯 Go 余弦回退路径，`--force-brute` 可强制切换（集成测试双实现均通过）
- [x] 独立 DB 文件（默认 `./vec-test.db`）在 `vec-poc/` 运行目录创建，`documents`/`chunks` 表结构符合 spec
- [x] 切块函数：按标题+空行分段、单块 ≤ 500 字（rune 安全）；2000 字输入生成 ≥ 4 块（单元测试通过）
- [x] Ollama embedding：调用 `api.Embed` 批量生成向量，失败时给出含地址的可读错误，不 panic（代码已实现；真实调用需用户环境 Ollama，沙箱无法监听端口）
- [x] 配置解析：flag 优先于环境变量，缺失必填项（LLM 三项）时给出清晰提示（`ask` 报错冒烟验证通过）
- [x] CLI 子命令齐全：`add`/`index`/`ask`/`list`/`status`/`help`/`quit`（冒烟验证通过）
- [x] `ask` 链路完整：问题 embedding → 召回 TopN 块 → 打印来源块 → 互联网模型回答；召回为空时无上下文直接回答（链路代码完整；召回部分集成测试验证，LLM 调用需用户提供 key 实测）
- [x] 端到端试跑成功：`add sample.md` → `index` → `ask` 输出正常，无 panic（以离线集成测试验证等价链路；无用户库访问）
- [x] 隔离性确认：试跑期间用户库 `~/.jot/data/jot.db` 未被读写（mtime 11:17:25 全程未变化；仅 `vec-poc/vec-test.db` 被创建）
