package services

import (
	"gorm.io/gorm"
	"jot/internal/models"
)

// SettingService 封装配置相关的业务逻辑操作
type SettingService struct {
	db *gorm.DB
}

// NewSettingService 创建一个新的 SettingService 实例
func NewSettingService(db *gorm.DB) *SettingService {
	return &SettingService{db: db}
}

// Get 获取指定 key 的配置值，不存在时返回空字符串
func (s *SettingService) Get(key string) string {
	var setting models.Setting
	err := s.db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		return ""
	}
	return setting.Value
}

// Set 设置指定 key 的配置值，存在则更新，不存在则创建
func (s *SettingService) Set(key, value string) error {
	var setting models.Setting
	result := s.db.Where("key = ?", key).First(&setting)
	if result.Error != nil {
		// 不存在则创建
		setting = models.Setting{Key: key, Value: value}
		return s.db.Create(&setting).Error
	}
	// 存在则更新
	return s.db.Model(&setting).Update("value", value).Error
}

// DeleteAll 删除所有配置项，用于恢复出厂设置
func (s *SettingService) DeleteAll() error {
	return s.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Setting{}).Error
}
