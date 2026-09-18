# Check List

- [x] settings 表种子包含 `ai_chunk_target_rumes=600` / `ai_chunk_max_rumes=1500`，且仅缺失 key 插入
- [x] `SettingsConfig` 含两字段，`GetAllSettings`/`SaveAllSettings` 读写映射齐全，json tag 与 key 一致
- [x] `SaveAllSettings` 对 target/max 做 clamp（target∈[1,max]、max∈[100,10000]、max>=target）
- [x] 设置页「AI 设置」分组可见两个数字输入，修改即保存后刷新仍保留（落库）
- [x] `ChunkContent(content, targetRunes, maxRunes, meta)` 签名生效，硬切预算用 max、段落落刀用 target
- [x] 大节整段（介于 target..max）不被切开；纯文本聚合到 target 在段落边界落刀；单段超 max 才硬切
- [x] `chunkMaxRunes` 常量已移除，`IndexNotes` 与 `classifyVectorNotes` 均从配置读取同一份 target/max
- [x] 现有 chunk 测试在 target=max=600 下全绿（旧行为等价验证）
- [x] clamp 单测断言覆盖越界与 target>max 场景
- [x] `gofmt`/`go build`/`go vet`/`go test ./internal/services/` 全绿
- [x] AGENTS.md 临时记忆已记录本次改动