package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	responsepkg "github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type BalanceWalletHandler struct {
	svc *service.BalanceWalletService
}

func NewBalanceWalletHandler(svc *service.BalanceWalletService) *BalanceWalletHandler {
	return &BalanceWalletHandler{svc: svc}
}

type walletNumber = responsepkg.WalletNumber

type walletDebitResponse struct {
	TxnID    string       `json:"txn_id"`
	Amount   walletNumber `json:"amount"`
	Balance  walletNumber `json:"balance"`
	Currency string       `json:"currency"`
}

type walletTransactionResponse struct {
	TxnID        string       `json:"txn_id"`
	ClientID     string       `json:"client_id"`
	ClientName   string       `json:"client_name"`
	Amount       walletNumber `json:"amount"`
	BalanceAfter walletNumber `json:"balance_after"`
	Currency     string       `json:"currency"`
	Purpose      string       `json:"purpose"`
	Ref          string       `json:"ref"`
	CreatedAt    string       `json:"created_at"`
}

type walletTransactionsPageResponse struct {
	Items    []walletTransactionResponse `json:"items"`
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
	Pages    int                         `json:"pages"`
}

func writeWalletSuccess(c *gin.Context, data any) { responsepkg.WalletSuccess(c, data) }

func (h *BalanceWalletHandler) Debit(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		responsepkg.WalletError(c, 401, "INVALID_TOKEN", "invalid token", nil)
		return
	}
	client, err := h.svc.AuthenticateClient(c.Request.Context(), c.GetHeader("X-Balance-Client-Key"))
	if err != nil {
		responsepkg.WalletErrorFrom(c, err)
		return
	}
	c.Set(string(middleware2.ContextKeyBalanceClientID), client.ClientID)
	middleware2.SetAuditExtra(c, map[string]any{"client_id": client.ClientID, "user_id": subject.UserID})
	var req struct {
		Amount   json.RawMessage `json:"amount"`
		Currency string          `json:"currency"`
		Purpose  string          `json:"purpose"`
		Ref      string          `json:"ref"`
	}
	if err := decodeStrictWalletJSON(c, &req); err != nil || !isJSONNumber(req.Amount) {
		responsepkg.WalletErrorFrom(c, service.ErrInvalidBalanceDebitRequest)
		return
	}
	result, err := h.svc.Debit(c.Request.Context(), subject.UserID, client, service.BalanceDebitInput{
		Amount: strings.TrimSpace(string(req.Amount)), Currency: req.Currency, Purpose: req.Purpose,
		Ref: req.Ref, IdempotencyKey: c.GetHeader("Idempotency-Key"),
	})
	if err != nil {
		middleware2.SetAuditExtra(c, map[string]any{
			"client_id": client.ClientID, "user_id": subject.UserID, "purpose": req.Purpose,
			"ref": strings.TrimSpace(req.Ref), "amount_summary": strings.TrimSpace(string(req.Amount)),
			"error_code": infraerrors.Reason(err),
		})
		responsepkg.WalletErrorFrom(c, err)
		return
	}
	middleware2.SetAuditExtra(c, map[string]any{
		"client_id": client.ClientID, "user_id": subject.UserID, "txn_id": result.TxnID,
		"purpose": req.Purpose, "ref": strings.TrimSpace(req.Ref), "amount_summary": result.Amount,
	})
	writeWalletSuccess(c, walletDebitResponse{
		TxnID: result.TxnID, Amount: walletNumber(result.Amount), Balance: walletNumber(result.BalanceAfter), Currency: result.Currency,
	})
}

func (h *BalanceWalletHandler) ListTransactions(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		responsepkg.WalletError(c, 401, "INVALID_TOKEN", "invalid token", nil)
		return
	}
	page := parsePositiveQueryInt(c.Query("page"), 1)
	pageSize := parsePositiveQueryInt(c.Query("page_size"), 20)
	result, err := h.svc.ListTransactions(c.Request.Context(), subject.UserID, service.BalanceTransactionFilter{
		Page: page, PageSize: pageSize, Purpose: c.Query("purpose"), Ref: c.Query("ref"), ClientID: c.Query("client_id"),
	})
	if err != nil {
		responsepkg.WalletErrorFrom(c, err)
		return
	}
	items := make([]walletTransactionResponse, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, toWalletTransactionResponse(&result.Items[i]))
	}
	pages := int(math.Ceil(float64(result.Total) / float64(result.PageSize)))
	if pages < 1 {
		pages = 1
	}
	writeWalletSuccess(c, walletTransactionsPageResponse{
		Items: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize, Pages: pages,
	})
}

func (h *BalanceWalletHandler) GetTransaction(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		responsepkg.WalletError(c, 401, "INVALID_TOKEN", "invalid token", nil)
		return
	}
	item, err := h.svc.GetTransaction(c.Request.Context(), subject.UserID, c.Param("txn_id"))
	if err != nil {
		responsepkg.WalletErrorFrom(c, err)
		return
	}
	writeWalletSuccess(c, toWalletTransactionResponse(item))
}

func toWalletTransactionResponse(item *service.BalanceDebitTransaction) walletTransactionResponse {
	if item == nil {
		return walletTransactionResponse{}
	}
	return walletTransactionResponse{
		TxnID: item.TxnID, ClientID: item.ClientID, ClientName: item.ClientName,
		Amount: walletNumber(item.Amount), BalanceAfter: walletNumber(item.BalanceAfter),
		Currency: item.Currency, Purpose: item.Purpose, Ref: item.Ref,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func decodeStrictWalletJSON(c *gin.Context, target any) error {
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return service.ErrInvalidBalanceDebitRequest
	}
	return nil
}

func isJSONNumber(raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] == '"' || bytes.Equal(raw, []byte("null")) {
		return false
	}
	var number json.Number
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&number); err != nil {
		return false
	}
	return string(number) == string(raw)
}

func parsePositiveQueryInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
