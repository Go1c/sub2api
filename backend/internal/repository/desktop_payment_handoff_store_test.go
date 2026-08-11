package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestDesktopPaymentHandoffStoreConsumesOnceAndExpires(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	store := NewDesktopPaymentHandoffStore(rdb)
	ctx := context.Background()

	require.NoError(t, store.Store(ctx, "hash-only", service.DesktopPaymentHandoffData{UserID: 42}, time.Minute))
	require.True(t, mr.Exists("desktop_payment_handoff:hash-only"))
	stored, err := mr.Get("desktop_payment_handoff:hash-only")
	require.NoError(t, err)
	require.Equal(t, `{"user_id":42}`, stored)
	require.Equal(t, time.Minute, mr.TTL("desktop_payment_handoff:hash-only"))

	got, err := store.Consume(ctx, "hash-only")
	require.NoError(t, err)
	require.Equal(t, int64(42), got.UserID)
	_, err = store.Consume(ctx, "hash-only")
	require.ErrorIs(t, err, service.ErrDesktopPaymentHandoffInvalid)

	require.NoError(t, store.Store(ctx, "expires", service.DesktopPaymentHandoffData{UserID: 7}, time.Minute))
	mr.FastForward(time.Minute)
	_, err = store.Consume(ctx, "expires")
	require.ErrorIs(t, err, service.ErrDesktopPaymentHandoffInvalid)
}

func TestDesktopPaymentHandoffStoreConsumeIsAtomicAcrossInstances(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	first := NewDesktopPaymentHandoffStore(rdb)
	second := NewDesktopPaymentHandoffStore(rdb)
	ctx := context.Background()
	require.NoError(t, first.Store(ctx, "race", service.DesktopPaymentHandoffData{UserID: 91}, time.Minute))

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, store := range []service.DesktopPaymentHandoffStore{first, second} {
		wg.Add(1)
		go func(candidate service.DesktopPaymentHandoffStore) {
			defer wg.Done()
			<-start
			_, err := candidate.Consume(ctx, "race")
			errs <- err
		}(store)
	}
	close(start)
	wg.Wait()
	close(errs)

	var successes, invalid int
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, service.ErrDesktopPaymentHandoffInvalid):
			invalid++
		default:
			t.Fatalf("unexpected consume error: %v", err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, invalid)
}
