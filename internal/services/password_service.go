package services

import (
	"strings"

	"jot/internal/models"

	"gitee.com/MM-Q/fastlog"
	"gorm.io/gorm"
)

// PasswordService 密码记录服务
type PasswordService struct {
	db     *gorm.DB
	logger *fastlog.Logger
}

func NewPasswordService(db *gorm.DB, logger *fastlog.Logger) *PasswordService {
	return &PasswordService{db: db, logger: logger}
}

// PasswordListItem 列表项 DTO：仅含名称、用户名、URL，不含密码与备注
type PasswordListItem struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	URL      string `json:"url"`
}

// passwordOrder 统一排序：更新时间倒序（新增/编辑的记录浮到最前），追加 id DESC 保证顺序稳定
const passwordOrder = "updated_at DESC, id DESC"

// Create 创建密码记录，password 字段编码后存储（无前缀明文兼容由 EncodeB64/DecodeB64 处理）
func (s *PasswordService) Create(name, username, password, url, note string) (*models.PasswordRecord, error) {
	rec := &models.PasswordRecord{
		Name:     name,
		Username: username,
		Password: EncodeB64(password),
		URL:      url,
		Note:     note,
	}
	if err := s.db.Create(rec).Error; err != nil {
		s.logger.Errorw("PasswordService.Create 失败", fastlog.Error(err))
		return nil, err
	}
	return rec, nil
}

// List 分页返回密码记录（按更新时间倒序），不解码密码字段
func (s *PasswordService) List(page, pageSize int) ([]PasswordListItem, int64, error) {
	var total int64
	if err := s.db.Model(&models.PasswordRecord{}).Count(&total).Error; err != nil {
		s.logger.Errorw("PasswordService.List Count 失败", fastlog.Error(err))
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	var recs []models.PasswordRecord
	if err := s.db.Order(passwordOrder).Offset(offset).Limit(pageSize).Find(&recs).Error; err != nil {
		s.logger.Errorw("PasswordService.List 失败", fastlog.Error(err))
		return nil, 0, err
	}
	return toPasswordListItems(recs), total, nil
}

// Search 分页搜索名称、用户名、URL、备注字段，不解码密码字段。
// keyword trim 后为空时等价于 List。
func (s *PasswordService) Search(keyword string, page, pageSize int) ([]PasswordListItem, int64, error) {
	keyword = strings.TrimSpace(keyword)
	db := s.db.Model(&models.PasswordRecord{})
	if keyword != "" {
		like := "%" + escapeLike(keyword) + "%"
		db = db.Where("name LIKE ? ESCAPE '\\' OR username LIKE ? ESCAPE '\\' OR url LIKE ? ESCAPE '\\' OR note LIKE ? ESCAPE '\\'", like, like, like, like)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		s.logger.Errorw("PasswordService.Search Count 失败", fastlog.Error(err))
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	var recs []models.PasswordRecord
	if err := db.Order(passwordOrder).Offset(offset).Limit(pageSize).Find(&recs).Error; err != nil {
		s.logger.Errorw("PasswordService.Search 失败", fastlog.Error(err))
		return nil, 0, err
	}
	return toPasswordListItems(recs), total, nil
}

// GetPasswordRecord 根据 ID 查询单条记录，password 字段解码后返回明文。
// 存量无前缀明文由 DecodeB64 原样返回。
func (s *PasswordService) GetPasswordRecord(id uint) (*models.PasswordRecord, error) {
	var rec models.PasswordRecord
	if err := s.db.First(&rec, id).Error; err != nil {
		s.logger.Errorw("PasswordService.GetPasswordRecord 失败", fastlog.Uint("id", id), fastlog.Error(err))
		return nil, err
	}
	rec.Password = DecodeB64(rec.Password)
	return &rec, nil
}

// Update 更新密码记录，password 字段重新编码后存储；password 为空时保留原密码
func (s *PasswordService) Update(id uint, name, username, password, url, note string) (*models.PasswordRecord, error) {
	var rec models.PasswordRecord
	if err := s.db.First(&rec, id).Error; err != nil {
		s.logger.Errorw("PasswordService.Update 查询失败", fastlog.Uint("id", id), fastlog.Error(err))
		return nil, err
	}
	rec.Name = name
	rec.Username = username
	rec.URL = url
	rec.Note = note
	if password != "" {
		rec.Password = EncodeB64(password)
	}
	if err := s.db.Save(&rec).Error; err != nil {
		s.logger.Errorw("PasswordService.Update 失败", fastlog.Uint("id", id), fastlog.Error(err))
		return nil, err
	}
	return &rec, nil
}

// Delete 软删除密码记录
func (s *PasswordService) Delete(id uint) error {
	if err := s.db.Delete(&models.PasswordRecord{}, id).Error; err != nil {
		s.logger.Errorw("PasswordService.Delete 失败", fastlog.Uint("id", id), fastlog.Error(err))
		return err
	}
	return nil
}

// BatchDelete 根据 ID 列表批量软删除密码记录
func (s *PasswordService) BatchDelete(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	result := s.db.Where("id IN ?", ids).Delete(&models.PasswordRecord{})
	if result.Error != nil {
		s.logger.Errorw("PasswordService.BatchDelete 失败", fastlog.Int("count", len(ids)), fastlog.Error(result.Error))
		return result.Error
	}
	return nil
}

// Count 统计密码记录总数（不含已软删除）
func (s *PasswordService) Count() (int64, error) {
	var count int64
	err := s.db.Model(&models.PasswordRecord{}).Count(&count).Error
	return count, err
}

// toPasswordListItems 将模型记录转换为列表项 DTO（不携带密码）
func toPasswordListItems(recs []models.PasswordRecord) []PasswordListItem {
	items := make([]PasswordListItem, len(recs))
	for i, r := range recs {
		items[i] = PasswordListItem{
			ID:       r.ID,
			Name:     r.Name,
			Username: r.Username,
			URL:      r.URL,
		}
	}
	return items
}
