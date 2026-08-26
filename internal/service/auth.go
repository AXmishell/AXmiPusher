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
	ErrBadOldPass    = errors.New("旧密码不正确")
)

// jwtClaims JWT 载荷。
type jwtClaims struct {
	UserID   uint64 `json:"uid"`
	TenantID uint64 `json:"tid"`
	Role     string `json:"role"`
	Kind     string `json:"kind"` // user | admin
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

// defaultIfEmpty 空值回退。
func defaultIfEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// Register 开放注册: 直接创建用户(租户已折叠入用户, 业务表 tenant_id 即用户 ID)。
func (s *AuthService) Register(email, password, tenantName, nickname string) (*models.User, error) {
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	user := models.User{}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrEmailExists
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			return err
		}
		user = models.User{
			Email:        email,
			PasswordHash: string(hash),
			Nickname:     nickname,
			Name:         defaultIfEmpty(tenantName, email),
			Role:         models.RoleTenantAdmin,
			Status:       models.StatusActive,
		}
		return tx.Create(&user).Error
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
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

// ChangePassword 修改密码。
// 密码用 bcrypt 存储(bcrypt 自动生成随机盐并嵌入哈希串, 无需额外处理盐)。
func (s *AuthService) ChangePassword(userID uint64, oldPassword, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}
	// 验证旧密码。
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrBadOldPass
	}
	// 新旧一致直接成功(幂等)。
	if oldPassword == newPassword {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return err
	}
	return s.db.Model(&user).Update("password_hash", string(hash)).Error
}

// CreateToken 签发用户 JWT。
func (s *AuthService) CreateToken(user *models.User) (string, error) {
	claims := jwtClaims{
		UserID:   user.ID,
		TenantID: user.ID,
		Role:     user.Role,
		Kind:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "messagepusher",
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

// parseJWT 解析并校验 JWT 签名, 返回载荷。
func (s *AuthService) parseJWT(tokenStr string) (*jwtClaims, error) {
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
	return claims, nil
}

// ParseToken 解析用户 JWT。
func (s *AuthService) ParseToken(tokenStr string) (*models.User, error) {
	claims, err := s.parseJWT(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.Kind != "user" {
		return nil, fmt.Errorf("凭证类型错误")
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

// AdminLogin 管理员登录(查 admins 表)。
func (s *AuthService) AdminLogin(email, password string) (*models.Admin, error) {
	var admin models.Admin
	if err := s.db.Where("email = ?", email).First(&admin).Error; err != nil {
		return nil, ErrBadCredential
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return nil, ErrBadCredential
	}
	if admin.Status != models.StatusActive {
		return nil, ErrUserDisabled
	}
	now := time.Now()
	s.db.Model(&admin).Update("last_login_at", &now)
	return &admin, nil
}

// CreateAdminToken 签发管理员 JWT(kind=admin)。
func (s *AuthService) CreateAdminToken(admin *models.Admin) (string, error) {
	claims := jwtClaims{
		UserID:   admin.ID,
		TenantID: 0,
		Role:     admin.Role,
		Kind:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "messagepusher",
			Subject:   fmt.Sprintf("%d", admin.ID),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

// ParseAdminToken 解析管理员 JWT。
func (s *AuthService) ParseAdminToken(tokenStr string) (*models.Admin, error) {
	claims, err := s.parseJWT(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.Kind != "admin" {
		return nil, fmt.Errorf("凭证类型错误")
	}
	var admin models.Admin
	if err := s.db.First(&admin, claims.UserID).Error; err != nil {
		return nil, err
	}
	if admin.Status != models.StatusActive {
		return nil, ErrUserDisabled
	}
	return &admin, nil
}

// AdminChangePassword 修改管理员密码。
func (s *AuthService) AdminChangePassword(adminID uint64, oldPassword, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	var admin models.Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return err
	}
	// 验证旧密码。
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrBadOldPass
	}
	// 新旧一致直接成功(幂等)。
	if oldPassword == newPassword {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return err
	}
	return s.db.Model(&admin).Update("password_hash", string(hash)).Error
}

// ListAdmins 管理员列表(超管管理用)。
func (s *AuthService) ListAdmins() ([]models.Admin, error) {
	var admins []models.Admin
	err := s.db.Order("id DESC").Find(&admins).Error
	return admins, err
}

// CreateAdmin 创建普通管理员。
func (s *AuthService) CreateAdmin(email, password, nickname string) (*models.Admin, error) {
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	var count int64
	if err := s.db.Model(&models.Admin{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrEmailExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, err
	}
	admin := models.Admin{
		Email:        email,
		PasswordHash: string(hash),
		Nickname:     nickname,
		Role:         models.AdminRoleNormal,
		Status:       models.StatusActive,
	}
	if err := s.db.Create(&admin).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

// SetAdminStatus 启用/禁用管理员(含自我保护与最后一位保护)。
func (s *AuthService) SetAdminStatus(operatorID, targetID uint64, status string) error {
	if targetID == operatorID && status == models.StatusDisabled {
		return errors.New("不能禁用当前登录的管理员")
	}
	var target models.Admin
	if err := s.db.First(&target, targetID).Error; err != nil {
		return errors.New("管理员不存在")
	}
	if status == models.StatusDisabled {
		var activeCount int64
		if err := s.db.Model(&models.Admin{}).Where("status = ?", models.StatusActive).Count(&activeCount).Error; err != nil {
			return err
		}
		if activeCount <= 1 {
			return errors.New("不能禁用最后一位有效管理员")
		}
	}
	return s.db.Model(&target).Update("status", status).Error
}

// ResetAdminPassword 重置管理员密码(超管操作, 免旧密码)。
func (s *AuthService) ResetAdminPassword(id uint64, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	var admin models.Admin
	if err := s.db.First(&admin, id).Error; err != nil {
		return errors.New("管理员不存在")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return err
	}
	return s.db.Model(&admin).Update("password_hash", string(hash)).Error
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

// ResolveAPIKey 校验 API Key, 返回 key(tenant_id 即归属用户 ID)。
func (s *AuthService) ResolveAPIKey(plain string) (*models.APIKey, error) {
	var key models.APIKey
	if err := s.db.Where("key_hash = ? AND status = ?", hashKey(plain), models.StatusActive).First(&key).Error; err != nil {
		return nil, err
	}
	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, fmt.Errorf("API Key 已过期")
	}
	now := time.Now()
	s.db.Model(&key).Update("last_used_at", &now)
	return &key, nil
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
