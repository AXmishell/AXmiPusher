package service

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"messagepusher/internal/models"

	"gorm.io/gorm"
)

// 支付业务错误。
var (
	ErrEpayNotConfigured = errors.New("易支付尚未配置, 请联系管理员")
	ErrOrderNotFound     = errors.New("订单不存在")
	ErrOrderClosed       = errors.New("订单已关闭")
	ErrPlanInvalid       = errors.New("套餐不存在或已下架")
)

// EpayConfig 易支付配置(来自平台设置)。
type EpayConfig struct {
	Gateway string `json:"gateway"`
	PID     string `json:"pid"`
	Key     string `json:"key"`
}

// PaymentService 易支付服务: 下单/验签/回调/订阅生效。
type PaymentService struct {
	db       *gorm.DB
	settings *SettingsService
	baseURL  string
}

// NewPaymentService 创建支付服务。
func NewPaymentService(db *gorm.DB, settings *SettingsService, baseURL string) *PaymentService {
	return &PaymentService{db: db, settings: settings, baseURL: baseURL}
}

// LoadConfig 读取易支付配置。
func (s *PaymentService) LoadConfig() (*EpayConfig, error) {
	var cfg EpayConfig
	if err := s.settings.GetJSON("epay", &cfg); err != nil {
		return nil, err
	}
	if cfg.Gateway == "" || cfg.PID == "" || cfg.Key == "" {
		return nil, ErrEpayNotConfigured
	}
	return &cfg, nil
}

// CreateOrder 创建支付订单, 返回订单与易支付跳转链接。
func (s *PaymentService) CreateOrder(tenantID, planID uint64, payType string) (*models.PaymentOrder, string, error) {
	cfg, err := s.LoadConfig()
	if err != nil {
		return nil, "", err
	}
	if payType == "" {
		payType = "alipay"
	}
	var plan models.Plan
	if err := s.db.First(&plan, "id = ? AND status = ?", planID, models.StatusActive).Error; err != nil {
		return nil, "", ErrPlanInvalid
	}

	order := &models.PaymentOrder{
		TenantID:   tenantID,
		PlanID:     planID,
		Type:       payType,
		OutTradeNo: genOutTradeNo(),
		Amount:     plan.Price,
		Status:     "pending",
		ExpiredAt:  timePtr(time.Now().Add(30 * time.Minute)),
	}
	if err := s.db.Create(order).Error; err != nil {
		return nil, "", err
	}

	payURL, err := s.BuildPayURL(cfg, order, plan)
	if err != nil {
		return nil, "", err
	}
	return order, payURL, nil
}

// BuildPayURL 构建易支付下单链接。
func (s *PaymentService) BuildPayURL(cfg *EpayConfig, order *models.PaymentOrder, plan models.Plan) (string, error) {
	notifyURL := strings.TrimRight(s.baseURL, "/") + "/api/v1/pay/notify"
	returnURL := strings.TrimRight(s.baseURL, "/") + "/api/v1/pay/return?out_trade_no=" + order.OutTradeNo

	params := map[string]string{
		"pid":          cfg.PID,
		"type":         order.Type,
		"out_trade_no": order.OutTradeNo,
		"notify_url":   notifyURL,
		"return_url":   returnURL,
		"name":         plan.Name,
		"money":        formatMoney(order.Amount),
	}
	sign := SignParams(params, cfg.Key)
	params["sign"] = sign

	// 构建 query(值需要 URL 编码, name 可能是中文)。
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(url.QueryEscape(params[k]))
	}
	return strings.TrimRight(cfg.Gateway, "/") + "/submit.php?" + sb.String(), nil
}

// SignParams 易支付标准签名: 参数按 key 升序拼接 k=v(& 连接), 跳过空值与 sign, 末尾追加 key=商户密钥, MD5 小写。
func SignParams(params map[string]string, merchantKey string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" || params[k] == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(params[k])
		sb.WriteString("&")
	}
	sb.WriteString("key=")
	sb.WriteString(merchantKey)
	sum := md5.Sum([]byte(sb.String()))
	return hex.EncodeToString(sum[:])
}

// VerifyNotify 校验易支付回调签名。
func (s *PaymentService) VerifyNotify(cfg *EpayConfig, params map[string]string) bool {
	sign := params["sign"]
	if sign == "" {
		return false
	}
	return SignParams(params, cfg.Key) == sign
}

// HandleNotify 处理支付回调: 验签 → 校验金额 → 订单置已支付 → 激活订阅。
// 幂等: 同一订单重复回调只生效一次。
// 返回是否成功(成功时上游停止重试)。
func (s *PaymentService) HandleNotify(params map[string]string) error {
	cfg, err := s.LoadConfig()
	if err != nil {
		return err
	}
	if params["pid"] != cfg.PID {
		return errors.New("pid 不匹配")
	}
	if !s.VerifyNotify(cfg, params) {
		return errors.New("签名校验失败")
	}
	if params["trade_status"] != "TRADE_SUCCESS" {
		return errors.New("交易未成功")
	}

	outTradeNo := params["out_trade_no"]
	money, _ := strconv.ParseFloat(params["money"], 64)

	var order models.PaymentOrder
	if err := s.db.Where("out_trade_no = ?", outTradeNo).First(&order).Error; err != nil {
		return errors.New("订单不存在")
	}
	// 幂等: 已支付直接返回成功。
	if order.Status == "paid" {
		return nil
	}
	if order.Status == "closed" {
		return ErrOrderClosed
	}
	// 金额一致性校验(防篡改)。
	if abs(order.Amount-money) > 0.001 {
		return errors.New("金额不匹配")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Model(&order).Updates(map[string]any{
			"status":       "paid",
			"epay_trade_no": params["trade_no"],
			"paid_at":      &now,
			"notify_data":  formatParams(params),
		}).Error; err != nil {
			return err
		}
		return activateSubscription(tx, order.TenantID, order.PlanID, now)
	})
}

// QueryOrder 查询订单(租户隔离)。
func (s *PaymentService) QueryOrder(tenantID, orderID uint64) (*models.PaymentOrder, error) {
	var order models.PaymentOrder
	if err := s.db.Where("id = ? AND tenant_id = ?", orderID, tenantID).First(&order).Error; err != nil {
		return nil, ErrOrderNotFound
	}
	return &order, nil
}

// CurrentSubscription 当前生效订阅。
func (s *PaymentService) CurrentSubscription(tenantID uint64) (*models.Subscription, *models.Plan, error) {
	var sub models.Subscription
	if err := s.db.Where("tenant_id = ? AND status = ?", tenantID, "active").First(&sub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var plan models.Plan
	if err := s.db.First(&plan, sub.PlanID).Error; err != nil {
		return nil, nil, err
	}
	return &sub, &plan, nil
}

// ListPlans 可用套餐。
func (s *PaymentService) ListPlans() ([]models.Plan, error) {
	var plans []models.Plan
	err := s.db.Where("status = ?", models.StatusActive).Order("sort_order ASC, id ASC").Find(&plans).Error
	return plans, err
}

// activateSubscription 激活订阅(替换旧订阅)。
func activateSubscription(tx *gorm.DB, tenantID, planID uint64, now time.Time) error {
	var plan models.Plan
	if err := tx.First(&plan, planID).Error; err != nil {
		return err
	}
	duration := plan.DurationDays
	if duration <= 0 {
		duration = 30
	}
	// 若已有未过期订阅且同套餐, 则顺延; 否则从现在开始。
	start := now
	var old models.Subscription
	if err := tx.Where("tenant_id = ? AND status = ?", tenantID, "active").First(&old).Error; err == nil {
		if old.PlanID == planID && old.EndAt.After(now) {
			start = old.EndAt
		}
	}
	end := start.Add(time.Duration(duration) * 24 * time.Hour)
	// 关闭旧订阅。
	tx.Model(&models.Subscription{}).Where("tenant_id = ?", tenantID).Update("status", "expired")
	sub := models.Subscription{TenantID: tenantID, PlanID: planID, StartAt: start, EndAt: end, Status: "active"}
	if err := tx.Create(&sub).Error; err != nil {
		return err
	}
	return tx.Model(&models.Tenant{}).Where("id = ?", tenantID).Update("plan_id", planID).Error
}

// genOutTradeNo 生成平台订单号: MP + 时间戳 + 随机。
func genOutTradeNo() string {
	return "MP" + strconv.FormatInt(time.Now().UnixMilli(), 10) + randomDigits(4)
}

func randomDigits(n int) string {
	const digits = "0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b)
}

func formatMoney(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}

func formatParams(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(params[k])
	}
	return sb.String()
}

func timePtr(t time.Time) *time.Time { return &t }

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
