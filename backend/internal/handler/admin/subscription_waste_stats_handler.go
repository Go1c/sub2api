package admin

import (
	"context"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SubscriptionWasteStatsService interface {
	GetWasteStats(ctx context.Context, query service.WasteStatsQuery) (service.WasteStatsResult, error)
}

type SubscriptionWasteStatsHandler struct {
	service SubscriptionWasteStatsService
	now     func() time.Time
}

func NewSubscriptionWasteStatsHandler(svc SubscriptionWasteStatsService) *SubscriptionWasteStatsHandler {
	return &SubscriptionWasteStatsHandler{
		service: svc,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// Get returns the full waste-stats projection.
func (h *SubscriptionWasteStatsHandler) Get(c *gin.Context) {
	query, err := h.parseQuery(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	query.Projection = service.WasteStatsProjectionFull
	result, err := h.service.GetWasteStats(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// GetByPlan returns only the plan projection.
func (h *SubscriptionWasteStatsHandler) GetByPlan(c *gin.Context) {
	query, err := h.parseQuery(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	query.Projection = service.WasteStatsProjectionByPlan
	result, err := h.service.GetWasteStats(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result.ByPlan)
}

// GetTimeSeries returns only the time-series projection.
func (h *SubscriptionWasteStatsHandler) GetTimeSeries(c *gin.Context) {
	query, err := h.parseQuery(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	query.Projection = service.WasteStatsProjectionTimeSeries
	result, err := h.service.GetWasteStats(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result.TimeSeries)
}

func (h *SubscriptionWasteStatsHandler) parseQuery(c *gin.Context) (service.WasteStatsQuery, error) {
	var query service.WasteStatsQuery
	now := h.now().UTC()
	startRaw := c.Query("start")
	endRaw := c.Query("end")
	if startRaw == "" && endRaw == "" {
		query.EndTime = now
		query.StartTime = now.AddDate(0, 0, -30)
	} else {
		start, err := parseWasteStatsTime(startRaw)
		if err != nil {
			return query, err
		}
		end, err := parseWasteStatsEndTime(endRaw)
		if err != nil {
			return query, err
		}
		query.StartTime = start
		query.EndTime = end
	}
	query.Window = c.DefaultQuery("window", service.WasteStatsWindowAll)

	if raw := c.Query("plan_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return query, errInvalidWasteStatsParam("plan_id")
		}
		query.PlanID = &id
	}
	if raw := c.Query("user_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return query, errInvalidWasteStatsParam("user_id")
		}
		query.UserID = &id
	}
	return query, nil
}

func parseWasteStatsTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, errInvalidWasteStatsParam("start/end")
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, errInvalidWasteStatsParam("start/end")
}

func parseWasteStatsEndTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, errInvalidWasteStatsParam("start/end")
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t.AddDate(0, 0, 1).UTC(), nil
	}
	return time.Time{}, errInvalidWasteStatsParam("start/end")
}

type wasteStatsParamError string

func errInvalidWasteStatsParam(name string) error {
	return wasteStatsParamError("invalid " + name)
}

func (e wasteStatsParamError) Error() string {
	return string(e)
}
