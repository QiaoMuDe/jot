package services

import (
	"errors"
	"jot/internal/models"

	"gitee.com/MM-Q/fastlog"
	"gorm.io/gorm"
)

// ProfileService 封装 API 配置预设相关的业务逻辑操作
type ProfileService struct {
	db     *gorm.DB
	logger *fastlog.Logger
}

// NewProfileService 创建一个新的 ProfileService 实例
func NewProfileService(db *gorm.DB, logger *fastlog.Logger) *ProfileService {
	return &ProfileService{db: db, logger: logger}
}

// ListProfiles 获取所有预设，按创建时间降序
func (p *ProfileService) ListProfiles() []models.APIProfile {
	var profiles []models.APIProfile
	p.db.Order("created_at desc").Find(&profiles)
	for i := range profiles {
		profiles[i].APIKey = DecodeB64(profiles[i].APIKey)
	}
	return profiles
}

// CreateProfile 创建预设
func (p *ProfileService) CreateProfile(name, provider, baseURL, apiKey string, isDefault ...bool) models.APIProfile {
	profile := models.APIProfile{
		Name:     name,
		Provider: provider,
		BaseURL:  baseURL,
		APIKey:   EncodeB64(apiKey),
	}
	if len(isDefault) > 0 && isDefault[0] {
		profile.IsDefault = true
	}
	p.db.Create(&profile)
	return profile
}

// UpdateProfile 更新预设
func (p *ProfileService) UpdateProfile(id uint, name, provider, baseURL, apiKey string) error {
	err := p.db.Model(&models.APIProfile{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":     name,
		"provider": provider,
		"base_url": baseURL,
		"api_key":  EncodeB64(apiKey),
	}).Error
	if err != nil {
		p.logger.Errorw("ProfileService.UpdateProfile 失败", fastlog.Error(err))
	}
	return err
}

// SetActive 将指定预设标记为激活（不清除模型，仅改标记）
func (p *ProfileService) SetActive(id uint) error {
	p.db.Model(&models.APIProfile{}).Where("1 = 1").Update("is_active", false)
	err := p.db.Model(&models.APIProfile{}).Where("id = ?", id).Update("is_active", true).Error
	if err != nil {
		p.logger.Errorw("ProfileService.SetActive 失败", fastlog.Error(err))
	}
	return err
}

// DeleteProfile 删除预设（默认配置不可删除）
func (p *ProfileService) DeleteProfile(id uint) error {
	var profile models.APIProfile
	if err := p.db.First(&profile, id).Error; err != nil {
		p.logger.Errorw("ProfileService.DeleteProfile 失败", fastlog.Error(err))
		return err
	}
	if profile.IsDefault {
		return errors.New("默认配置不可删除")
	}
	err := p.db.Delete(&models.APIProfile{}, id).Error
	if err != nil {
		p.logger.Errorw("ProfileService.DeleteProfile 失败", fastlog.Error(err))
	}
	return err
}

// SwitchProfile 切换预设：将指定预设的值写入当前配置（settings 表），并标记为激活
func (p *ProfileService) SwitchProfile(id uint) error {
	var profile models.APIProfile
	if err := p.db.First(&profile, id).Error; err != nil {
		p.logger.Errorw("ProfileService.SwitchProfile 失败", fastlog.Error(err))
		return err
	}
	// 清除所有预设的激活标记
	p.db.Model(&models.APIProfile{}).Where("1 = 1").Update("is_active", false)
	// 标记当前预设为激活
	p.db.Model(&models.APIProfile{}).Where("id = ?", id).Update("is_active", true)
	// 写入 settings 表
	svc := NewSettingService(p.db)
	if err := svc.Set("ai_provider", profile.Provider); err != nil {
		p.logger.Errorw("ProfileService.SwitchProfile 失败", fastlog.Error(err))
		return err
	}
	if err := svc.Set("ai_base_url", profile.BaseURL); err != nil {
		p.logger.Errorw("ProfileService.SwitchProfile 失败", fastlog.Error(err))
		return err
	}
	if err := svc.Set("ai_api_key", profile.APIKey); err != nil {
		p.logger.Errorw("ProfileService.SwitchProfile 失败", fastlog.Error(err))
		return err
	}
	// 清除模型，由用户在切换后重新选择
	if err := svc.Set("ai_model", ""); err != nil {
		p.logger.Errorw("ProfileService.SwitchProfile 失败", fastlog.Error(err))
		return err
	}
	// 清除所有会话配置中的模型（切换预设后旧模型不可用）
	p.db.Model(&models.AISessionConfig{}).Where("1 = 1").Update("model_name", "")
	return nil
}
