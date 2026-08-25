package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"messagepusher/internal/models"
	"messagepusher/internal/queue"
	"messagepusher/internal/store"

	"gorm.io/gorm"
)

// 业务错误。
var (
	ErrRateLimited     = errors.New("请求频率超限")
	ErrTemplateMissing = errors.New("模板不存在或不可用")
	ErrChannelInvalid  = errors.New("渠道不可用")
	ErrEmptyRecipients = errors.New("收件人不能为空")
	ErrDuplicate       = errors.New("重复请求")
)

// Recipient 单个收件人。
type Recipient struct {
	Target string            `json:"target"`
	Params map[string]string `json:"params"`
}

// SendRequest 发送请求。
type SendRequest struct {
	RequestID    string      `json:"request_id"`
	TemplateCode string      `json:"template_code"`
	Title        string      `json:"title"`
	Content      string      `json:"content"`
	Channel      string      `json:"channel"` // auto | webhook | email | apns ...
	Priority     string      `json:"priority"`
	Recipients   []Recipient `json:"recipients"`
}

// SendResult 发送结果。
type SendResult struct {
	MessageID uint64 `json:"message_id"`
	Duplicate bool   `json:"duplicate"`
	Count     int    `json:"count"`
}

// MessageService 消息发送服务(受理链路核心)。
type MessageService struct {
	db          *gorm.DB
	store       store.MessageStore
	queue       queue.Queue
	limiter     RateLimiter
	rateEnabled bool
	// redis 幂等加速缓存(可选, DB 兜底)。
	redis redisCmdable
	// HasChannel 校验渠道是否已注册(由容器注入)。
	HasChannel func(channel string) bool
	// IsAvailable 校验渠道可用于 auto 路由(已注册且未熔断, 由容器注入)。
	IsAvailable func(tenantID uint64, channel string) bool
}

// NewMessageService 创建消息服务。
func NewMessageService(db *gorm.DB, st store.MessageStore, q queue.Queue, limiter RateLimiter, rateEnabled bool) *MessageService {
	return &MessageService{
		db:          db,
		store:       st,
		queue:       q,
		limiter:     limiter,
		rateEnabled: rateEnabled,
	}
}

// SetRedisCache 注入 Redis 客户端(幂等加速)。
func (s *MessageService) SetRedisCache(c redisCmdable) {
	s.redis = c
}

// SetLimiter 替换限流器(Redis 接线时调用)。
func (s *MessageService) SetLimiter(l RateLimiter) {
	s.limiter = l
}

// SetQueue 替换队列(Reinit 保留运行中队列时调用)。
func (s *MessageService) SetQueue(q queue.Queue) {
	s.queue = q
}

// Send 受理一条发送请求:
// ① 幂等检查 → ② 限流 → ③ 模板渲染 → ④ 逐收件人建记录并入队。
func (s *MessageService) Send(ctx context.Context, tenantID uint64, req *SendRequest) (*SendResult, error) {
	if len(req.Recipients) == 0 {
		return nil, ErrEmptyRecipients
	}

	// ① 幂等: 同租户 + 同 request_id 只受理一次。
	var firstMessageID uint64
	if req.RequestID != "" {
		// 快路径: Redis 缓存(命中即返回原 message_id)。
		if s.redis != nil {
			ctx2, cancel := context.WithTimeout(ctx, time.Second)
			mid, err := s.redis.Eval(ctx2, redisGetScript, []string{"idem:" + idemKey(tenantID, req.RequestID)}).Int64()
			cancel()
			if err == nil && mid > 0 {
				return &SendResult{MessageID: uint64(mid), Duplicate: true}, nil
			}
		}
		var rec models.IdempotencyRecord
		err := s.db.Where("tenant_id = ? AND request_id = ?", tenantID, req.RequestID).First(&rec).Error
		if err == nil {
			return &SendResult{MessageID: rec.MessageID, Duplicate: true}, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// ② 限流。
	if s.rateEnabled && !s.limiter.Allow(tenantID) {
		return nil, ErrRateLimited
	}

	// ③ 模板解析。
	var template *models.Template
	if req.TemplateCode != "" {
		var t models.Template
		if err := s.db.Where("tenant_id = ? AND code = ? AND status = ?",
			tenantID, req.TemplateCode, models.StatusActive).First(&t).Error; err != nil {
			return nil, ErrTemplateMissing
		}
		template = &t
		if req.Title == "" {
			req.Title = t.Name
		}
	}

	// 渠道解析: 显式指定 > 模板渠道 > 默认 webhook。
	channel := req.Channel
	if channel == "" || channel == "auto" {
		channel = "webhook"
		if template != nil && template.ChannelType != "" {
			channel = template.ChannelType
		}
		// 熔断降级: 优先渠道若熔断, 按优先级选第一个可用渠道。
		if s.IsAvailable != nil && !s.IsAvailable(tenantID, channel) {
			channel = s.pickFallbackChannel(tenantID, channel)
		}
	}
	// 快速失败: 渠道未注册/未配置。
	if s.HasChannel != nil && !s.HasChannel(channel) {
		return nil, fmt.Errorf("%w: %s", ErrChannelInvalid, channel)
	}

	// ④ 逐收件人创建消息记录并入队。
	count := 0
	for _, r := range req.Recipients {
		if strings.TrimSpace(r.Target) == "" {
			continue
		}
		content := req.Content
		if template != nil {
			content = renderTemplate(template.Content, r.Params)
		}
		mid, err := s.EnqueueOne(ctx, tenantID, channel, req.Title, content, r.Target, req.RequestID, req.Priority)
		if err != nil {
			// 入队失败: 标记消息失败, 但不中断整体(幂等记录也不写入, 允许重试)。
			continue
		}
		if firstMessageID == 0 {
			firstMessageID = mid
		}
		count++
	}

	if count == 0 {
		return nil, fmt.Errorf("没有成功入队的消息")
	}

	// 幂等记录: 保存整个请求对应的首个 message_id。
	if req.RequestID != "" {
		rec := models.IdempotencyRecord{TenantID: tenantID, RequestID: req.RequestID, MessageID: firstMessageID}
		if err := s.db.Create(&rec).Error; err != nil {
			return nil, err
		}
		// 回填 Redis 缓存(带 TTL, 加速后续重复请求判定)。
		if s.redis != nil {
			ctx2, cancel := context.WithTimeout(ctx, time.Second)
			_, _ = s.redis.Eval(ctx2, redisSetScript, []string{"idem:" + idemKey(tenantID, req.RequestID)}, firstMessageID, 86400).Result()
			cancel()
		}
	}
	return &SendResult{MessageID: firstMessageID, Count: count}, nil
}

// redisGetScript 读取幂等缓存, 不存在返回 0。
const redisGetScript = `return redis.call('GET', KEYS[1]) or 0`

// redisSetScript 写入幂等缓存(带 TTL 秒)。
const redisSetScript = `redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[2]); return 1`

func idemKey(tenantID uint64, requestID string) string {
	return strconv.FormatUint(tenantID, 10) + ":" + requestID
}

// EnqueueOne 为单个收件人创建消息记录并入队(单发与批量共用)。
func (s *MessageService) EnqueueOne(ctx context.Context, tenantID uint64, channel, title, content, recipient, requestID, priority string) (uint64, error) {
	now := time.Now()
	mid, err := s.store.SaveMessage(ctx, &store.Message{
		TenantID:  tenantID,
		RequestID: requestID,
		Channel:   channel,
		Title:     title,
		Content:   content,
		Recipient: recipient,
		Status:    models.MsgPending,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return 0, fmt.Errorf("保存消息记录失败: %w", err)
	}
	if err := s.store.SaveEvent(ctx, &store.MessageEvent{
		MessageID: mid, EventType: models.EventCreated, CreatedAt: now,
	}); err != nil {
		return 0, err
	}
	if err := s.queue.Publish(ctx, &queue.TaskMessage{
		MessageID: mid,
		TenantID:  tenantID,
		RequestID: requestID,
		Channel:   channel,
		Title:     title,
		Content:   content,
		Recipient: recipient,
		Priority:  priority,
		CreatedAt: now,
	}); err != nil {
		s.store.UpdateStatus(ctx, mid, models.MsgFailed, "入队失败: "+err.Error())
		return 0, err
	}
	return mid, nil
}

// GetMessage 查询单条消息(校验租户隔离)。
func (s *MessageService) GetMessage(ctx context.Context, tenantID, messageID uint64) (*store.Message, error) {
	m, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if m.TenantID != tenantID {
		return nil, gorm.ErrRecordNotFound
	}
	return m, nil
}

// QueryMessages 分页查询消息(租户隔离)。
func (s *MessageService) QueryMessages(ctx context.Context, tenantID uint64, f store.MessageFilter, page, size int) ([]*store.Message, int64, error) {
	return s.store.QueryMessages(ctx, tenantID, f, page, size)
}

var templateVarRe = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

// fallbackOrder 熔断降级优先级(优先渠道之后的备选顺序)。
var fallbackOrder = []string{"webhook", "email", "apns", "fcm", "inapp"}

// pickFallbackChannel 在优先渠道熔断时, 按优先级选择第一个可用的降级渠道。
func (s *MessageService) pickFallbackChannel(tenantID uint64, preferred string) string {
	for _, ch := range fallbackOrder {
		if ch == preferred {
			continue
		}
		if s.IsAvailable != nil {
			// 有熔断器: 只选真正可用(未熔断)的渠道, 熔断的跳过。
			if s.IsAvailable(tenantID, ch) {
				return ch
			}
			continue
		}
		if s.HasChannel != nil && s.HasChannel(ch) {
			return ch
		}
	}
	return preferred
}

// renderTemplate 将模板中的 {{var}} 替换为参数值。
func renderTemplate(tpl string, params map[string]string) string {
	return templateVarRe.ReplaceAllStringFunc(tpl, func(m string) string {
		key := strings.TrimSpace(strings.Trim(m, "{}"))
		if v, ok := params[key]; ok {
			return v
		}
		return ""
	})
}
