package telemetry

import (
	"context"
	"database/sql"

	"github.com/gin-gonic/gin"
)

type Module struct {
	handler *Handler
	service *Service
}

func NewModule(db *sql.DB, identity AccessTokenIdentifier) *Module {
	service := NewService(newSQLRepository(db), nil)
	return &Module{handler: newHandler(service, identity), service: service}
}

func (m *Module) RecordServerAuthEvent(ctx context.Context, userID int64, event string) error {
	if m == nil || m.service == nil {
		return nil
	}
	return m.service.RecordServerAuthEvent(ctx, userID, event)
}

func (m *Module) RegisterPublicRoutes(v1 *gin.RouterGroup, middleware ...gin.HandlerFunc) {
	if m == nil || m.handler == nil || v1 == nil {
		return
	}
	group := v1.Group("/telemetry")
	handlers := make([]gin.HandlerFunc, 0, len(middleware)+1)
	handlers = append(handlers, middleware...)
	handlers = append(handlers, m.handler.Ingest)
	group.POST("/events", handlers...)
}

func (m *Module) RegisterAdminRoutes(v1 *gin.RouterGroup, middleware ...gin.HandlerFunc) {
	if m == nil || m.handler == nil || v1 == nil {
		return
	}
	group := v1.Group("/telemetry")
	group.Use(middleware...)
	group.GET("/stats", m.handler.Stats)
}
