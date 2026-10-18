package handler

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RedeemHandler handles redeem code-related requests
type RedeemHandler struct {
	redeemService redeemCodeRedeemer
	promoService  promoCodeRedeemer
}

type redeemCodeRedeemer interface {
	Redeem(ctx context.Context, userID int64, code string) (*service.RedeemCode, error)
	GetUserHistory(ctx context.Context, userID int64, limit int) ([]service.RedeemCode, error)
	GetUserHistoryPaginated(ctx context.Context, userID int64, params pagination.PaginationParams) ([]service.RedeemCode, *pagination.PaginationResult, error)
}

type promoCodeRedeemer interface {
	RedeemPromoCode(ctx context.Context, userID int64, code string) (*service.PromoCode, error)
	GetUserHistory(ctx context.Context, userID int64, limit int) ([]service.RedeemCode, error)
}

// NewRedeemHandler creates a new RedeemHandler
func NewRedeemHandler(redeemService redeemCodeRedeemer, promoService promoCodeRedeemer) *RedeemHandler {
	return &RedeemHandler{
		redeemService: redeemService,
		promoService:  promoService,
	}
}

// RedeemRequest represents the redeem code request payload
type RedeemRequest struct {
	Code string `json:"code" binding:"required"`
}

// RedeemResponse represents the redeem response
type RedeemResponse struct {
	ID             int64      `json:"id,omitempty"`
	Message        string     `json:"message"`
	Code           string     `json:"code,omitempty"`
	Type           string     `json:"type"`
	Value          float64    `json:"value"`
	Status         string     `json:"status,omitempty"`
	UsedBy         *int64     `json:"used_by,omitempty"`
	UsedAt         *time.Time `json:"used_at,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	GroupID        *int64     `json:"group_id,omitempty"`
	ValidityDays   int        `json:"validity_days,omitempty"`
	NewBalance     *float64   `json:"new_balance,omitempty"`
	NewConcurrency *int       `json:"new_concurrency,omitempty"`
	Group          *dto.Group `json:"group,omitempty"`
}

// Redeem handles redeeming a code
// POST /api/v1/redeem
func (h *RedeemHandler) Redeem(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req RedeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.redeemService.Redeem(c.Request.Context(), subject.UserID, req.Code)
	if err != nil {
		if !errors.Is(err, service.ErrRedeemCodeNotFound) || h.promoService == nil {
			response.ErrorFrom(c, err)
			return
		}
		promoCode, promoErr := h.promoService.RedeemPromoCode(c.Request.Context(), subject.UserID, req.Code)
		if promoErr != nil {
			if errors.Is(promoErr, service.ErrPromoCodeNotFound) {
				response.ErrorFrom(c, err)
				return
			}
			response.ErrorFrom(c, promoErr)
			return
		}
		response.Success(c, redeemResponseFromPromoCode(promoCode))
		return
	}

	response.Success(c, redeemResponseFromRedeemCode(result))
}

// GetHistory returns the user's redemption history
// GET /api/v1/redeem/history
func (h *RedeemHandler) GetHistory(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Keep the legacy array response for clients that do not request pagination.
	if c.Request.URL.Query().Has("page") || c.Request.URL.Query().Has("page_size") {
		page, pageSize := 1, 20
		for name, target := range map[string]*int{"page": &page, "page_size": &pageSize} {
			if raw, exists := c.GetQuery(name); exists {
				value, err := strconv.Atoi(raw)
				if err != nil || value < 1 {
					response.BadRequest(c, "Invalid "+name)
					return
				}
				*target = value
			}
		}
		if pageSize > 100 {
			pageSize = 100
		}
		if page-1 > int(^uint(0)>>1)/pageSize {
			response.BadRequest(c, "Invalid page")
			return
		}
		codes, result, err := h.redeemService.GetUserHistoryPaginated(c.Request.Context(), subject.UserID, pagination.PaginationParams{Page: page, PageSize: pageSize})
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		out := make([]dto.RedeemCode, 0, len(codes))
		for i := range codes {
			out = append(out, *dto.RedeemCodeFromService(&codes[i]).Localized(c.GetHeader("Accept-Language")))
		}
		response.Paginated(c, out, result.Total, page, pageSize)
		return
	}

	// Default limit is 25
	limit := 25

	codes, err := h.redeemService.GetUserHistory(c.Request.Context(), subject.UserID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if h.promoService != nil {
		promoCodes, err := h.promoService.GetUserHistory(c.Request.Context(), subject.UserID, limit)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		codes = mergeRecentRedeemHistory(codes, promoCodes, limit)
	}

	out := make([]dto.RedeemCode, 0, len(codes))
	for i := range codes {
		out = append(out, *dto.RedeemCodeFromService(&codes[i]).Localized(c.GetHeader("Accept-Language")))
	}
	response.Success(c, out)
}

func mergeRecentRedeemHistory(a, b []service.RedeemCode, limit int) []service.RedeemCode {
	combined := append(append([]service.RedeemCode{}, a...), b...)
	sort.SliceStable(combined, func(i, j int) bool {
		return redeemHistoryTime(combined[i]).After(redeemHistoryTime(combined[j]))
	})
	if limit <= 0 {
		limit = 10
	}
	if len(combined) > limit {
		combined = combined[:limit]
	}
	return combined
}

func redeemHistoryTime(code service.RedeemCode) time.Time {
	if code.UsedAt != nil {
		return *code.UsedAt
	}
	return code.CreatedAt
}

func redeemResponseFromRedeemCode(result *service.RedeemCode) *RedeemResponse {
	if result == nil {
		return nil
	}
	return &RedeemResponse{
		ID:           result.ID,
		Message:      "redeem successful",
		Code:         result.Code,
		Type:         result.Type,
		Value:        result.Value,
		Status:       result.Status,
		UsedBy:       result.UsedBy,
		UsedAt:       result.UsedAt,
		CreatedAt:    &result.CreatedAt,
		GroupID:      result.GroupID,
		ValidityDays: result.ValidityDays,
		Group:        dto.GroupFromServiceShallow(result.Group),
	}
}

func redeemResponseFromPromoCode(result *service.PromoCode) *RedeemResponse {
	if result == nil {
		return nil
	}
	return &RedeemResponse{
		Message: "promo code redeemed successfully",
		Code:    result.Code,
		Type:    "promo",
		Value:   result.BonusAmount,
		Status:  result.Status,
	}
}
