package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

func TestProvideHTTPServer_H2CPreservesGlobalRequestBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusNoContent)
	})

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:               "127.0.0.1",
			Port:               0,
			MaxRequestBodySize: 8,
			H2C: config.H2CConfig{
				Enabled:                      true,
				MaxConcurrentStreams:         50,
				MaxReadFrameSize:             1 << 20,
				MaxUploadBufferPerConnection: 2 << 20,
				MaxUploadBufferPerStream:     512 << 10,
			},
		},
		Gateway: config.GatewayConfig{MaxBodySize: 8},
	}

	server := ProvideHTTPServer(cfg, router)
	if server.Protocols == nil || !server.Protocols.HTTP1() || !server.Protocols.UnencryptedHTTP2() {
		t.Fatalf("server protocols = %v, want HTTP/1 and unencrypted HTTP/2", server.Protocols)
	}
	if server.HTTP2 == nil {
		t.Fatal("server HTTP2 config is nil")
	}
	if server.HTTP2.MaxConcurrentStreams != 50 {
		t.Fatalf("MaxConcurrentStreams = %d, want 50", server.HTTP2.MaxConcurrentStreams)
	}
	if server.HTTP2.MaxReadFrameSize != 1<<20 {
		t.Fatalf("MaxReadFrameSize = %d, want %d", server.HTTP2.MaxReadFrameSize, 1<<20)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("0123456789"))
	rec := httptest.NewRecorder()

	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}
