package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"messagepusher/internal/config"
	"messagepusher/internal/models"
	"messagepusher/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// InstallRoutes 注册安装相关路由(未安装状态下可用)。
func (a *App) InstallRoutes(r *gin.Engine) {
	r.GET("/install", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, installHTML)
	})
	g := r.Group("/api/install")
	{
		g.POST("/status", a.handleInstallStatus)
		g.POST("/env-check", a.handleEnvCheck)
		g.POST("/init", a.handleInstallInit)
		g.POST("/admin", a.handleInstallAdmin)
		g.POST("/complete", a.handleInstallComplete)
	}
}

// --- 请求/响应 DTO ---

type envCheckRequest struct {
	DBType     string `json:"db_type"`     // sqlite | postgres
	SQLitePath string `json:"sqlite_path"` //
	PG         pgInfo `json:"pg"`
	StoreType  string `json:"store_type"` // sqlite | clickhouse
	CHDSN      string `json:"ch_dsn"`
	QueueType  string `json:"queue_type"` // inprocess | kafka
	Brokers    string `json:"kafka_brokers"`
}

type pgInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
	SSLMode  string `json:"sslmode"`
}

type checkItem struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
	Msg  string `json:"msg"`
}

type installInitRequest struct {
	AppName       string  `json:"app_name"`
	BaseURL       string  `json:"base_url"`
	DBType        string  `json:"db_type"`
	SQLitePath    string  `json:"sqlite_path"`
	PG            pgInfo  `json:"pg"`
	StoreType     string  `json:"store_type"`
	CHDSN         string  `json:"ch_dsn"`
	QueueType     string  `json:"queue_type"`
	KafkaBrokers  string  `json:"kafka_brokers"`
	RetentionDays int     `json:"retention_days"`
	RedisAddr     string  `json:"redis_addr"`     // 可选
	RedisPassword string  `json:"redis_password"` // 可选
	RedisDB       int     `json:"redis_db"`       // 可选
}

type installInitResponse struct {
	AdminPath string `json:"admin_path"`
	DBType    string `json:"db_type"`
}

type installAdminRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

// --- handlers ---

func (a *App) handleInstallStatus(c *gin.Context) {
	response.OK(c, gin.H{"installed": config.IsInstalled()})
}

func (a *App) handleEnvCheck(c *gin.Context) {
	var req envCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	var checks []checkItem

	switch req.DBType {
	case "sqlite":
		path := req.SQLitePath
		if path == "" {
			path = "./data/messagepusher.db"
		}
		dir := filepath.Dir(path)
		checks = append(checks, checkItem{Name: "SQLite 目录", OK: checkWritable(dir), Msg: dir})
	case "postgres":
		ok, msg := checkPostgres(req.PG)
		checks = append(checks, checkItem{Name: "PostgreSQL 连接", OK: ok, Msg: msg})
	case "":
		checks = append(checks, checkItem{Name: "数据库类型", OK: false, Msg: "未选择"})
	default:
		checks = append(checks, checkItem{Name: "数据库类型", OK: false, Msg: "不支持: " + req.DBType})
	}

	switch req.StoreType {
	case "clickhouse":
		if req.CHDSN == "" {
			checks = append(checks, checkItem{Name: "ClickHouse DSN", OK: false, Msg: "未填写"})
		} else {
			checks = append(checks, checkItem{Name: "ClickHouse DSN", OK: true, Msg: "已填写(连通性在 M2 完善)"})
		}
	default:
		checks = append(checks, checkItem{Name: "消息存储", OK: true, Msg: "SQLite(本地)"})
	}

	if req.QueueType == "kafka" && req.Brokers == "" {
		checks = append(checks, checkItem{Name: "Kafka Brokers", OK: false, Msg: "未填写"})
	}

	response.OK(c, gin.H{"checks": checks})
}

func (a *App) handleInstallInit(c *gin.Context) {
	var req installInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if config.IsInstalled() {
		response.Forbidden(c, "平台已安装, 无法重复初始化")
		return
	}
	if req.DBType != "sqlite" && req.DBType != "postgres" {
		response.BadRequest(c, "数据库类型必须是 sqlite 或 postgres")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		response.ServerError(c, "读取配置失败: "+err.Error())
		return
	}

	// 生成安全随机串: JWT 密钥 + admin 随机路径。
	jwtSecret, _ := randomHex(32)
	adminPath, _ := randomHex(8)

	cfg.App.Name = defaultIfEmpty(req.AppName, "MessagePusher")
	cfg.App.BaseURL = req.BaseURL
	cfg.Auth.JWTSecret = jwtSecret
	cfg.Admin.RandomPath = adminPath
	cfg.Database.Type = req.DBType
	switch req.DBType {
	case "sqlite":
		if req.SQLitePath == "" {
			req.SQLitePath = "./data/messagepusher.db"
		}
		cfg.Database.SQLitePath = req.SQLitePath
	case "postgres":
		cfg.Database.Host = req.PG.Host
		cfg.Database.Port = req.PG.Port
		cfg.Database.User = req.PG.User
		cfg.Database.Password = req.PG.Password
		cfg.Database.Name = req.PG.Name
		cfg.Database.SSLMode = defaultIfEmpty(req.PG.SSLMode, "disable")
	}
	cfg.Store.Type = req.StoreType
	if req.StoreType == "clickhouse" {
		cfg.Store.DSN = req.CHDSN
	}
	cfg.Queue.Type = req.QueueType
	if req.QueueType == "kafka" {
		cfg.Queue.Brokers = splitBrokers(req.KafkaBrokers)
	}
	if req.RetentionDays > 0 {
		cfg.Retention.MessageDays = req.RetentionDays
	}
	// Redis(可选): 留空 = 纯内存模式。
	if req.RedisAddr != "" {
		cfg.Redis.Addr = req.RedisAddr
		cfg.Redis.Password = req.RedisPassword
		cfg.Redis.DB = req.RedisDB
	}
	// 端口固化: 主程序 8080 / 用户中心 19876 / 管理后台 19877。
	if cfg.Server.Port <= 0 {
		cfg.Server.Port = 8080
	}
	cfg.Web.UserPort = 19876
	cfg.Web.AdminPort = 19877
	cfg.Web.APITarget = fmt.Sprintf("http://127.0.0.1:%d", cfg.Server.Port)
	// 前端 dist 目录固化(主程序托管与 cmd/web 均读此配置; 缺省时前端 404)。
	if cfg.Web.UserDist == "" {
		cfg.Web.UserDist = "web/user/dist"
	}
	if cfg.Web.AdminDist == "" {
		cfg.Web.AdminDist = "web/admin/dist"
	}

	// 校验 PG/Kafka 配置正确性(只做格式校验)。
	if cfg.Database.Type == "postgres" && (cfg.Database.Host == "" || cfg.Database.Name == "") {
		response.BadRequest(c, "PostgreSQL 连接信息不完整")
		return
	}

	if err := cfg.Save(); err != nil {
		response.ServerError(c, "写入配置失败: "+err.Error())
		return
	}

	// 用新配置重建应用。
	if err := a.Reinit(cfg); err != nil {
		response.ServerError(c, "初始化组件失败: "+err.Error())
		return
	}

	// 写入默认套餐。
	if err := seedPlans(a.DB); err != nil {
		response.ServerError(c, "初始化套餐失败: "+err.Error())
		return
	}

	response.OK(c, installInitResponse{AdminPath: adminPath, DBType: cfg.Database.Type})
}

func (a *App) handleInstallAdmin(c *gin.Context) {
	var req installAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if config.IsInstalled() {
		response.Forbidden(c, "平台已安装")
		return
	}
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		response.BadRequest(c, "邮箱格式不正确")
		return
	}
	if len(req.Password) < 8 {
		response.BadRequest(c, "密码长度至少 8 位")
		return
	}
	var count int64
	a.DB.Model(&models.User{}).Where("role = ?", models.RolePlatformAdmin).Count(&count)
	if count > 0 {
		response.Conflict(c, "平台管理员已存在")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		response.ServerError(c, "密码加密失败")
		return
	}
	user := models.User{
		TenantID:     0,
		Email:        req.Email,
		PasswordHash: string(hash),
		Nickname:     defaultIfEmpty(req.Nickname, "Administrator"),
		Role:         models.RolePlatformAdmin,
		Status:       models.StatusActive,
	}
	if err := a.DB.Create(&user).Error; err != nil {
		response.Conflict(c, "创建失败: "+err.Error())
		return
	}
	response.OK(c, gin.H{"user_id": user.ID})
}

func (a *App) handleInstallComplete(c *gin.Context) {
	if err := config.MarkInstalled(); err != nil {
		response.ServerError(c, "写入安装锁失败: "+err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

// --- helpers ---

func checkWritable(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	f, err := os.CreateTemp(dir, "mp-check-*")
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(f.Name())
	return true
}

func checkPostgres(p pgInfo) (bool, string) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.Name, defaultIfEmpty(p.SSLMode, "disable"))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return false, "连接失败: " + err.Error()
	}
	conn.Close(ctx)
	return true, "连接成功"
}

// seedPlans 写入默认套餐。
func seedPlans(gdb *gorm.DB) error {
	var count int64
	gdb.Model(&models.Plan{}).Count(&count)
	if count > 0 {
		return nil
	}
	plans := []models.Plan{
		{Name: "免费版", Price: 0, DurationDays: 30, Quota: `{"monthly_messages":1000,"channels":["webhook","email"]}`, Description: "适合个人试用", SortOrder: 1},
		{Name: "基础版", Price: 19.9, DurationDays: 30, Quota: `{"monthly_messages":50000,"channels":["webhook","email","apns"]}`, Description: "适合小团队", SortOrder: 2},
		{Name: "专业版", Price: 99.0, DurationDays: 30, Quota: `{"monthly_messages":1000000,"channels":["webhook","email","apns","fcm","inapp"]}`, Description: "适合规模化业务", SortOrder: 3},
	}
	return gdb.Create(&plans).Error
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func defaultIfEmpty(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func splitBrokers(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
