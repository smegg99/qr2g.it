// core/repositories/link_test.go
package repositories

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"smegg.me/qr2g.it/core/datastore"
	"smegg.me/qr2g.it/core/models"
)

func TestSQLiteLinkLifecycle(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "links.sqlite")
	db, err := datastore.Open(ctx, path, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	r := &LinkRepository{db: db}
	a := &AliasRepository{db: db}
	params := CreateLinkParams{NodeID: "node-old", DatabaseID: 1234, FullName: "owner/repo", URL: "https://github.com/owner/repo", FreshTTL: 3600}
	link, created, err := r.CreateOrGet(ctx, params)
	if err != nil || !created || link.ID != 1 {
		t.Fatalf("create: %+v %v %v", link, created, err)
	}
	if !link.IsFresh(time.Now()) || link.CreatedAt.IsZero() {
		t.Fatalf("timestamps: %+v", link)
	}
	duplicate, created, err := r.CreateOrGet(ctx, params)
	if err != nil || created || duplicate.ID != link.ID {
		t.Fatalf("duplicate: %+v %v %v", duplicate, created, err)
	}
	if _, err := r.GetByID(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if got, err := r.GetByCurrentFullName(ctx, "OWNER/REPO"); err != nil || got.ID != link.ID {
		t.Fatalf("name lookup: %+v %v", got, err)
	}
	link, err = r.ApplyTransientFailure(ctx, link.ID, FailureUpdate{Message: "timeout", BackoffSecs: 120})
	if err != nil || link.FailCount != 1 || link.LastError == nil || *link.LastError != "timeout" {
		t.Fatalf("failure: %+v %v", link, err)
	}
	link, err = r.ApplyTombstone(ctx, link.ID, TombstoneUpdate{State: models.StatePrivate, Reason: "private", Backoff: 120})
	if err != nil || link.State != models.StatePrivate || link.StateChangedAt == nil {
		t.Fatalf("tombstone: %+v %v", link, err)
	}
	link, err = r.ApplySuccessfulVerification(ctx, link.ID, VerificationUpdate{NodeID: "node-new", FullName: "new/renamed", URL: "https://github.com/new/renamed", FreshTTL: 3600, JitterRange: 60})
	if err != nil || link.State != models.StateActive || link.FailCount != 0 || link.LastError != nil || link.CurrentFullName != "new/renamed" {
		t.Fatalf("verify: %+v %v", link, err)
	}
	if remaining := time.Until(link.VerifyAfter); remaining < 3590*time.Second || remaining > 3660*time.Second {
		t.Fatalf("verification TTL: %s", remaining)
	}
	for _, node := range []string{"node-old", "node-new"} {
		got, err := r.GetByNodeID(ctx, node)
		if err != nil || got.ID != link.ID {
			t.Fatalf("node %s: %+v %v", node, got, err)
		}
		alias, err := a.GetByNodeID(ctx, node)
		if err != nil || alias.LinkID != link.ID || alias.FirstSeenAt.IsZero() {
			t.Fatalf("alias %s: %+v %v", node, alias, err)
		}
	}
	if err := a.Upsert(ctx, "orphan", 999); err == nil {
		t.Fatal("foreign key violation accepted")
	}
	if _, err := db.ExecContext(ctx, "UPDATE links SET state='invalid' WHERE id=?1", link.ID); err == nil {
		t.Fatal("invalid state accepted")
	}
	if n, err := r.CountActive(ctx); err != nil || n != 1 {
		t.Fatalf("count: %d %v", n, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = datastore.Open(ctx, path, true)
	if err != nil {
		t.Fatal(err)
	}
	r.db = db
	if got, err := r.GetByID(ctx, link.ID); err != nil || got.CurrentURL != link.CurrentURL {
		t.Fatalf("reopen: %+v %v", got, err)
	}
}

func TestSQLiteConcurrentCreateAndClaim(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "links.sqlite")
	db1, err := datastore.Open(ctx, path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer db1.Close()
	db2, err := datastore.Open(ctx, path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	repos := []*LinkRepository{{db: db1}, {db: db2}}
	var wg sync.WaitGroup
	for i := range 12 {
		wg.Go(func() {
			_, _, err := repos[i%2].CreateOrGet(ctx, CreateLinkParams{NodeID: "same", DatabaseID: 1, FullName: "a/b", URL: "https://github.com/a/b", FreshTTL: -10})
			if err != nil {
				t.Errorf("concurrent create: %v", err)
			}
		})
	}
	wg.Wait()
	if n, err := repos[0].CountActive(ctx); err != nil || n != 1 {
		t.Fatalf("count: %d %v", n, err)
	}
	claims := make(chan []models.Link, 2)
	for _, repo := range repos {
		wg.Go(func() {
			rows, err := repo.ClaimStale(ctx, 10, 120)
			if err != nil {
				t.Errorf("claim: %v", err)
			}
			claims <- rows
		})
	}
	wg.Wait()
	if total := len(<-claims) + len(<-claims); total != 1 {
		t.Fatalf("claimed %d links, want 1", total)
	}
}
