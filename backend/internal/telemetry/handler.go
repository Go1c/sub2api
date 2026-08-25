package telemetry

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service  *Service
	identity AccessTokenIdentifier
}

func newHandler(service *Service, identity AccessTokenIdentifier) *Handler {
	return &Handler{service: service, identity: identity}
}

func (h *Handler) Ingest(c *gin.Context) {
	var req ingestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid json")
		return
	}
	if err := h.service.Ingest(c.Request.Context(), req, h.optionalUserID(c)); err != nil {
		if errors.Is(err, ErrUnknownEvent) {
			response.ErrorWithDetails(c, http.StatusBadRequest, "unknown event", "UNKNOWN_EVENT", nil)
			return
		}
		slog.Warn("telemetry ingest failed", "error", err)
		// Fail-open: ingest errors must never look like auth failures to the client.
		response.Success(c, gin.H{"accepted": true})
		return
	}
	response.Success(c, gin.H{"accepted": true})
}

func (h *Handler) Stats(c *gin.Context) {
	query, err := parseStatsQuery(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.Stats(c.Request.Context(), query)
	if err != nil {
		slog.Warn("telemetry stats failed", "error", err)
		response.InternalError(c, "telemetry stats failed")
		return
	}
	response.Success(c, result)
}

func (h *Handler) optionalUserID(c *gin.Context) int64 {
	header := c.GetHeader("Authorization")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return 0
	}
	token := strings.TrimSpace(parts[1])
	if token == "" || h.identity == nil {
		return 0
	}
	userID, err := h.identity.IdentifyAccessToken(token)
	if err != nil || userID <= 0 {
		return 0
	}
	return userID
}

func parseStatsQuery(c *gin.Context) (StatsQuery, error) {
	fromRaw := strings.TrimSpace(c.Query("from"))
	toRaw := strings.TrimSpace(c.Query("to"))
	if fromRaw == "" || toRaw == "" {
		return StatsQuery{}, ErrInvalidStats
	}
	from, err := strconv.ParseInt(fromRaw, 10, 64)
	if err != nil {
		return StatsQuery{}, ErrInvalidStats
	}
	to, err := strconv.ParseInt(toRaw, 10, 64)
	if err != nil {
		return StatsQuery{}, ErrInvalidStats
	}
	if from > to {
		return StatsQuery{}, ErrStatsRangeOrder
	}
	if to-from > maxStatsRange.Milliseconds() {
		return StatsQuery{}, ErrStatsRangeCap
	}
	return StatsQuery{
		From:         from,
		To:           to,
		ClientSource: strings.TrimSpace(c.Query("client_source")),
		Campaign:     strings.TrimSpace(c.Query("campaign")),
		Event:        strings.TrimSpace(c.Query("event")),
	}, nil
}
