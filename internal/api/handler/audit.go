package handler

import (
	"encoding/json"

	"axmipusher/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Audit 写入审计日志。
func Audit(db *gorm.DB, c *gin.Context, actorID uint64, actorEmail, action string, detail interface{}) {
	raw := ""
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			raw = string(b)
		}
	}
	db.Create(&models.AuditLog{
		ActorID:    actorID,
		ActorEmail: actorEmail,
		Action:     action,
		Detail:     raw,
		IP:         c.ClientIP(),
	})
}
