package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type LotteryHandler struct {
	lotteryService *service.LotteryService
}

func NewLotteryHandler(lotteryService *service.LotteryService) *LotteryHandler {
	return &LotteryHandler{lotteryService: lotteryService}
}

func (h *LotteryHandler) ListCampaigns(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, result, err := h.lotteryService.ListCampaigns(c.Request.Context(), pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, dto.LotteryCampaignsFromService(items), result.Total, page, pageSize)
}

func (h *LotteryHandler) CreateCampaign(c *gin.Context) {
	var req dto.CreateLotteryCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	created, err := h.lotteryService.CreateCampaign(c.Request.Context(), subject.UserID, service.CreateLotteryCampaignInput{
		Name:                         req.Name,
		Subtitle:                     req.Subtitle,
		PrizeCount:                   req.PrizeCount,
		MaxParticipants:              req.MaxParticipants,
		EarlyBoostParticipantPercent: req.EarlyBoostParticipantPercent,
		RechargeBoostCapPercent:      req.RechargeBoostCapPercent,
		PromoText:                    req.PromoText,
		PromoImageURL:                req.PromoImageURL,
		Codes:                        req.Codes,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, dto.LotteryCampaignFromService(created))
}

func (h *LotteryHandler) GetCampaign(c *gin.Context) {
	id, ok := parseLotteryCampaignID(c)
	if !ok {
		return
	}
	campaign, err := h.lotteryService.GetCampaign(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.LotteryCampaignFromService(campaign))
}

func (h *LotteryHandler) FinishCampaign(c *gin.Context) {
	id, ok := parseLotteryCampaignID(c)
	if !ok {
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	campaign, err := h.lotteryService.FinishCampaign(c.Request.Context(), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.LotteryCampaignFromService(campaign))
}

func parseLotteryCampaignID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid lottery campaign ID")
		return 0, false
	}
	return id, true
}
