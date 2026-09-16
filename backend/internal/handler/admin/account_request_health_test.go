package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type requestHealthDirStub struct {
	accounts []*service.Account
}

func (s *requestHealthDirStub) GetAccountsByIDs(_ context.Context, ids []int64) ([]*service.Account, error) {
	want := map[int64]struct{}{}
	for _, id := range ids {
		want[id] = struct{}{}
	}
	out := make([]*service.Account, 0, len(ids))
	for _, acc := range s.accounts {
		if acc == nil {
			continue
		}
		if _, ok := want[acc.ID]; ok {
			out = append(out, acc)
		}
	}
	return out, nil
}

func (s *requestHealthDirStub) ListProxyIPGroups(context.Context) ([]service.ProxyIPGroup, error) {
	return nil, nil
}

func (s *requestHealthDirStub) GetProxiesByIDs(context.Context, []int64) ([]service.Proxy, error) {
	return nil, nil
}

func TestGetBatchRequestHealthEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.POST("/admin/accounts/request-health/batch", handler.GetBatchRequestHealth)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/admin/accounts/request-health/batch", bytes.NewBufferString(`{"account_ids":[]}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var payload struct {
		Data struct {
			Items []any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Empty(t, payload.Data.Items)
}

func TestGetBatchRequestHealthReturnsRecordedEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mem := newHandlerHealthStore()
	svc := service.NewAccountRequestHealthService(mem, &requestHealthDirStub{
		accounts: []*service.Account{{ID: 8, Name: "acct", Concurrency: 2}},
	})
	require.NoError(t, mem.Append(context.Background(), service.RequestHealthEvent{
		AccountID: 8,
		Slot:      service.RequestHealthSlotOK,
	}))
	handler := NewAccountHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.SetRequestHealthService(svc)

	router := gin.New()
	router.POST("/admin/accounts/request-health/batch", handler.GetBatchRequestHealth)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/admin/accounts/request-health/batch", bytes.NewBufferString(`{"account_ids":[8],"window":8}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var payload struct {
		Data struct {
			Items []service.AccountRequestHealthDTO `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Len(t, payload.Data.Items, 1)
	require.Equal(t, int64(8), payload.Data.Items[0].AccountID)
	require.Equal(t, "single", payload.Data.Items[0].Mode)
	require.Len(t, payload.Data.Items[0].Lines, 1)
	require.Equal(t, "ok", payload.Data.Items[0].Lines[0].Outcomes[0].Slot)
}

type handlerHealthStore struct {
	events []service.RequestHealthEvent
}

func newHandlerHealthStore() *handlerHealthStore {
	return &handlerHealthStore{}
}

func (s *handlerHealthStore) Append(_ context.Context, ev service.RequestHealthEvent) error {
	s.events = append(s.events, ev)
	return nil
}

func (s *handlerHealthStore) List(_ context.Context, accountID, _ int64, limit int) ([]service.RequestHealthEvent, error) {
	out := make([]service.RequestHealthEvent, 0)
	for _, ev := range s.events {
		if ev.AccountID == accountID {
			out = append(out, ev)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func (s *handlerHealthStore) Runtime(context.Context, int64, int64) (int, *time.Time) {
	return 0, nil
}
