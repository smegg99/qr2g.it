// core/stats/recorder_test.go

package stats

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"smegg.me/qr2g.it/core/datastore"
	"smegg.me/qr2g.it/core/repositories"
)

type failingStore struct{ err error }

func (f failingStore) AddRedirects(context.Context, repositories.RedirectBatch) error { return f.err }

func TestRecorderConcurrentCountsReachSQLite(t *testing.T) {
	ctx := context.Background()
	db, err := datastore.Open(ctx, filepath.Join(t.TempDir(), "stats.sqlite"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	links := repositories.NewLinkRepository(db)
	link, _, err := links.CreateOrGet(ctx, repositories.CreateLinkParams{NodeID: "n", DatabaseID: 1, FullName: "a/b", URL: "https://github.com/a/b", FreshTTL: 60})
	if err != nil {
		t.Fatal(err)
	}
	store := repositories.NewStatsRepository(db)
	rec := NewRecorder(store, 0)

	var wg sync.WaitGroup
	for range 200 {
		wg.Go(func() { rec.Record(link.ID) })
	}
	for range 3 {
		wg.Go(func() {
			if err := rec.Flush(ctx); err != nil {
				t.Errorf("flush: %v", err)
			}
		})
	}
	wg.Wait()
	if err := rec.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	days, err := store.Daily(ctx, link.ID, "2000-01-01")
	if err != nil || len(days) != 1 || days[0].Redirects != 200 {
		t.Fatalf("daily = %+v %v, want one day with 200", days, err)
	}
	top, err := store.Top(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 0 {
		t.Fatalf("an unchecked link is not listed, got %+v", top)
	}
	var at string
	if err := db.QueryRowContext(ctx, `SELECT at FROM redirect_latest WHERE link_id = ?1`, link.ID).Scan(&at); err != nil || at == "" {
		t.Fatalf("latest redirect not stored: %q %v", at, err)
	}
}

func TestRecorderKeepsCountsAfterFailedFlush(t *testing.T) {
	rec := NewRecorder(failingStore{errors.New("disk full")}, 0)
	rec.Record(7)
	rec.Record(7)
	if err := rec.Flush(context.Background()); err == nil {
		t.Fatal("flush succeeded against a failing store")
	}
	var got int
	for k, n := range rec.pending {
		if k.LinkID == 7 {
			got += n
		}
	}
	if got != 2 || rec.latest[7].IsZero() {
		t.Fatalf("pending = %d, latest %v; want 2 and the time kept for retry", got, rec.latest[7])
	}
}
