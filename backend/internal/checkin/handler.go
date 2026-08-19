package checkin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	now     func() time.Time
}

func newHandler(service *Service) *Handler {
	return &Handler{service: service, now: time.Now}
}

func (h *Handler) GetUserStatus(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}
	status, err := h.service.GetUserStatus(c.Request.Context(), userID, h.now())
	if err != nil {
		writeCheckInError(c, err)
		return
	}
	response.Success(c, toUserStatusResponse(status))
}

func (h *Handler) CheckIn(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}
	result, err := h.service.CheckIn(c.Request.Context(), userID, h.now(), ClientInfo{
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		writeCheckInError(c, err)
		return
	}
	payload := toRecordResponse(result.Record, false)
	payload.AlreadyCheckedIn = &result.AlreadyCheckedIn
	response.Success(c, payload)
}

func (h *Handler) ListAdminRecords(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := AdminRecordFilter{
		Search:    c.Query("search"),
		Status:    strings.TrimSpace(c.Query("status")),
		SortBy:    strings.TrimSpace(c.Query("sort_by")),
		SortOrder: strings.TrimSpace(c.Query("sort_order")),
		Page:      page,
		PageSize:  pageSize,
	}
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		userID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || userID <= 0 {
			middleware.AbortWithError(c, http.StatusBadRequest, "INVALID_USER_ID", "user_id must be a positive integer")
			return
		}
		filter.UserID = userID
	}
	if raw := strings.TrimSpace(c.Query("business_date")); raw != "" {
		businessDate, err := time.Parse("2006-01-02", raw)
		if err != nil {
			middleware.AbortWithError(c, http.StatusBadRequest, "INVALID_BUSINESS_DATE", "business_date must use YYYY-MM-DD")
			return
		}
		filter.BusinessDate = &businessDate
	}
	if filter.Status != "" && filter.Status != StatusAwarded && filter.Status != StatusBudgetExhausted {
		middleware.AbortWithError(c, http.StatusBadRequest, "INVALID_CHECKIN_STATUS", "invalid check-in status")
		return
	}

	records, total, err := h.service.ListAdminRecords(c.Request.Context(), filter)
	if err != nil {
		writeCheckInError(c, err)
		return
	}
	items := make([]recordResponse, 0, len(records))
	for _, record := range records {
		items = append(items, toRecordResponse(record, true))
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *Handler) GetSettings(c *gin.Context) {
	settings, err := h.service.GetSettings(c.Request.Context())
	if err != nil {
		writeCheckInError(c, err)
		return
	}
	response.Success(c, toSettingsResponse(settings))
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	var request SettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.AbortWithError(c, http.StatusBadRequest, "INVALID_CHECKIN_SETTINGS", "invalid check-in settings request")
		return
	}
	settings, err := h.service.UpdateSettings(c.Request.Context(), request)
	if err != nil {
		writeCheckInError(c, err)
		return
	}
	response.Success(c, toSettingsResponse(settings))
}

func authenticatedUserID(c *gin.Context) (int64, bool) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		middleware.AbortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return 0, false
	}
	return subject.UserID, true
}

func writeCheckInError(c *gin.Context, err error) {
	var validationError *SettingsValidationError
	switch {
	case errors.As(err, &validationError):
		middleware.AbortWithError(c, http.StatusBadRequest, "INVALID_CHECKIN_SETTINGS", validationError.Error())
	case errors.Is(err, ErrDisabled):
		middleware.AbortWithError(c, http.StatusForbidden, "CHECKIN_DISABLED", "daily check-in is disabled")
	case errors.Is(err, ErrUserInactive):
		middleware.AbortWithError(c, http.StatusForbidden, "USER_INACTIVE", "user account is not active")
	case errors.Is(err, ErrUserNotFound):
		middleware.AbortWithError(c, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	default:
		middleware.AbortWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
