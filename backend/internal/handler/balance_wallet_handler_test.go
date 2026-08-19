//go:build unit

package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestWalletSuccessEnvelopeKeepsReasonAndNumericMoney(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writeWalletSuccess(c, walletDebitResponse{
		TxnID: "txn", Amount: walletNumber("19.90"), Balance: walletNumber("3.25000000"), Currency: "CNY",
	})

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.JSONEq(t, `0`, string(body["code"]))
	require.JSONEq(t, `"ok"`, string(body["message"]))
	require.JSONEq(t, `""`, string(body["reason"]))
	require.Contains(t, string(body["data"]), `"amount":19.90`)
	require.Contains(t, string(body["data"]), `"balance":3.25000000`)
}
