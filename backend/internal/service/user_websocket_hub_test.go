//go:build unit

package service

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockWSConn struct {
	mu     sync.Mutex
	msgs   []any
	err    error
	closed bool
}

func (m *mockWSConn) SendJSON(v any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.msgs = append(m.msgs, v)
	return nil
}

func (m *mockWSConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockWSConn) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.msgs)
}

type wsUserReaderStub struct {
	user *User
}

func (s *wsUserReaderStub) GetByID(context.Context, int64) (*User, error) {
	return s.user, nil
}

func TestUserWebsocketHub_Publish(t *testing.T) {
	hub := NewUserWebsocketHub()
	c1 := &mockWSConn{}
	c2 := &mockWSConn{}
	hub.Register(7, c1)
	hub.Register(7, c2)
	require.Equal(t, 2, hub.OnlineCount(7))

	n := hub.PublishJSON(7, UserWSEventTest, "t", "m", nil)
	require.Equal(t, 2, n)
	require.Equal(t, 1, c1.count())
	require.Equal(t, 1, c2.count())
}

func TestShouldPushWebsocketBalanceAlert(t *testing.T) {
	thr := 5.0
	u := &User{
		WebsocketNotifyEnabled:         true,
		WebsocketBalanceAlertEnabled:   true,
		WebsocketBalanceAlertThreshold: &thr,
	}
	require.True(t, ShouldPushWebsocketBalanceAlert(u))
	u.WebsocketNotifyEnabled = false
	require.False(t, ShouldPushWebsocketBalanceAlert(u))
}

func TestResolveWebsocketBalanceThreshold_Default10(t *testing.T) {
	require.Equal(t, 10.0, ResolveWebsocketBalanceThreshold(&User{}))
	v := 3.5
	require.Equal(t, 3.5, ResolveWebsocketBalanceThreshold(&User{WebsocketBalanceAlertThreshold: &v}))
}

func TestCheckWebsocketBalanceAfterDeduction_Crossing(t *testing.T) {
	hub := NewUserWebsocketHub()
	conn := &mockWSConn{}
	hub.Register(1, conn)
	svc := NewUserWebsocketNotifyService(hub, nil)
	u := &User{
		ID:                           1,
		WebsocketNotifyEnabled:       true,
		WebsocketBalanceAlertEnabled: true,
	}
	// 12 -> 9 crosses default 10
	svc.CheckWebsocketBalanceAfterDeduction(context.Background(), u, 12, 3)
	require.Equal(t, 1, conn.count())

	// already below, no second fire
	svc.CheckWebsocketBalanceAfterDeduction(context.Background(), u, 9, 1)
	require.Equal(t, 1, conn.count())
}

func TestNotifyTest_RequiresOnlineAndEnabled(t *testing.T) {
	hub := NewUserWebsocketHub()
	repo := &wsUserReaderStub{user: &User{ID: 1, WebsocketNotifyEnabled: false}}
	svc := NewUserWebsocketNotifyService(hub, repo)
	_, err := svc.NotifyTest(context.Background(), 1)
	require.ErrorIs(t, err, ErrUserWebsocketDisabled)

	repo.user.WebsocketNotifyEnabled = true
	_, err = svc.NotifyTest(context.Background(), 1)
	require.ErrorIs(t, err, ErrUserWebsocketNotConnected)

	conn := &mockWSConn{}
	hub.Register(1, conn)
	n, err := svc.NotifyTest(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, 1, n)
}
