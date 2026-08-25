//go:build unit

package telemetry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRecordServerAuthEventUsesStoredAttribution(t *testing.T) {
	repo := newMemoryRepository()
	userID := int64(9)
	require.NoError(t, repo.UpsertAttribution(t.Context(), AccountAttribution{
		UserID:             userID,
		FirstTouchSource:   "sb.sb",
		FirstTouchMedium:   "referral",
		FirstTouchCampaign: "t1299",
		FirstAttributionID: "bc_storedattribution1",
		LastTouchSource:    "sb.sb",
		LastTouchMedium:    "referral",
		LastTouchCampaign:  "t1299",
		LastAttributionID:  "bc_storedattribution1",
		FirstTouchAt:       testNow(),
		LastTouchAt:        testNow(),
	}))
	svc := NewService(repo, testNow)
	require.NoError(t, svc.RecordServerAuthEvent(t.Context(), userID, EventAuthLoginSuccess))
	require.Len(t, repo.events, 1)
	require.Equal(t, IngestSourceServer, repo.events[0].IngestSource)
	require.Equal(t, "t1299", repo.events[0].FirstTouchCampaign)
	require.Equal(t, "bc_storedattribution1", repo.events[0].AttributionID)
	require.NotNil(t, repo.events[0].UserID)
	require.Equal(t, userID, *repo.events[0].UserID)
}

func TestRecordServerAuthEventFailureIsDuplicateAccepted(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo, testNow)
	require.NoError(t, svc.RecordServerAuthEvent(t.Context(), 3, EventAuthRegisterSuccess))
	require.NoError(t, svc.RecordServerAuthEvent(t.Context(), 3, EventAuthRegisterSuccess))
	require.Len(t, repo.events, 1)
	require.Equal(t, IngestSourceServer, repo.events[0].IngestSource)
}

func TestServerRegisterThenClientIngestMergesCampaign(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo, testNow)
	userID := int64(11)
	require.NoError(t, svc.RecordServerAuthEvent(t.Context(), userID, EventAuthRegisterSuccess))
	require.NoError(t, svc.Ingest(t.Context(), ingestRequest{
		Event: EventAuthRegisterSuccess,
		TS:    testNow().UnixMilli(),
		Props: map[string]string{
			"attribution_id":       "bc_t1299attributionxx",
			"first_touch_source":   "sb.sb",
			"first_touch_medium":   "referral",
			"first_touch_campaign": "t1299",
			"client_source":        "bestcodex_web",
		},
	}, userID))
	require.Len(t, repo.events, 1)
	require.Equal(t, IngestSourceServer, repo.events[0].IngestSource)
	require.Equal(t, "t1299", repo.events[0].FirstTouchCampaign)
	require.Equal(t, "bc_t1299attributionxx", repo.events[0].AttributionID)
	require.Equal(t, "bestcodex_web", repo.events[0].ClientSource)

	from := testNow().Add(-time.Minute).UnixMilli()
	to := testNow().Add(time.Minute).UnixMilli()
	stats, err := svc.Stats(t.Context(), StatsQuery{
		From:     from,
		To:       to,
		Campaign: "t1299",
		Event:    EventAuthRegisterSuccess,
	})
	require.NoError(t, err)
	require.Len(t, stats.Rows, 1)
	require.Equal(t, int64(1), stats.Rows[0].EventCount)
	require.Equal(t, int64(1), stats.Rows[0].UniqueAttributionIDs)
}
