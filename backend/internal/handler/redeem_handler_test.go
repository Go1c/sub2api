package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type redeemCodeRedeemerStub struct {
	redeemCode *service.RedeemCode
	redeemErr  error
	history    []service.RedeemCode
	calls      int
	userID     int64
	code       string
}

func (s *redeemCodeRedeemerStub) Redeem(_ context.Context, userID int64, code string) (*service.RedeemCode, error) {
	s.calls++
	s.userID = userID
	s.code = code
	if s.redeemErr != nil {
		return nil, s.redeemErr
	}
	return s.redeemCode, nil
}

func (s *redeemCodeRedeemerStub) GetUserHistory(context.Context, int64, int) ([]service.RedeemCode, error) {
	return append([]service.RedeemCode(nil), s.history...), nil
}

type promoCodeRedeemerStub struct {
	promo   *service.PromoCode
	history []service.RedeemCode
	err     error
	calls   int
	userID  int64
	code    string
}

func (s *promoCodeRedeemerStub) RedeemPromoCode(_ context.Context, userID int64, code string) (*service.PromoCode, error) {
	s.calls++
	s.userID = userID
	s.code = code
	if s.err != nil {
		return nil, s.err
	}
	return s.promo, nil
}

func (s *promoCodeRedeemerStub) GetUserHistory(context.Context, int64, int) ([]service.RedeemCode, error) {
	return append([]service.RedeemCode(nil), s.history...), nil
}

func TestRedeemHandlerReturnsRedeemCodeWithoutPromoFallback(t *testing.T) {
	redeemSvc := &redeemCodeRedeemerStub{
		redeemCode: &service.RedeemCode{
			ID:     7,
			Code:   "BAL-1",
			Type:   service.RedeemTypeBalance,
			Value:  5,
			Status: service.StatusUsed,
		},
	}
	promoSvc := &promoCodeRedeemerStub{}
	handler := NewRedeemHandler(redeemSvc, promoSvc)

	rec := postRedeemForTest(t, handler, "BAL-1")

	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeRedeemResponseForTest(t, rec)
	require.Equal(t, "BAL-1", body.Data.Code)
	require.Equal(t, service.RedeemTypeBalance, body.Data.Type)
	require.Equal(t, 5.0, body.Data.Value)
	require.NotEmpty(t, body.Data.Message)
	require.Equal(t, 1, redeemSvc.calls)
	require.Equal(t, 0, promoSvc.calls)
}

func TestRedeemHandlerFallsBackToPromoCodeWhenRedeemCodeMissing(t *testing.T) {
	redeemSvc := &redeemCodeRedeemerStub{redeemErr: service.ErrRedeemCodeNotFound}
	promoSvc := &promoCodeRedeemerStub{
		promo: &service.PromoCode{
			Code:        "LUMIOAPI",
			BonusAmount: 1.88,
			Status:      service.PromoCodeStatusActive,
		},
	}
	handler := NewRedeemHandler(redeemSvc, promoSvc)

	rec := postRedeemForTest(t, handler, "LUMIOAPI")

	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeRedeemResponseForTest(t, rec)
	require.Equal(t, "LUMIOAPI", body.Data.Code)
	require.Equal(t, "promo", body.Data.Type)
	require.Equal(t, 1.88, body.Data.Value)
	require.NotEmpty(t, body.Data.Message)
	require.Equal(t, int64(42), promoSvc.userID)
	require.Equal(t, "LUMIOAPI", promoSvc.code)
}

func TestRedeemHandlerPropagatesPromoUsageLimit(t *testing.T) {
	redeemSvc := &redeemCodeRedeemerStub{redeemErr: service.ErrRedeemCodeNotFound}
	promoSvc := &promoCodeRedeemerStub{err: service.ErrPromoCodeMaxUsed}
	handler := NewRedeemHandler(redeemSvc, promoSvc)

	rec := postRedeemForTest(t, handler, "LUMIOAPI")

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var body struct {
		Code    int    `json:"code"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "PROMO_CODE_MAX_USED", body.Reason)
}

func TestRedeemHandlerKeepsRedeemNotFoundWhenPromoAlsoMissing(t *testing.T) {
	redeemSvc := &redeemCodeRedeemerStub{redeemErr: service.ErrRedeemCodeNotFound}
	promoSvc := &promoCodeRedeemerStub{err: service.ErrPromoCodeNotFound}
	handler := NewRedeemHandler(redeemSvc, promoSvc)

	rec := postRedeemForTest(t, handler, "MISSING")

	require.Equal(t, http.StatusNotFound, rec.Code)
	var body struct {
		Reason string `json:"reason"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "REDEEM_CODE_NOT_FOUND", body.Reason)
}

func TestRedeemHandlerReturnsDataConflictWithoutPromoFallback(t *testing.T) {
	redeemSvc := &redeemCodeRedeemerStub{redeemErr: service.ErrRedeemCodeDataConflict}
	promoSvc := &promoCodeRedeemerStub{}
	handler := NewRedeemHandler(redeemSvc, promoSvc)

	rec := postRedeemForTest(t, handler, "DUPLICATE")

	require.Equal(t, http.StatusConflict, rec.Code)
	var body struct {
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "REDEEM_CODE_DATA_CONFLICT", body.Reason)
	require.Equal(t, "redeem code data conflict, please contact support", body.Message)
	require.Zero(t, promoSvc.calls)
}

func TestRedeemHandlerHistoryIncludesPromoBalance(t *testing.T) {
	redeemUsedAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	promoUsedAt := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	redeemSvc := &redeemCodeRedeemerStub{history: []service.RedeemCode{{
		ID:        1,
		Code:      "BAL-1",
		Type:      service.RedeemTypeBalance,
		Value:     1,
		Status:    service.StatusUsed,
		UsedAt:    &redeemUsedAt,
		CreatedAt: redeemUsedAt,
	}}}
	promoSvc := &promoCodeRedeemerStub{history: []service.RedeemCode{{
		ID:        -800000000001,
		Code:      "LUMIOAPI",
		Type:      service.RedeemTypePromoBalance,
		Value:     1.88,
		Status:    service.StatusUsed,
		UsedAt:    &promoUsedAt,
		CreatedAt: promoUsedAt,
	}}}
	handler := NewRedeemHandler(redeemSvc, promoSvc)

	rec := getRedeemHistoryForTest(t, handler)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Code int `json:"code"`
		Data []struct {
			Code  string  `json:"code"`
			Type  string  `json:"type"`
			Value float64 `json:"value"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.Len(t, body.Data, 2)
	require.Equal(t, "LUMIOAPI", body.Data[0].Code)
	require.Equal(t, service.RedeemTypePromoBalance, body.Data[0].Type)
	require.Equal(t, 1.88, body.Data[0].Value)
}

func postRedeemForTest(t *testing.T, handler *RedeemHandler, code string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/redeem", bytes.NewBufferString(`{"code":"`+code+`"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
	handler.Redeem(c)
	return rec
}

func getRedeemHistoryForTest(t *testing.T, handler *RedeemHandler) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/redeem/history", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
	handler.GetHistory(c)
	return rec
}

func decodeRedeemResponseForTest(t *testing.T, rec *httptest.ResponseRecorder) struct {
	Code int `json:"code"`
	Data struct {
		ID      int64   `json:"id"`
		Code    string  `json:"code"`
		Type    string  `json:"type"`
		Value   float64 `json:"value"`
		Status  string  `json:"status"`
		Message string  `json:"message"`
	} `json:"data"`
} {
	t.Helper()
	var body struct {
		Code int `json:"code"`
		Data struct {
			ID      int64   `json:"id"`
			Code    string  `json:"code"`
			Type    string  `json:"type"`
			Value   float64 `json:"value"`
			Status  string  `json:"status"`
			Message string  `json:"message"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	return body
}

var _ redeemCodeRedeemer = (*redeemCodeRedeemerStub)(nil)
var _ promoCodeRedeemer = (*promoCodeRedeemerStub)(nil)
