package service

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
)

const (
	RequestHealthSlotOK   = "ok"
	RequestHealthSlotFail = "fail"

	RequestHealthModeSingle  = "single"
	RequestHealthModeIPGroup = "ip_group"

	RequestHealthMaxEvents       = 20
	DefaultRequestHealthWindow   = 12
	requestHealthMessageMaxRunes = 512
	requestHealthTTL             = 7 * 24 * time.Hour
	requestHealthBatchLimit      = 200
)

type RequestHealthEvent struct {
	Slot       string    `json:"slot"`
	AccountID  int64     `json:"account_id"`
	ProxyID    int64     `json:"proxy_id,omitempty"`
	StatusCode int       `json:"status_code,omitempty"`
	Message    string    `json:"message,omitempty"`
	Model      string    `json:"model,omitempty"`
	Endpoint   string    `json:"endpoint,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

type RequestHealthRecordInput struct {
	AccountID  int64
	ProxyID    int64
	Slot       string
	StatusCode int
	Message    string
	Model      string
	Endpoint   string
	OccurredAt time.Time
}

type RequestHealthOutcomeDTO struct {
	Slot       string    `json:"slot"`
	StatusCode int       `json:"status_code,omitempty"`
	Message    string    `json:"message,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
	Model      string    `json:"model,omitempty"`
	Endpoint   string    `json:"endpoint,omitempty"`
}

type AccountRequestHealthLineDTO struct {
	ProxyID       int64                     `json:"proxy_id,omitempty"`
	IP            string                    `json:"ip"`
	Outcomes      []RequestHealthOutcomeDTO `json:"outcomes"`
	Current       int                       `json:"current"`
	Max           int                       `json:"max"`
	CooldownUntil *time.Time                `json:"cooldown_until,omitempty"`
	RateLimited   bool                      `json:"rate_limited"`
	Overloaded    bool                      `json:"overloaded"`
}

type AccountRequestHealthDTO struct {
	AccountID   int64                         `json:"account_id"`
	Mode        string                        `json:"mode"`
	IPGroupName string                        `json:"ip_group_name,omitempty"`
	Lines       []AccountRequestHealthLineDTO `json:"lines"`
}

type RequestHealthStore interface {
	Append(ctx context.Context, ev RequestHealthEvent) error
	List(ctx context.Context, accountID, proxyID int64, limit int) ([]RequestHealthEvent, error)
	Runtime(ctx context.Context, accountID, proxyID int64) (int, *time.Time)
}

type RequestHealthDirectory interface {
	GetAccountsByIDs(ctx context.Context, ids []int64) ([]*Account, error)
	ListProxyIPGroups(ctx context.Context) ([]ProxyIPGroup, error)
	GetProxiesByIDs(ctx context.Context, ids []int64) ([]Proxy, error)
}

func ClampRequestHealthWindow(n int) int {
	switch n {
	case 8, 12, 16, 20:
		return n
	default:
		return DefaultRequestHealthWindow
	}
}

func MaskRequestHealthHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "—"
	}
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	if at := strings.LastIndex(host, "@"); at >= 0 {
		host = host[at+1:]
	}
	if cut := strings.IndexAny(host, "/?"); cut >= 0 {
		host = host[:cut]
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			return fmt.Sprintf("%d.%d.*.*", ip4[0], ip4[1])
		}
	}
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		return parts[0] + ".***." + parts[len(parts)-1]
	}
	return host
}

func EgressProxyIDFrom(ctx context.Context, account *Account) int64 {
	if ctx != nil {
		if id, ok := ctx.Value(ctxkey.EgressProxyID).(int64); ok && id > 0 {
			return id
		}
		if gc, ok := ctx.(*gin.Context); ok {
			if v, exists := gc.Get(string(ctxkey.EgressProxyID)); exists {
				if id, ok := v.(int64); ok && id > 0 {
					return id
				}
			}
		}
	}
	if account != nil && account.ProxyID != nil && *account.ProxyID > 0 {
		return *account.ProxyID
	}
	return 0
}

func RememberEgressProxyID(c *gin.Context, proxyID int64) {
	if c == nil || proxyID <= 0 {
		return
	}
	c.Set(string(ctxkey.EgressProxyID), proxyID)
	if c.Request != nil {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.EgressProxyID, proxyID))
	}
}

func recordRequestHealthOK(svc *AccountRequestHealthService, ctx context.Context, account *Account, endpoint, model string) {
	if svc == nil || account == nil || account.ID <= 0 {
		return
	}
	svc.Record(ctx, RequestHealthRecordInput{
		AccountID: account.ID,
		ProxyID:   EgressProxyIDFrom(ctx, account),
		Slot:      RequestHealthSlotOK,
		Model:     model,
		Endpoint:  endpoint,
	})
}

func truncateRequestHealthMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	runes := []rune(message)
	if len(runes) <= requestHealthMessageMaxRunes {
		return message
	}
	return string(runes[:requestHealthMessageMaxRunes])
}

func (in RequestHealthRecordInput) toEvent() RequestHealthEvent {
	occurredAt := in.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	slot := RequestHealthSlotOK
	if in.Slot == RequestHealthSlotFail {
		slot = RequestHealthSlotFail
	}
	return RequestHealthEvent{
		Slot:       slot,
		AccountID:  in.AccountID,
		ProxyID:    in.ProxyID,
		StatusCode: in.StatusCode,
		Message:    truncateRequestHealthMessage(in.Message),
		Model:      strings.TrimSpace(in.Model),
		Endpoint:   strings.TrimSpace(in.Endpoint),
		OccurredAt: occurredAt,
	}
}
