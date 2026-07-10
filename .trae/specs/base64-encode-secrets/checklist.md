# Checklist

- [x] Task 1: `internal/services/crypto.go` 创建，`EncodeB64` / `DecodeB64` / `isEncoded` 函数可用
- [x] Task 2: `AIService.SaveConfig()` 写入 DB 时密钥字段已带前缀编码
- [x] Task 3: `AIService.GetConfig()` 返回前端时密钥字段已解码
- [x] Task 4: `ProfileService` 的 Create/Update/Switch/List 中 `api_key` 正确编解码
- [x] Task 5: 应用启动时自动迁移存量明文密钥（无前缀→编码，有前缀→跳过）
- [x] 迁移幂等性：反复启动，数据不会重复编码（`(zk)` 前缀检查防重复）
- [x] 编解码一致性：保存后再读取，密钥值与原值完全一致
- [x] 编译通过：`go build ./...` 无报错
