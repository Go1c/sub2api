package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetUserWalletDebitsReturnsLedgerFieldsWithoutSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()
	createdAt := time.Date(2026, 8, 19, 8, 12, 0, 0, time.UTC)
	adminSvc.walletDebits = []service.BalanceDebitTransaction{
		{
			TxnID:        "8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a",
			ClientID:     "9668f69e-32c4-48e9-9992-280951dcb85c",
			ClientName:   "CCHaven Control",
			Amount:       "19.90",
			BalanceAfter: "583.46000000",
			Currency:     "CNY",
			Purpose:      "cchaven_monthly",
			Ref:          "CC20260819-100001",
			CreatedAt:    createdAt,
		},
	}
	handler := NewUserHandler(adminSvc, nil, nil, nil, nil, nil, nil)
	router.GET("/api/v1/admin/users/:id/wallet-debits", handler.GetUserWalletDebits)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/7/wallet-debits?page=1&page_size=20", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data struct {
			Items []map[string]any `json:"items"`
			Total int64            `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, int64(1), body.Data.Total)
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, "8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a", body.Data.Items[0]["txn_id"])
	require.Equal(t, "CCHaven Control", body.Data.Items[0]["client_name"])
	require.Equal(t, 19.9, body.Data.Items[0]["amount"])
	require.Equal(t, "cchaven_monthly", body.Data.Items[0]["purpose"])
	require.Equal(t, "CC20260819-100001", body.Data.Items[0]["ref"])
	_, hasSecret := body.Data.Items[0]["secret"]
	_, hasSecretHash := body.Data.Items[0]["secret_hash"]
	require.False(t, hasSecret)
	require.False(t, hasSecretHash)
	require.NotContains(t, rec.Body.String(), "bcs_")
}

func TestGetBalanceHistoryReturnsWalletDebitItems(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()
	usedBy := int64(7)
	usedAt := time.Date(2026, 8, 19, 8, 12, 0, 0, time.UTC)
	adminSvc.redeems = []service.RedeemCode{
		{
			ID:        -700000000004,
			Code:      "8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a",
			Type:      service.RedeemTypeWalletDebit,
			Value:     -19.90,
			Status:    service.StatusUsed,
			UsedBy:    &usedBy,
			UsedAt:    &usedAt,
			CreatedAt: usedAt,
			Notes:     "CCHaven Control · purpose=cchaven_monthly · ref=CC20260819-100001 · txn_id=8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a · 扣后余额 583.46000000",
		},
	}
	handler := NewUserHandler(adminSvc, nil, nil, nil, nil, nil, nil)
	router.GET("/api/v1/admin/users/:id/balance-history", handler.GetBalanceHistory)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/7/balance-history?type=wallet_debit", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data struct {
			Items []struct {
				Type  string  `json:"type"`
				Value float64 `json:"value"`
				Notes string  `json:"notes"`
				Code  string  `json:"code"`
			} `json:"items"`
			TotalRecharged float64 `json:"total_recharged"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, service.RedeemTypeWalletDebit, body.Data.Items[0].Type)
	require.InDelta(t, -19.90, body.Data.Items[0].Value, 0.0001)
	require.Contains(t, body.Data.Items[0].Notes, "purpose=cchaven_monthly")
	require.Equal(t, "8d2e3ca4-ccf2-47af-a26e-a0170ea39e7a", body.Data.Items[0].Code)
	require.Equal(t, 100.0, body.Data.TotalRecharged)
}
