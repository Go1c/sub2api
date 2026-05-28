package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminSubscriptionLedgerBadID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewSubscriptionHandler(nil)
	router.GET("/admin/subscriptions/:id/ledger", h.GetLedger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/subscriptions/not-a-number/ledger", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseOptionalAdminEndTimeIncludesDateOnlyDay(t *testing.T) {
	got, err := parseOptionalAdminEndTime("2026-05-28")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, time.Date(2026, 5, 28, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC), *got)
}

func TestParseOptionalAdminEndTimeKeepsExplicitTimestamp(t *testing.T) {
	want := time.Date(2026, 5, 28, 8, 30, 0, 0, time.UTC)
	got, err := parseOptionalAdminEndTime(want.Format(time.RFC3339))

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, want, *got)
}

func TestAdminSubscriptionPatchBadID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewSubscriptionHandler(nil)
	router.PATCH("/admin/subscriptions/:id", h.Update)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/admin/subscriptions/not-a-number", strings.NewReader(`{"reason":"manual correction"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}
