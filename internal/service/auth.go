// Package service 业务逻辑层。
package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"messagepusher/internal/config"
	"messagepusher/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 业务错误。
var (
	ErrEmailExists   = errors.New("邮箱已注册")
	ErrBadCredential = errors.New("邮箱或密码错误")
	ErrUserDisabled  = errors.New("账号已被禁用")
	ErrKeyExists     = errors.New("key 已存在")
)

// jwtClaims JWT 载荷。
type jwtClaims struct {
	UserID   uint64 `json:"uid"`
	TenantID uint64 `json:"tid"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// AuthService 认证服务。
type AuthService struct {
	db        *gorm.DB
	jwtSecret []byte
	tokenTTL  time.Duration
	keyPrefix string
	bindDir   string
}

// NewAuthService 创建认证服务。
func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: []byte(cfg.Auth.JWTSecret),
		tokenTTL:  cfg.Auth.TokenTTL,
		keyPrefix: cfg.Auth.APIKeyPrefix,
	}
}

// Register 开放注册: 创建租户 + 租户管理员。
func (s *AuthService) Register(email, password, tenantName, nickname string) (*models.User, *models.Tenant, error) {
	if err := validatePassword(password); err != nil {
		return nil, nil, err
	}
	if tenantName == "" {
		tenantName = email
	}

	var tenant models.Tenant
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrEmailExists
		}
		tenant = models.Tenant{Name: tenantName, Status: models.StatusActive}
		if err := tx.Create(&tenant).Error; err != nil {
			return err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			return err
		}
		user := models.User{
			TenantID:     tenant.ID,
			Email:        email,
			PasswordHash: string(hash),
			Nickname:     nickname,
			Role:         models.RoleTenantAdmin,
			Status:       models.StatusActive,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	user := &models.User{}
	if err := s.db.Where("email = ?", email).First(user).Error; err != nil {
		return nil, nil, err
	}
	return user, &tenant, nil
}

// Login 登录。
func (s *AuthService) Login(email, password string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, ErrBadCredential
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrBadCredential
	}
	if user.Status != models.StatusActive {
		return nil, ErrUserDisabled
	}
	now := time.Now()
	s.db.Model(&user).Update("last_login_at", &now)
	return &user, nil
}

// CreateToken 签发 JWT。
func (s *AuthService) CreateToken(user *models.User) (string, error) {
	claims := jwtClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "messagepusher",
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

// ParseToken 解析 JWT。
func (s *AuthService) ParseToken(tokenStr string) (*models.User, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("意外的签名方法: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("token 无效")
	}
	var user models.User
	if err := s.db.First(&user, claims.UserID).Error; err != nil {
		return nil, err
	}
	if user.Status != models.StatusActive {
		return nil, ErrUserDisabled
	}
	return &user, nil
}

// CreateAPIKey 创建 API Key, 返回完整明文 key(仅此一次)。
func (s *AuthService) CreateAPIKey(tenantID uint64, name, scopes string, expiresAt *time.Time) (*models.APIKey, string, error) {
	plain := generateKey(s.keyPrefix)
	key := &models.APIKey{
		TenantID:  tenantID,
		Name:      name,
		KeyPrefix: plain[:8],
		KeyHash:   hashKey(plain),
		Scopes:    scopes,
		Status:    models.StatusActive,
		ExpiresAt: expiresAt,
	}
	if err := s.db.Create(key).Error; err != nil {
		return nil, "", err
	}
	return key, plain, nil
}

// ResolveAPIKey 校验 API Key, 返回 key 与所属租户。
func (s *AuthService) ResolveAPIKey(plain string) (*models.APIKey, *models.Tenant, error) {
	var key models.APIKey
	if err := s.db.Where("key_hash = ? AND status = ?", hashKey(plain), models.StatusActive).First(&key).Error; err != nil {
		return nil, nil, err
	}
	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, nil, fmt.Errorf("API Key 已过期")
	}
	now := time.Now()
	s.db.Model(&key).Update("last_used_at", &now)
	var tenant models.Tenant
	if err := s.db.First(&tenant, key.TenantID).Error; err != nil {
		return nil, nil, err
	}
	return &key, &tenant, nil
}

// RevokeAPIKey 吊销 API Key。
func (s *AuthService) RevokeAPIKey(tenantID, keyID uint64) error {
	return s.db.Model(&models.APIKey{}).
		Where("id = ? AND tenant_id = ?", keyID, tenantID).
		Update("status", models.StatusDisabled).Error
}

// ListAPIKeys 列出租户的 API Key。
func (s *AuthService) ListAPIKeys(tenantID uint64) ([]models.APIKey, error) {
	var keys []models.APIKey
	err := s.db.Where("tenant_id = ?", tenantID).Order("id DESC").Find(&keys).Error
	return keys, err
}

func validatePassword(p string) error {
	if len(p) < 8 {
		return fmt.Errorf("密码长度至少 8 位")
	}
	return nil
}

func hashKey(plain string) string {
	h := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(h[:])
}

// generateKey 生成随机 API Key。
func generateKey(prefix string) string {
	buf := make([]byte, 24)
	rand.Read(buf)
	return prefix + hex.EncodeToString(buf)
}
