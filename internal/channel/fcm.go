package channel

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
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

	"messagepusher/internal/models"
	"messagepusher/internal/queue"

	"gorm.io/gorm"
)

// FCMConfig FCM v1 配置(Google 服务账号 JSON 中的关键字段)。
type FCMConfig struct {
	ProjectID    string `json:"project_id"`
	ClientEmail  string `json:"client_email"`
	PrivateKey   string `json:"private_key"` // RSA 私钥 PEM
	TokenURL     string `json:"token_url"`   // 默认 https://oauth2.googleapis.com/token
	Endpoint     string `json:"endpoint"`    // 默认 https://fcm.googleapis.com/v1
	DeviceTokens string `json:"device_tokens"` // 逗号分隔, 简化单 token 场景取第一个
}

// FCMSender FCM v1 渠道(服务账号 OAuth2 + messages:send)。
type FCMSender struct {
	db     *gorm.DB
	client *http.Client
}

// NewFCMSender 创建 FCM 渠道。
func NewFCMSender(db *gorm.DB) *FCMSender {
	return &FCMSender{
		db:     db,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Type 渠道类型。
func (s *FCMSender) Type() string { return "fcm" }

// fcmMessage FCM v1 消息体。
type fcmMessage struct {
	Message fcmMsg `json:"message"`
}

type fcmMsg struct {
	Token        string        `json:"token"`
	Notification fcmNotif      `json:"notification"`
	Android      fcmAndroid    `json:"android,omitempty"`
}

type fcmNotif struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type fcmAndroid struct {
	Priority string `json:"priority,omitempty"`
}

// Send 发送 FCM 推送。Recipient 为设备 registration token。
func (s *FCMSender) Send(ctx context.Context, msg *queue.TaskMessage) error {
	cfg, err := s.resolveConfig(ctx, msg.TenantID)
	if err != nil {
		return err
	}
	if msg.Recipient == "" {
		return errors.New("FCM 设备 token 为空")
	}

	accessToken, err := s.fetchAccessToken(ctx, cfg)
	if err != nil {
		return err
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://fcm.googleapis.com/v1"
	}
	body, _ := json.Marshal(fcmMessage{Message: fcmMsg{
		Token: msg.Recipient,
		Notification: fcmNotif{Title: msg.Title, Body: msg.Content},
		Android:      fcmAndroid{Priority: "high"},
	}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/projects/%s/messages:send", endpoint, cfg.ProjectID), strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("FCM 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != 200 {
		return fmt.Errorf("FCM 发送失败(%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// resolveConfig 读取租户 FCM 配置。
func (s *FCMSender) resolveConfig(ctx context.Context, tenantID uint64) (FCMConfig, error) {
	var ch models.Channel
	if err := s.db.WithContext(ctx).Where("tenant_id = ? AND type = ? AND status = ?",
		tenantID, "fcm", models.StatusActive).First(&ch).Error; err != nil {
		return FCMConfig{}, errors.New("FCM 渠道未配置: 请先在渠道设置中填写服务账号信息")
	}
	var cfg FCMConfig
	if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
		return FCMConfig{}, fmt.Errorf("FCM 配置格式错误: %w", err)
	}
	if cfg.ProjectID == "" || cfg.ClientEmail == "" || cfg.PrivateKey == "" {
		return FCMConfig{}, errors.New("FCM 配置不完整(project_id/client_email/private_key 必填)")
	}
	return cfg, nil
}

// fetchAccessToken 用服务账号 JWT 换取 OAuth2 access token。
func (s *FCMSender) fetchAccessToken(ctx context.Context, cfg FCMConfig) (string, error) {
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = "https://oauth2.googleapis.com/token"
	}
	key, err := parseRSAKey(cfg.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("解析私钥失败: %w", err)
	}
	now := time.Now()
	claims := fmt.Sprintf(`{"iss":"%s","scope":"https://www.googleapis.com/auth/firebase.messaging","aud":"%s","iat":%d,"exp":%d}`,
		cfg.ClientEmail, tokenURL, now.Unix(), now.Add(time.Hour).Unix())
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(claims))
	signingInput := header + "." + payload

	hash := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	jwt := signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)

	form := fmt.Sprintf("grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer&assertion=%s", jwt)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取 OAuth token 失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("获取 OAuth token 失败(%d): %s", resp.StatusCode, string(respBody))
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(respBody, &tok); err != nil || tok.AccessToken == "" {
		return "", errors.New("OAuth token 响应异常")
	}
	return tok.AccessToken, nil
}

// parseRSAKey 解析 PEM 中的 RSA 私钥。
func parseRSAKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("无效的 PEM 数据")
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	return nil, errors.New("无法解析 RSA 私钥")
}
