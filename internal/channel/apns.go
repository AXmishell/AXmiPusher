package channel

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"axmipusher/internal/models"
	"axmipusher/internal/queue"

	"gorm.io/gorm"
)

// APNsConfig APNs 令牌认证配置(租户自带凭证)。
type APNsConfig struct {
	TeamID   string `json:"team_id"`   // 10 位 Team ID
	KeyID    string `json:"key_id"`    // 10 位 Key ID
	BundleID string `json:"bundle_id"` // app bundle id
	KeyP8    string `json:"key_p8"`    // .p8 私钥内容(PEM)
	Sandbox  bool   `json:"sandbox"`   // true=开发环境
}

// APNsSender APNs 渠道(HTTP/2, token-based auth)。
type APNsSender struct {
	db     *gorm.DB
	client *http.Client // 可注入用于测试
}

// NewAPNsSender 创建 APNs 渠道。
func NewAPNsSender(db *gorm.DB) *APNsSender {
	return &APNsSender{
		db:     db,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Type 渠道类型。
func (s *APNsSender) Type() string { return "apns" }

// apnsPayload APNs 通知载荷。
type apnsPayload struct {
	Aps aps `json:"aps"`
}

type aps struct {
	Alert alert `json:"alert"`
	Sound string `json:"sound,omitempty"`
	Badge int    `json:"badge,omitempty"`
}

type alert struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Send 发送 APNs 推送。Recipient 为设备 device token。
func (s *APNsSender) Send(ctx context.Context, msg *queue.TaskMessage) error {
	cfg, err := s.resolveConfig(ctx, msg.TenantID)
	if err != nil {
		return err
	}
	if msg.Recipient == "" {
		return errors.New("APNs 设备 token 为空")
	}

	jwt, err := s.buildToken(cfg)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(apnsPayload{Aps: aps{Alert: alert{Title: msg.Title, Body: msg.Content}}})

	endpoint := "https://api.push.apple.com"
	if cfg.Sandbox {
		endpoint = "https://api.sandbox.push.apple.com"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		endpoint+"/3/device/"+msg.Recipient, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "bearer "+jwt)
	req.Header.Set("apns-topic", cfg.BundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("APNs 请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	switch resp.StatusCode {
	case 200:
		return nil
	case 400, 403, 404, 410:
		// 设备 token 无效/过期类错误: 直接失败不再重试。
		return fmt.Errorf("APNs 拒绝(%d): %s", resp.StatusCode, string(body))
	default:
		return fmt.Errorf("APNs 发送失败(%d): %s", resp.StatusCode, string(body))
	}
}

// resolveConfig 读取租户 APNs 配置。
func (s *APNsSender) resolveConfig(ctx context.Context, tenantID uint64) (APNsConfig, error) {
	var ch models.Channel
	if err := s.db.WithContext(ctx).Where("tenant_id = ? AND type = ? AND status = ?",
		tenantID, "apns", models.StatusActive).First(&ch).Error; err != nil {
		return APNsConfig{}, errors.New("APNs 渠道未配置: 请先在渠道设置中填写 Team ID / Key ID / Bundle ID / .p8 密钥")
	}
	var cfg APNsConfig
	if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
		return APNsConfig{}, fmt.Errorf("APNs 配置格式错误: %w", err)
	}
	if cfg.TeamID == "" || cfg.KeyID == "" || cfg.BundleID == "" || cfg.KeyP8 == "" {
		return APNsConfig{}, errors.New("APNs 配置不完整(team_id/key_id/bundle_id/key_p8 必填)")
	}
	return cfg, nil
}

// buildToken 生成 ES256 JWT(APNs token auth)。
func (s *APNsSender) buildToken(cfg APNsConfig) (string, error) {
	key, err := parseECKey(cfg.KeyP8)
	if err != nil {
		return "", fmt.Errorf("解析 .p8 密钥失败: %w", err)
	}
	now := time.Now()
	header := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"alg":"ES256","kid":"%s"}`, cfg.KeyID)))
	claims := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(
		`{"iss":"%s","iat":%d,"exp":%d}`, cfg.TeamID, now.Unix(), now.Add(time.Hour).Unix())))
	signingInput := header + "." + claims

	hash := sha256.Sum256([]byte(signingInput))
	r, sInt, err := ecdsa.Sign(rand.Reader, key, hash[:])
	if err != nil {
		return "", err
	}
	// ES256 签名: r || s 各 32 字节。
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	sInt.FillBytes(sig[32:])
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// parseECKey 解析 P8 中的 EC 私钥。
func parseECKey(pemStr string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("无效的 PEM 数据")
	}
	var key *ecdsa.PrivateKey
	var err error
	if x509.IsEncryptedPEMBlock(block) {
		return nil, errors.New("不支持加密的 PEM")
	}
	// 尝试 PKCS8, 再尝试 SEC1。
	if k, e := x509.ParsePKCS8PrivateKey(block.Bytes); e == nil {
		if ecdsaKey, ok := k.(*ecdsa.PrivateKey); ok {
			key = ecdsaKey
			err = nil
		}
	}
	if key == nil {
		if k, e := x509.ParseECPrivateKey(block.Bytes); e == nil {
			key = k
			err = nil
		} else {
			err = e
		}
	}
	if err != nil {
		return nil, err
	}
	return key, nil
}
