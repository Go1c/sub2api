//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type telemetryRecorderStub struct {
	calls []struct {
		userID int64
		event  string
	}
	err error
}

func (s *telemetryRecorderStub) RecordServerAuthEvent(_ context.Context, userID int64, event string) error {
	s.calls = append(s.calls, struct {
		userID int64
		event  string
	}{userID: userID, event: event})
	return s.err
}

func TestRecordSuccessfulLoginRecordsFirstPartyTelemetry(t *testing.T) {
	svc := newAuthService(&userRepoStub{}, nil, nil)
	rec := &telemetryRecorderStub{}
	svc.SetFirstPartyTelemetry(rec)

	svc.RecordSuccessfulLogin(context.Background(), 42)

	require.Len(t, rec.calls, 1)
	require.Equal(t, int64(42), rec.calls[0].userID)
	require.Equal(t, "auth_login_success", rec.calls[0].event)
}

func TestPostAuthUserBootstrapRecordsRegisterSuccessWhenTouchLogin(t *testing.T) {
	svc := newAuthService(&userRepoStub{}, nil, nil)
	rec := &telemetryRecorderStub{}
	svc.SetFirstPartyTelemetry(rec)

	svc.postAuthUserBootstrap(context.Background(), &User{ID: 7}, "email", true)

	require.Len(t, rec.calls, 1)
	require.Equal(t, int64(7), rec.calls[0].userID)
	require.Equal(t, "auth_register_success", rec.calls[0].event)
}

func TestPostAuthUserBootstrapSkipsTelemetryWhenNotTouchLogin(t *testing.T) {
	svc := newAuthService(&userRepoStub{}, nil, nil)
	rec := &telemetryRecorderStub{}
	svc.SetFirstPartyTelemetry(rec)

	svc.postAuthUserBootstrap(context.Background(), &User{ID: 7}, "oauth", false)

	require.Empty(t, rec.calls)
}

func TestRecordSuccessfulLoginTelemetryFailureDoesNotFailLogin(t *testing.T) {
	svc := newAuthService(&userRepoStub{}, nil, nil)
	rec := &telemetryRecorderStub{err: errors.New("db down")}
	svc.SetFirstPartyTelemetry(rec)

	require.NotPanics(t, func() {
		svc.RecordSuccessfulLogin(context.Background(), 42)
	})
	require.Len(t, rec.calls, 1)
}
