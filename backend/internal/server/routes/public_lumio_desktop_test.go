//go:build unit

package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type lumioDesktopRouteSettingRepo struct {
	values map[string]string
}

func (r *lumioDesktopRouteSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (r *lumioDesktopRouteSettingRepo) GetValue(context.Context, string) (string, error) {
	panic("unexpected GetValue call")
}

func (r *lumioDesktopRouteSettingRepo) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (r *lumioDesktopRouteSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (r *lumioDesktopRouteSettingRepo) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (r *lumioDesktopRouteSettingRepo) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (r *lumioDesktopRouteSettingRepo) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestRegisterPublicRoutes_LumioDesktopConfigDoesNotRequireAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settingService := service.NewSettingService(&lumioDesktopRouteSettingRepo{
		values: map[string]string{
			service.SettingKeyRegistrationEnabled: "true",
		},
	}, &config.Config{})
	handlers := &handler.Handlers{
		Setting: handler.NewSettingHandler(settingService, "test-version"),
	}
	router := gin.New()
	RegisterPublicRoutes(router.Group("/api/v1"), handlers)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/config", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"default_model":"gpt-5.4"`)
}
