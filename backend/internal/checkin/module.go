package checkin

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

type Module struct {
	handler *Handler
}

func NewModule(db *sql.DB, balance BalanceCacheInvalidator, auth AuthCacheInvalidator) *Module {
	repository := newSQLRepository(db, cryptoRandomSource{})
	return &Module{handler: newHandler(NewService(repository, balance, auth))}
}

func (m *Module) RegisterUserRoutes(v1 *gin.RouterGroup, middleware ...gin.HandlerFunc) {
	checkIn := v1.Group("/user/checkin")
	checkIn.Use(middleware...)
	checkIn.GET("", m.handler.GetUserStatus)
	checkIn.POST("", m.handler.CheckIn)
}

func (m *Module) RegisterAdminRoutes(v1 *gin.RouterGroup, middleware ...gin.HandlerFunc) {
	checkIns := v1.Group("/admin/affiliates/checkins")
	checkIns.Use(middleware...)
	checkIns.GET("", m.handler.ListAdminRecords)
	checkIns.GET("/stats", m.handler.GetAdminStats)
	checkIns.GET("/settings", m.handler.GetSettings)
	checkIns.PUT("/settings", m.handler.UpdateSettings)
}
