package service

import (
	"fmt"

	"axmipusher/internal/models"

	"gorm.io/gorm"
)

// TemplateService 模板服务。
type TemplateService struct {
	db *gorm.DB
}

// NewTemplateService 创建模板服务。
func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{db: db}
}

// CreateTemplate 创建模板(审核已移除, 创建即生效)。
func (s *TemplateService) CreateTemplate(tenantID, createdBy uint64, code, name, content, channelType string) (*models.Template, error) {
	if content == "" {
		return nil, fmt.Errorf("模板内容不能为空")
	}
	tpl := models.Template{
		TenantID:    tenantID,
		Code:        code,
		Name:        name,
		Content:     content,
		ChannelType: channelType,
		Status:      models.StatusActive,
		CreatedBy:   createdBy,
	}
	if err := s.db.Create(&tpl).Error; err != nil {
		return nil, err
	}
	return &tpl, nil
}

// UpdateTemplate 更新模板内容(审核已移除, 更新后直接生效)。
func (s *TemplateService) UpdateTemplate(tenantID, id uint64, name, content, channelType string) error {
	var tpl models.Template
	if err := s.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&tpl).Error; err != nil {
		return err
	}
	// 更新后 status 显式置 active(不再有"待审核"状态)。
	return s.db.Model(&tpl).Updates(map[string]any{
		"name":         name,
		"content":      content,
		"channel_type": channelType,
		"status":       models.StatusActive,
	}).Error
}

// ListTemplates 租户模板列表。
func (s *TemplateService) ListTemplates(tenantID uint64, page, size int) ([]models.Template, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	q := s.db.Model(&models.Template{}).Where("tenant_id = ?", tenantID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.Template
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetTemplate 查询单个模板。
func (s *TemplateService) GetTemplate(tenantID, id uint64) (*models.Template, error) {
	var tpl models.Template
	if err := s.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&tpl).Error; err != nil {
		return nil, err
	}
	return &tpl, nil
}

// DeleteTemplate 删除模板。
func (s *TemplateService) DeleteTemplate(tenantID, id uint64) error {
	return s.db.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&models.Template{}).Error
}
