// core/automation/automation_test.go

package automation

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"smegg.me/qr2g.it/core/datastore"
	"smegg.me/qr2g.it/core/repositories"
)

func TestTokenLifecycleAndConcurrentQuota(t *testing.T) {
	ctx := context.Background()
	db, err := datastore.Open(ctx, filepath.Join(t.TempDir(), "tokens.sqlite"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	svc := NewService(repositories.NewAutomationRepository(db))

	raw, issued, err := svc.Issue(ctx, "ci", 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(ctx, raw+"x"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("tampered token: %v", err)
	}
	tok, err := svc.Authenticate(ctx, raw)
	if err != nil || tok.ID != issued.ID {
		t.Fatalf("authenticate: %+v %v", tok, err)
	}

	var ok, denied atomic.Int32
	var wg sync.WaitGroup
	for range 25 {
		wg.Go(func() {
			_, err := svc.Consume(ctx, tok, 1)
			switch {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, ErrQuotaExceeded):
				denied.Add(1)
			default:
				t.Errorf("consume: %v", err)
			}
		})
	}
	wg.Wait()
	if ok.Load() != 10 || denied.Load() != 15 {
		t.Fatalf("ok=%d denied=%d, want 10 and 15", ok.Load(), denied.Load())
	}
	if used, err := svc.UsedToday(ctx, tok); err != nil || used != 10 {
		t.Fatalf("used = %d %v", used, err)
	}
	if _, err := svc.Consume(ctx, tok, 11); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("oversized charge: %v", err)
	}

	if err := svc.Revoke(ctx, tok.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(ctx, raw); !errors.Is(err, ErrInvalid) {
		t.Fatalf("revoked token: %v", err)
	}
	if err := svc.Revoke(ctx, tok.ID); !errors.Is(err, repositories.ErrNotFound) {
		t.Fatalf("second revoke: %v", err)
	}
}
