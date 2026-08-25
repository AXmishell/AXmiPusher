package service

import (
	"encoding/json"
	"fmt"
	"time"

	"messagepusher/internal/models"

	"gorm.io/gorm"
)

// SettingsService 平台运行时设置(DB 键值存储)。
type SettingsService struct {
	db *gorm.DB
}

// NewSettingsService 创建设置服务。
func NewSettingsService(db *gorm.DB) *SettingsService {
	return &SettingsService{db: db}
}

// Get 读取设置, 不存在返回默认值(JSON 字符串)。
func (s *SettingsService) Get(key, def string) (string, error) {
	var st models.Setting
	err := s.db.First(&st, "key = ?", key).Error
	if err == nil {
		return st.Value, nil
	}
	if err == gorm.ErrRecordNotFound {
		return def, nil
	}
	return "", err
}

// GetJSON 读取 JSON 设置到目标结构。
func (s *SettingsService) GetJSON(key string, target interface{}) error {
	raw, err := s.Get(key, "")
	if err != nil {
		return err
	}
	if raw == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), target)
}

// Set 写入设置。
func (s *SettingsService) Set(key, value string, updatedBy uint64) error {
	st := models.Setting{Key: key, Value: value, UpdatedBy: updatedBy, UpdatedAt: time.Now()}
	// 不存在则插入, 存在则更新。
	return s.db.Save(&st).Error
}

// SetJSON 序列化并写入 JSON 设置。
func (s *SettingsService) SetJSON(key string, value interface{}, updatedBy uint64) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化设置失败: %w", err)
	}
	return s.Set(key, string(data), updatedBy)
}

// List 列出全部设置(供管理后台展示, 隐藏敏感字段由 handler 负责)。
func (s *SettingsService) List() ([]models.Setting, error) {
	var list []models.Setting
	err := s.db.Order("key ASC").Find(&list).Error
	return list, err
}
