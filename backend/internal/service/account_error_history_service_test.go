//go:build unit

package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNormalizeAccountErrorMessage_FoldsVolatileParts(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
		same bool
	}{
		{
			name: "same error with different numbers/timestamps folds to one fingerprint",
			a:    "upstream returned 429 after 1234ms, request id req-123",
			b:    "upstream returned 503 after 9876ms, request id req-987",
			same: true,
		},
		{
			name: "uuid difference folds",
			a:    "session 550e8400-e29b-41d4-a716-446655440000 expired",
			b:    "session 550e8400-e29b-41d4-a716-999999999999 expired",
			same: true,
		},
		{
			name: "case and whitespace insensitive",
			a:    "Connection   Reset  by Peer",
			b:    "connection reset by peer",
			same: true,
		},
		{
			name: "genuinely different errors do not fold",
			a:    "rate limited by upstream",
			b:    "invalid api key provided",
			same: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fa := computeAccountErrorFingerprint(tc.a)
			fb := computeAccountErrorFingerprint(tc.b)
			if tc.same && fa != fb {
				t.Fatalf("expected same fingerprint, got %q vs %q", fa, fb)
			}
			if !tc.same && fa == fb {
				t.Fatalf("expected different fingerprints, both %q", fa)
			}
		})
	}
}

func TestComputeAccountErrorFingerprint_Length(t *testing.T) {
	fp := computeAccountErrorFingerprint("anything")
	if len(fp) != 16 {
		t.Fatalf("expected 16 hex chars, got %d (%q)", len(fp), fp)
	}
}

func TestNormalizeAccountErrorSource_Fallback(t *testing.T) {
	if got := normalizeAccountErrorSource("bogus"); got != AccountErrorSourceSchedule {
		t.Fatalf("unknown source should fall back to schedule, got %q", got)
	}
	if got := normalizeAccountErrorSource(AccountErrorSourceGateway); got != AccountErrorSourceGateway {
		t.Fatalf("valid source must pass through, got %q", got)
	}
}

// fakeAccountErrorRepo 记录最后一次 Record 调用，供异步写入断言。
type fakeAccountErrorRepo struct {
	mu   sync.Mutex
	rows []*AccountErrorRow
	done chan struct{}
}

func (f *fakeAccountErrorRepo) Record(ctx context.Context, row *AccountErrorRow) error {
	f.mu.Lock()
	f.rows = append(f.rows, row)
	f.mu.Unlock()
	if f.done != nil {
		close(f.done)
	}
	return nil
}

func (f *fakeAccountErrorRepo) ListRecent(ctx context.Context, accountID int64, limit int) ([]*AccountErrorHistoryEntry, error) {
	return nil, nil
}

func TestRecordAccountError_AsyncComputesFingerprintAndSnapshot(t *testing.T) {
	repo := &fakeAccountErrorRepo{done: make(chan struct{})}
	svc := NewAccountErrorHistoryService(repo)

	email := "user@example.com"
	svc.RecordAccountError(context.Background(), AccountErrorEvent{
		AccountID: 42,
		UserEmail: &email,
		Source:    AccountErrorSourceGateway,
		Message:   "upstream 500 error id 999",
	})

	select {
	case <-repo.done:
	case <-time.After(2 * time.Second):
		t.Fatal("Record was not invoked asynchronously within timeout")
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.rows) != 1 {
		t.Fatalf("expected 1 recorded row, got %d", len(repo.rows))
	}
	row := repo.rows[0]
	if row.AccountID != 42 {
		t.Fatalf("account id mismatch: %d", row.AccountID)
	}
	if row.Fingerprint == "" {
		t.Fatal("fingerprint should be computed by service")
	}
	if row.UserEmail == nil || *row.UserEmail != email {
		t.Fatal("user email snapshot not carried through")
	}
}

func TestRecordAccountError_IgnoresZeroAccount(t *testing.T) {
	repo := &fakeAccountErrorRepo{}
	svc := NewAccountErrorHistoryService(repo)
	svc.RecordAccountError(context.Background(), AccountErrorEvent{Message: "x"})
	// 给可能的 goroutine 一点时间；account_id=0 应被直接丢弃，不调用 repo。
	time.Sleep(50 * time.Millisecond)
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.rows) != 0 {
		t.Fatalf("zero account id should not be recorded, got %d rows", len(repo.rows))
	}
}
