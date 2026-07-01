# Checklist

- [x] Task 1: `internal/models/api_profile.go` 存在，APIProfile 结构体包含 ID/Name/Provider/BaseURL/APIKey/CreatedAt 六个字段
- [x] Task 1: 自动迁移已将 `api_profiles` 表创建到数据库中
- [x] Task 2: `internal/services/profile_service.go` 存在，包含 ListProfiles/CreateProfile/UpdateProfile/DeleteProfile/SwitchProfile 五个方法
- [x] Task 2: SwitchProfile 正确调用 SettingService.Set() 写入 ai_provider / ai_base_url / ai_api_key 三条记录
- [x] Task 3: `app.go` 中 GetProfiles/CreateProfile/UpdateProfile/DeleteProfile/SwitchProfile 五个绑定方法已添加
- [x] Task 3: 启动迁移逻辑：api_profiles 表空 + settings 表 ai_base_url 非空 → 自动创建"默认配置"预设
- [x] Task 4: `index.html` 中「API 连接」区域顶部新增配置预设选择行（标签 + 下拉 + 新增按钮 + 管理按钮）
- [x] Task 4: 新增/编辑弹窗表单包含：名称输入、服务商下拉、API 地址输入、API Key 输入、保存/取消按钮
- [x] Task 5: `settings-panel.css` 中包含预设选择器/弹窗/提示的完整样式
- [x] Task 6: `main.js` 中 loadProfiles() 在 loadAISettings() 中被调用，下拉列表正确渲染
- [x] Task 6: 切换预设后，URL/Key/服务商输入框自动填充，模型下拉重置
- [x] Task 6: 新增/编辑弹窗的保存按钮正确调用后端 CreateProfile/UpdateProfile
- [x] Task 6: 删除预设前弹出确认对话框
- [x] Task 7: 应用启动后 `api_profiles` 表已存在
- [x] Task 7: 已有配置的用户首次启动，出现"默认配置"预设且选中
- [x] Task 7: 新增预设 → 下拉出现新项且自动切换
- [x] Task 7: 切换预设 → settings 表中 ai_base_url / ai_api_key / ai_provider 对应更新
- [x] Task 7: 手动修改 URL/Key 后切换预设 → 值被预设覆盖
