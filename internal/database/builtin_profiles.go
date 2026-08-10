package database

import (
	"jot/internal/models"

	"gorm.io/gorm"
)

// InitBuiltinProfiles 增量插入内置 API 服务商预设（仅插入不存在的）
// 用户可在 builtinProfiles 切片中按相同格式逐条添加更多服务商
func InitBuiltinProfiles(db *gorm.DB) error {
	// 查询所有已有预设的名称
	var existingNames []string
	db.Model(&models.APIProfile{}).Pluck("name", &existingNames)
	existing := make(map[string]bool, len(existingNames))
	for _, n := range existingNames {
		existing[n] = true
	}

	builtinProfiles := []models.APIProfile{
		{
			Name:    "DeepSeek",
			BaseURL: "https://api.deepseek.com",
		},
		{
			Name:    "VIP 中转",
			BaseURL: "https://vip.j3gb.com/v1",
		},
		{
			Name:    "OpenRouter",
			BaseURL: "https://openrouter.ai/api/v1",
		},
		{
			Name:    "NVIDIA",
			BaseURL: "https://integrate.api.nvidia.com/v1",
		},
		{
			Name:    "智谱 GLM",
			BaseURL: "https://open.bigmodel.cn/api/paas/v4",
		},
		{
			Name:    "小米 Mimo",
			BaseURL: "https://api.xiaomimimo.com/v1",
		},
		{
			Name:    "小米 Mimo TokenPlan",
			BaseURL: "https://token-plan-cn.xiaomimimo.com/v1",
		},
		{
			Name:    "商汤日日新",
			BaseURL: "https://token.sensenova.cn/v1",
		},
		{
			Name:    "Ollama",
			BaseURL: "http://localhost:11434/v1",
		},
		{
			Name:    "阶跃星辰",
			BaseURL: "https://api.stepfun.com/v1",
		},
		{
			Name:    "阶跃星辰 TokenPlan",
			BaseURL: "https://api.stepfun.com/step_plan/v1",
		},
		{
			Name:    "Agnes",
			BaseURL: "https://api.agnes-ai.cn/v1",
		},
		// ↓ 用户可在下面继续添加更多内置服务商 ↓
		// {Name: "XX", BaseURL: "https://api.xxx.com"},
	}

	var toInsert []models.APIProfile
	for _, p := range builtinProfiles {
		if !existing[p.Name] {
			toInsert = append(toInsert, p)
		}
	}
	if len(toInsert) == 0 {
		return nil
	}
	return db.Create(&toInsert).Error
}
