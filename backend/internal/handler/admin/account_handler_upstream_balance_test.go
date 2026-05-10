package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestAccountHandler_UpstreamBalanceLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/login":
		case "/api/user/self":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"id":    42,
					"quota": 500000,
				},
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"id":           42,
				"access_token": "newapi-access-token",
			},
		})
	}))
	defer upstream.Close()

	handler := &AccountHandler{accountUsageService: &service.AccountUsageService{}}
	router := gin.New()
	router.POST("/api/v1/admin/accounts/upstream-balance/login", handler.UpstreamBalanceLogin)

	body := `{"base_url":"` + upstream.URL + `","provider":"newapi","username":"alice@example.com","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/upstream-balance/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Code int `json:"code"`
		Data struct {
			Provider    string `json:"provider"`
			AccessToken string `json:"access_token"`
			UserID      string `json:"user_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != 0 {
		t.Fatalf("expected success code 0, got %d", response.Code)
	}
	if response.Data.Provider != "newapi" || response.Data.AccessToken != "newapi-access-token" || response.Data.UserID != "42" {
		t.Fatalf("unexpected response data: %#v", response.Data)
	}
}
