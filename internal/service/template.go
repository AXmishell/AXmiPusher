package service

import (
	"errors"
	"fmt"
	"time"

	"messagepusher/internal/models"

	"gorm.io/gorm"
)

// TemplateService 模板与审核流服务。
type TemplateService struct {
	db *gorm.DB
}

// NewTemplateService 创建模板服务。
func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{db: db}
}

// CreateTemplate 创建模板(自动生成 v1 版本, 待审核)。
func (s *TemplateService) CreateTemplate(tenantID, createdBy uint64, code, name, content, channelType string) (*models.Template, error) {
	if content == "" {
		return nil, fmt.Errorf("模板内容不能为空")
	}
	tpl := models.Template{
		TenantID:       tenantID,
		Code:           code,
		Name:           name,
		Content:        content,
		ChannelType:    channelType,
		Status:         models.StatusPending,
		CurrentVersion: 1,
		CreatedBy:      createdBy,
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tpl).Error; err != nil {
			return err
		}
		return tx.Create(&models.TemplateVersion{
			TemplateID:   tpl.ID,
			Version:      1,
			Content:      content,
			ReviewStatus: models.StatusPending,
			CreatedBy:    createdBy,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &tpl, nil
}

// UpdateTemplate 更新模板内容(生成新版本, 模板回到待审核)。
func (s *TemplateService) UpdateTemplate(tenantID, id, updatedBy uint64, name, content, channelType string) error {
	var tpl models.Template
	if err := s.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&tpl).Error; err != nil {
		return err
	}
	// 已有待审核版本时禁止再改(需先审完)。
	var pending int64
	s.db.Model(&models.TemplateVersion{}).
		Where("template_id = ? AND review_status = ?", id, models.StatusPending).Count(&pending)
	if pending > 0 {
		return errors.New("存在待审核版本, 请等待审核完成后再修改")
	}

	newVersion := tpl.CurrentVersion + 1
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&models.TemplateVersion{
			TemplateID:   id,
			Version:      newVersion,
			Content:      content,
			ReviewStatus: models.StatusPending,
			CreatedBy:    updatedBy,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&tpl).Updates(map[string]any{
			"name":            name,
			"content":         content,
			"channel_type":    channelType,
			"status":          models.StatusPending,
			"current_version": newVersion,
		}).Error
	})
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

// --- 审核流(平台管理员) ---

// ReviewItem 待审核项(模板 + 最新版本)。
type ReviewItem struct {
	TemplateID     uint64    `json:"template_id"`
	TenantID       uint64    `json:"tenant_id"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	ChannelType    string    `json:"channel_type"`
	VersionID      uint64    `json:"version_id"`
	Version        int       `json:"version"`
	Content        string    `json:"content"`
	ReviewStatus   string    `json:"review_status"`
	CurrentVersion int       `json:"current_version"`
	CreatedAt      time.Time `json:"created_at"`
	TenantName     string    `json:"tenant_name"`
}

// ListReviews 待审核/已审核列表(全部租户)。
func (s *TemplateService) ListReviews(status string, page, size int) ([]ReviewItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	q := s.db.Table("template_versions v").
		Select("t.id as template_id, t.tenant_id, t.code, t.name, t.channel_type, v.id as version_id, v.version, v.content, v.review_status, t.current_version, v.created_at, u.nickname as tenant_name").
		Joins("JOIN templates t ON t.id = v.template_id").
		Joins("LEFT JOIN users u ON u.id = t.tenant_id")
	if status != "" {
		q = q.Where("v.review_status = ?", status)
	}
	var total int64
	q.Count(&total)
	var items []ReviewItem
	if err := q.Order("v.id DESC").Offset((page - 1) * size).Limit(size).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ApproveVersion 批准版本: 模板内容更新为该版本, 状态转 active。
func (s *TemplateService) ApproveVersion(templateID, versionID, reviewerID uint64, note string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var ver models.TemplateVersion
		if err := tx.First(&ver, "id = ? AND template_id = ? AND review_status = ?",
			versionID, templateID, models.StatusPending).Error; err != nil {
			return errors.New("待审核版本不存在")
		}
		if err := tx.Model(&ver).Updates(map[string]any{
			"review_status": "approved",
			"review_note":   note,
			"reviewed_by":   reviewerID,
			"reviewed_at":   time.Now(),
		}).Error; err != nil {
			return err
		}
		return tx.Model(&models.Template{}).Where("id = ?", templateID).Updates(map[string]any{
			"content":         ver.Content,
			"status":          models.StatusActive,
			"current_version": ver.Version,
		}).Error
	})
}

// RejectVersion 驳回版本。
func (s *TemplateService) RejectVersion(templateID, versionID, reviewerID uint64, note string) error {
	res := s.db.Model(&models.TemplateVersion{}).
		Where("id = ? AND template_id = ? AND review_status = ?", versionID, templateID, models.StatusPending).
		Updates(map[string]any{
			"review_status": "rejected",
			"review_note":   note,
			"reviewed_by":   reviewerID,
			"reviewed_at":   time.Now(),
		})
	if res.RowsAffected == 0 {
		return errors.New("待审核版本不存在")
	}
	// 模板状态: 若还有其它 pending 版本则保持 pending, 否则回退为 rejected。
	var pending int64
	s.db.Model(&models.TemplateVersion{}).
		Where("template_id = ? AND review_status = ?", templateID, models.StatusPending).Count(&pending)
	status := models.StatusPending
	if pending == 0 {
		status = "rejected"
	}
	return s.db.Model(&models.Template{}).Where("id = ?", templateID).Update("status", status).Error
}
