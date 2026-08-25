//go:build unit

package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

var errInvalidToken = errors.New("invalid token")

func init() {
	gin.SetMode(gin.TestMode)
}

type stubIdentity struct {
	userID int64
	err    error
}

func (s stubIdentity) IdentifyAccessToken(string) (int64, error) {
	return s.userID, s.err
}

func testNow() time.Time {
	return time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
}

func newTestRouter(repo *memoryRepository, identity AccessTokenIdentifier) *gin.Engine {
	handler := newHandler(NewService(repo, testNow), identity)
	mod := &Module{handler: handler, service: handler.service}
	router := gin.New()
	v1 := router.Group("/api/v1")
	mod.RegisterPublicRoutes(v1)
	mod.RegisterAdminRoutes(v1, func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			return
		}
		c.Next()
	})
	return router
}

func postJSON(router http.Handler, path, body, bearer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func getStats(router http.Handler, rawQuery, bearer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/stats?"+rawQuery, nil)
	if bearer != "" {
		req.Header.Set("Authorization", bearer)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestIngestWithoutAuthAcceptedAndDoesNotEchoProps(t *testing.T) {
	repo := newMemoryRepository()
	router := newTestRouter(repo, nil)
	body := `{
		"event":"signup_page_view",
		"ts":` + itoa(testNow().UnixMilli()) + `,
		"props":{"email":"leak@example.com","client_source":"bestcodex_web","route":"/signup"}
	}`
	rec := postJSON(router, "/api/v1/telemetry/events", body, "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"accepted":true`)
	require.NotContains(t, rec.Body.String(), "leak@example.com")
	require.NotContains(t, rec.Body.String(), "user_id")
	require.NotContains(t, rec.Body.String(), "signup_page_view")
	require.Len(t, repo.events, 1)
	require.Nil(t, repo.events[0].UserID)
}

type failingInsertRepository struct {
	memoryRepository
}

func (f *failingInsertRepository) InsertEvent(context.Context, Event) error {
	return errInvalidToken
}

func TestIngestPersistFailureIsAcceptedSoAuthIsNotBlocked(t *testing.T) {
	repo := &failingInsertRepository{memoryRepository: *newMemoryRepository()}
	handler := newHandler(NewService(repo, testNow), nil)
	ginRouter := gin.New()
	ginRouter.POST("/api/v1/telemetry/events", handler.Ingest)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/events", bytes.NewBufferString(`{"event":"login_page_view","ts":1,"props":{"client_source":"bestcodex_desktop_codex"}}`))
	req.Header.Set("Content-Type", "application/json")
	ginRouter.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"accepted":true`)
}

func TestIngestUnknownEventAndInvalidJSON(t *testing.T) {
	router := newTestRouter(newMemoryRepository(), nil)
	unknown := postJSON(router, "/api/v1/telemetry/events", `{"event":"not_a_real_event","ts":1}`, "")
	require.Equal(t, http.StatusBadRequest, unknown.Code)
	require.Contains(t, unknown.Body.String(), "UNKNOWN_EVENT")
	require.NotContains(t, unknown.Body.String(), "not_a_real_event")

	invalid := postJSON(router, "/api/v1/telemetry/events", `{`, "")
	require.Equal(t, http.StatusBadRequest, invalid.Code)
}

func TestIngestInvalidBearerIsAnonymous(t *testing.T) {
	repo := newMemoryRepository()
	router := newTestRouter(repo, stubIdentity{err: errInvalidToken})
	rec := postJSON(router, "/api/v1/telemetry/events", `{"event":"login_page_view","ts":1,"props":{}}`, "expired-token")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.events, 1)
	require.Nil(t, repo.events[0].UserID)
}

func TestDuplicateAuthLoginSuccessWindowedDedup(t *testing.T) {
	repo := newMemoryRepository()
	router := newTestRouter(repo, nil)
	body := `{
		"event":"auth_login_success",
		"ts":` + itoa(testNow().UnixMilli()) + `,
		"props":{"attribution_id":"bc_abcdefghijklmnop","auth_method":"2fa"}
	}`
	first := postJSON(router, "/api/v1/telemetry/events", body, "")
	second := postJSON(router, "/api/v1/telemetry/events", body, "")
	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, http.StatusOK, second.Code)
	require.Len(t, repo.events, 1)
}

func TestDuplicateAuthRegisterSuccessLifetimeDedup(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo, testNow)
	require.NoError(t, svc.Ingest(t.Context(), ingestRequest{
		Event: EventAuthRegisterSuccess,
		TS:    testNow().UnixMilli(),
		Props: map[string]string{"attribution_id": "bc_abcdefghijklmnop"},
	}, 0))
	later := NewService(repo, func() time.Time { return testNow().Add(24 * time.Hour) })
	require.NoError(t, later.Ingest(t.Context(), ingestRequest{
		Event: EventAuthRegisterSuccess,
		TS:    testNow().Add(24 * time.Hour).UnixMilli(),
		Props: map[string]string{"attribution_id": "bc_abcdefghijklmnop"},
	}, 0))
	require.Len(t, repo.events, 1)
}

func TestCampaignT1299RegisterSuccessStats(t *testing.T) {
	repo := newMemoryRepository()
	router := newTestRouter(repo, nil)
	aid := "bc_t1299attributionxx"
	body := `{
		"event":"auth_register_success",
		"ts":` + itoa(testNow().UnixMilli()) + `,
		"props":{
			"attribution_id":"` + aid + `",
			"first_touch_source":"sb.sb",
			"first_touch_medium":"referral",
			"first_touch_campaign":"t1299",
			"client_source":"bestcodex_web"
		}
	}`
	require.Equal(t, http.StatusOK, postJSON(router, "/api/v1/telemetry/events", body, "").Code)

	from := testNow().Add(-time.Minute).UnixMilli()
	to := testNow().Add(time.Minute).UnixMilli()
	rec := getStats(router, "from="+itoa(from)+"&to="+itoa(to)+"&campaign=t1299&event=auth_register_success", "Bearer admin")
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Body.String(), "user_id")
	require.NotContains(t, rec.Body.String(), "leak@")
	var envelope struct {
		Data StatsResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, AuthorityFirstPartyIngest, envelope.Data.Authority)
	require.Len(t, envelope.Data.Rows, 1)
	require.Equal(t, EventAuthRegisterSuccess, envelope.Data.Rows[0].Event)
	require.Equal(t, int64(1), envelope.Data.Rows[0].EventCount)
	require.Equal(t, int64(1), envelope.Data.Rows[0].UniqueAttributionIDs)
	require.Equal(t, MeasureEventCountAndUniqueAnonymousID, envelope.Data.Rows[0].Measure)
}

func TestStatsAnonymousUnauthorizedAndOmitsUserID(t *testing.T) {
	router := newTestRouter(newMemoryRepository(), nil)
	rec := getStats(router, "from=1&to=2", "")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.NotContains(t, rec.Body.String(), "user_id")
}

func TestBearerOverlaysAccountFirstTouchOntoDesktopLaunch(t *testing.T) {
	repo := newMemoryRepository()
	userID := int64(42)
	require.NoError(t, repo.UpsertAttribution(t.Context(), AccountAttribution{
		UserID:             userID,
		FirstTouchSource:   "sb.sb",
		FirstTouchMedium:   "referral",
		FirstTouchCampaign: "t1299",
		FirstAttributionID: "bc_accountfirsttouch1",
		LastTouchSource:    "sb.sb",
		LastTouchMedium:    "referral",
		LastTouchCampaign:  "t1299",
		LastAttributionID:  "bc_accountfirsttouch1",
		FirstTouchAt:       testNow().Add(-time.Hour),
		LastTouchAt:        testNow().Add(-time.Hour),
	}))
	router := newTestRouter(repo, stubIdentity{userID: userID})
	newAID := "bc_newdesktoplaunchid1"
	body := `{
		"event":"app_first_launch",
		"ts":` + itoa(testNow().UnixMilli()) + `,
		"props":{
			"attribution_id":"` + newAID + `",
			"client_source":"bestcodex_desktop_codex"
		}
	}`
	rec := postJSON(router, "/api/v1/telemetry/events", body, "user-token")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.events, 1)
	require.Equal(t, "t1299", repo.events[0].FirstTouchCampaign)
	require.Equal(t, "sb.sb", repo.events[0].FirstTouchSource)
	require.NotNil(t, repo.events[0].UserID)
	require.Equal(t, userID, *repo.events[0].UserID)
	require.Equal(t, newAID, repo.events[0].AttributionID)
	attr, err := repo.GetAttribution(t.Context(), userID)
	require.NoError(t, err)
	require.Equal(t, "bc_accountfirsttouch1", attr.FirstAttributionID)
	require.Equal(t, newAID, attr.LastAttributionID)
	require.Equal(t, "t1299", attr.FirstTouchCampaign)
}

func TestIllegalPropsAreNotStored(t *testing.T) {
	repo := newMemoryRepository()
	router := newTestRouter(repo, nil)
	body := `{
		"event":"auth_login_failure",
		"ts":` + itoa(testNow().UnixMilli()) + `,
		"props":{
			"email":"leak@example.com",
			"password":"s3cret",
			"token":"access-token",
			"user_id":"99",
			"url":"https://bestcodex.app/login?x=1",
			"invite":"abc",
			"fingerprint":"fp",
			"code":"000000",
			"error_code":"AUTH_INVALID",
			"client_source":"bestcodex_web"
		}
	}`
	require.Equal(t, http.StatusOK, postJSON(router, "/api/v1/telemetry/events", body, "").Code)
	require.Len(t, repo.events, 1)
	stored, err := json.Marshal(repo.events[0])
	require.NoError(t, err)
	text := string(stored)
	require.NotContains(t, text, "leak@example.com")
	require.NotContains(t, text, "s3cret")
	require.NotContains(t, text, "access-token")
	require.NotContains(t, text, "000000")
	require.NotContains(t, text, "https://bestcodex.app/login")
	require.Equal(t, "AUTH_INVALID", repo.events[0].ErrorCode)
	require.Equal(t, "bestcodex_web", repo.events[0].ClientSource)
}

func TestStatsMissingRangeIsBadRequest(t *testing.T) {
	router := newTestRouter(newMemoryRepository(), nil)
	rec := getStats(router, "from=1", "Bearer admin")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
