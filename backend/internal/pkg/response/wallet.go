package response

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// WalletNumber marshals a validated decimal string as a JSON number without a
// float64 conversion. Database numeric snapshots retain their stored scale.
type WalletNumber string

func (n WalletNumber) MarshalJSON() ([]byte, error) {
	raw := strings.TrimSpace(string(n))
	if _, err := decimal.NewFromString(raw); err != nil {
		return nil, err
	}
	return []byte(raw), nil
}

type WalletResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
	Data    any    `json:"data,omitempty"`
}

func WalletSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, WalletResponse{Code: 0, Message: "ok", Reason: "", Data: data})
}

func WalletCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, WalletResponse{Code: 0, Message: "ok", Reason: "", Data: data})
}

func WalletError(c *gin.Context, status int, reason, message string, data any) {
	c.JSON(status, WalletResponse{Code: status, Message: message, Reason: reason, Data: data})
}

func WalletErrorFrom(c *gin.Context, err error) {
	status, body := infraerrors.ToHTTP(err)
	data := any(nil)
	if body.Reason == "INSUFFICIENT_BALANCE" && body.Metadata != nil {
		balance, balanceOK := decimalNumber(body.Metadata["balance"])
		required, requiredOK := decimalNumber(body.Metadata["required"])
		if balanceOK && requiredOK {
			data = gin.H{"balance": balance, "required": required}
		}
	}
	if body.Reason == "BALANCE_DEBIT_BUSY" && body.Metadata != nil {
		if seconds, err := strconv.Atoi(body.Metadata["retry_after"]); err == nil && seconds > 0 {
			c.Header("Retry-After", strconv.Itoa(seconds))
		}
	}
	WalletError(c, status, body.Reason, body.Message, data)
}

func decimalNumber(raw string) (WalletNumber, bool) {
	if _, err := decimal.NewFromString(strings.TrimSpace(raw)); err != nil {
		return "", false
	}
	return WalletNumber(strings.TrimSpace(raw)), true
}

var _ json.Marshaler = WalletNumber("")
