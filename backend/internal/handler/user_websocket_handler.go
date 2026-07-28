package handler

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	userWSBaseProtocol   = "sub2api-user"
	userWSWriteWait      = 10 * time.Second
	userWSPongWait       = 60 * time.Second
	userWSPingPeriod     = 30 * time.Second
	userWSMaxMessageSize = 1024
	userWSTestRateLimit  = 30 * time.Second
)

var userWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Same-origin browser clients; allow empty Origin for non-browser tools.
		_ = strings.TrimSpace(r.Header.Get("Origin"))
		return true
	},
	Subprotocols: []string{userWSBaseProtocol},
}

type userWSConnAdapter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *userWSConnAdapter) SendJSON(v any) error {
	if c == nil || c.conn == nil {
		return errors.New("nil websocket conn")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(userWSWriteWait))
	return c.conn.WriteJSON(v)
}

func (c *userWSConnAdapter) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// UserWebsocketWS handles GET /api/v1/user/ws/notifications
func (h *UserHandler) UserWebsocketWS(c *gin.Context) {
	if h.wsNotify == nil || h.wsNotify.Hub() == nil {
		response.Error(c, http.StatusServiceUnavailable, "WebSocket notifications unavailable")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	user, err := h.userService.GetByID(c.Request.Context(), subject.UserID)
	if err != nil || user == nil {
		response.Unauthorized(c, "User not found")
		return
	}
	if user.IsAdmin() {
		response.Forbidden(c, "WebSocket notifications are for non-admin users")
		return
	}
	if !user.WebsocketNotifyEnabled {
		// Close with application code so the client can stop reconnecting until enabled.
		conn, err := userWSUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(4001, "websocket notify disabled"),
			time.Now().Add(userWSWriteWait))
		_ = conn.Close()
		return
	}

	conn, err := userWSUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	adapter := &userWSConnAdapter{conn: conn}
	hub := h.wsNotify.Hub()
	hub.Register(user.ID, adapter)
	defer func() {
		hub.Unregister(user.ID, adapter)
		_ = adapter.Close()
	}()

	conn.SetReadLimit(userWSMaxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(userWSPongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(userWSPongWait))
		return nil
	})

	// Welcome event
	_ = adapter.SendJSON(service.UserWSEvent{
		Type:      "connected",
		Title:     "WebSocket 已连接",
		Message:   "实时通知已就绪",
		Timestamp: time.Now().Unix(),
	})

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(userWSPingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				adapter.mu.Lock()
				_ = conn.SetWriteDeadline(time.Now().Add(userWSWriteWait))
				err := conn.WriteMessage(websocket.PingMessage, nil)
				adapter.mu.Unlock()
				if err != nil {
					return
				}
			}
		}
	}()

	// Read loop: discard client app messages; keep connection for control frames.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			close(done)
			return
		}
	}
}

var (
	userWSTestMu   sync.Mutex
	userWSTestLast = map[int64]time.Time{}
)

// SendWebsocketTest handles POST /api/v1/user/websocket-notify/test
func (h *UserHandler) SendWebsocketTest(c *gin.Context) {
	if h.wsNotify == nil {
		response.Error(c, http.StatusServiceUnavailable, "WebSocket notifications unavailable")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	user, err := h.userService.GetByID(c.Request.Context(), subject.UserID)
	if err != nil || user == nil {
		response.ErrorFrom(c, err)
		return
	}
	if user.IsAdmin() {
		response.Forbidden(c, "WebSocket notifications are for non-admin users")
		return
	}

	userWSTestMu.Lock()
	last := userWSTestLast[subject.UserID]
	if time.Since(last) < userWSTestRateLimit {
		userWSTestMu.Unlock()
		response.Error(c, http.StatusTooManyRequests, "Please wait before sending another test notification")
		return
	}
	userWSTestLast[subject.UserID] = time.Now()
	userWSTestMu.Unlock()

	sent, err := h.wsNotify.NotifyTest(c.Request.Context(), subject.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserWebsocketDisabled):
			response.BadRequest(c, "请先启用 WebSocket 设置")
		case errors.Is(err, service.ErrUserWebsocketNotConnected):
			response.BadRequest(c, "当前没有活跃的 WebSocket 连接，请保持本站页面打开后重试")
		default:
			response.ErrorFrom(c, err)
		}
		return
	}
	response.Success(c, gin.H{"sent": sent, "message": "测试通知已发送"})
}
