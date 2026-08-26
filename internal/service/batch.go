package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"axmipusher/internal/models"

	"gorm.io/gorm"
)

// 批量任务状态。
const (
	BatchPending  = "pending"
	BatchRunning  = "running"
	BatchDone     = "done"
	BatchFailed   = "failed"
	BatchCanceled = "cancelled"
)

// batchChunkSize 每片处理的收件人数。
const batchChunkSize = 100

// BatchTaskConfig 批量任务参数(存入 Config JSON)。
type BatchTaskConfig struct {
	TemplateCode string      `json:"template_code"`
	Title        string      `json:"title"`
	Content      string      `json:"content"`
	Channel      string      `json:"channel"`
	Priority     string      `json:"priority"`
	Recipients   []Recipient `json:"recipients"`
}

// BatchService 批量任务服务。
type BatchService struct {
	db   *gorm.DB
	msgs *MessageService
	mu   sync.Mutex
	// running 正在执行的任务(防重复启动)。
	running map[uint64]bool
}

// NewBatchService 创建批量任务服务。
func NewBatchService(db *gorm.DB, msgs *MessageService) *BatchService {
	return &BatchService{
		db:      db,
		msgs:    msgs,
		running: make(map[uint64]bool),
	}
}

// Create 创建批量任务并异步启动处理。
func (s *BatchService) Create(_ context.Context, tenantID uint64, name string, cfg *BatchTaskConfig) (*models.BatchTask, error) {
	if len(cfg.Recipients) == 0 {
		return nil, errors.New("收件人不能为空")
	}
	if cfg.TemplateCode == "" && cfg.Content == "" {
		return nil, errors.New("template_code 与 content 至少提供一个")
	}
	if cfg.Channel == "" || cfg.Channel == "auto" {
		cfg.Channel = "webhook"
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	task := &models.BatchTask{
		TenantID:  tenantID,
		Name:      name,
		Status:    BatchPending,
		Total:     int64(len(cfg.Recipients)),
		Config:    string(raw),
	}
	if err := s.db.Create(task).Error; err != nil {
		return nil, err
	}
	// 后台任务必须用独立 ctx(请求 ctx 在 HTTP 结束时取消)。
	go s.process(context.Background(), task.ID)
	return task, nil
}

// List 任务列表。
func (s *BatchService) List(tenantID uint64, page, size int) ([]models.BatchTask, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	q := s.db.Model(&models.BatchTask{}).Where("tenant_id = ?", tenantID)
	var total int64
	q.Count(&total)
	var list []models.BatchTask
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Get 任务详情。
func (s *BatchService) Get(tenantID, taskID uint64) (*models.BatchTask, error) {
	var task models.BatchTask
	if err := s.db.Where("id = ? AND tenant_id = ?", taskID, tenantID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// Cancel 取消任务(仅 pending/running 可取消)。
func (s *BatchService) Cancel(tenantID, taskID uint64) error {
	res := s.db.Model(&models.BatchTask{}).
		Where("id = ? AND tenant_id = ? AND status IN ?", taskID, tenantID, []string{BatchPending, BatchRunning}).
		Update("status", BatchCanceled)
	if res.RowsAffected == 0 {
		return errors.New("任务不存在或已完成")
	}
	return nil
}

// process 后台处理任务: 分片解析模板, 逐片入队, 更新进度。
func (s *BatchService) process(ctx context.Context, taskID uint64) {
	s.mu.Lock()
	if s.running[taskID] {
		s.mu.Unlock()
		return
	}
	s.running[taskID] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.running, taskID)
		s.mu.Unlock()
	}()

	// 加锁防止并发处理同一任务。
	res := s.db.Model(&models.BatchTask{}).
		Where("id = ? AND status = ?", taskID, BatchPending).
		Update("status", BatchRunning)
	if res.Error != nil || res.RowsAffected == 0 {
		return
	}

	var task models.BatchTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return
	}
	var cfg BatchTaskConfig
	if err := json.Unmarshal([]byte(task.Config), &cfg); err != nil {
		s.db.Model(&task).Updates(map[string]any{"status": BatchFailed, "error": "任务配置解析失败"})
		return
	}

	// 模板预加载。
	var template *models.Template
	if cfg.TemplateCode != "" {
		var t models.Template
		if err := s.db.Where("tenant_id = ? AND code = ? AND status = ?",
			task.TenantID, cfg.TemplateCode, models.StatusActive).First(&t).Error; err == nil {
			template = &t
		} else {
			s.db.Model(&task).Updates(map[string]any{"status": BatchFailed, "error": "模板不存在或未审核通过"})
			return
		}
		if cfg.Title == "" {
			cfg.Title = t.Name
		}
	}

	success, failed := int64(0), int64(0)
	for _, r := range cfg.Recipients {
		// 检查取消。
		var cur models.BatchTask
		if err := s.db.First(&cur, taskID).Error; err != nil {
			return
		}
		if cur.Status == BatchCanceled {
			s.db.Model(&task).Updates(map[string]any{"status": BatchCanceled})
			return
		}

		content := cfg.Content
		if template != nil {
			content = renderTemplate(template.Content, r.Params)
		}
		if _, err := s.msgs.EnqueueOne(ctx, task.TenantID, cfg.Channel, cfg.Title, content, r.Target, "", cfg.Priority); err != nil {
			failed++
		} else {
			success++
		}
		// 分片更新进度。
		if (success+failed)%batchChunkSize == 0 {
			s.db.Model(&task).Updates(map[string]any{
				"success": success, "failed": failed, "status": BatchRunning,
			})
		}
	}

	status := BatchDone
	errMsg := ""
	if failed > 0 && success == 0 {
		status = BatchFailed
		errMsg = "全部入队失败"
	}
	s.db.Model(&task).Updates(map[string]any{
		"success": success, "failed": failed, "status": status, "error": errMsg,
	})
}

// IsRunning 任务是否在执行中(观测用)。
func (s *BatchService) IsRunning(taskID uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running[taskID]
}
