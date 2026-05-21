package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
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

func (h *LotteryHandler) GetActive(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	campaign, err := h.lotteryService.GetActiveForUser(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.LotteryActiveResponse{
		Campaign: dto.LotteryActiveCampaignFromService(campaign),
	})
}

func (h *LotteryHandler) Draw(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	campaignID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || campaignID <= 0 {
		response.BadRequest(c, "Invalid lottery campaign ID")
		return
	}
	result, err := h.lotteryService.Draw(c.Request.Context(), subject.UserID, campaignID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.LotteryDrawResultFromService(result))
}
