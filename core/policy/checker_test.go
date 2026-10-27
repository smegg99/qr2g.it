// core/policy/checker_test.go

package policy

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"smegg.me/qr2g.it/core/datastore"
	"smegg.me/qr2g.it/core/github"
	"smegg.me/qr2g.it/core/models"
	"smegg.me/qr2g.it/core/repositories"
)

// fakeGitHub answers identity and marker reads for one repository.
type fakeGitHub struct {
	links    *repositories.LinkRepository
	fullName string
	gone     bool
	rootErr  error
	files    []string
	readme   string
	reads    []int64
	verifies int
}

func (f *fakeGitHub) VerifyOnce(ctx context.Context, l models.Link) (models.Link, error) {
	f.verifies++
	if f.gone {
		return f.links.ApplyTombstone(ctx, l.ID, repositories.TombstoneUpdate{State: models.StateUnavailable, Reason: "gone", Backoff: 60})
	}
	return f.links.ApplySuccessfulVerification(ctx, l.ID, repositories.VerificationUpdate{
		NodeID: l.GithubNodeID, FullName: f.fullName, URL: "https://github.com/" + f.fullName, FreshTTL: 60, JitterRange: 1,
	})
}

func (f *fakeGitHub) RootNames(_ context.Context, id int64) ([]string, error) {
	f.reads = append(f.reads, id)
	return f.files, f.rootErr
}

func (f *fakeGitHub) Readme(context.Context, int64) (string, error) { return f.readme, nil }
func (f *fakeGitHub) BackgroundAllowed(time.Time) bool              { return true }

func TestCheckerReadsByIdentityAndKeepsRestrictionsOnFailure(t *testing.T) {
	ctx := context.Background()
	db, err := datastore.Open(ctx, filepath.Join(t.TempDir(), "policy.sqlite"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	links := repositories.NewLinkRepository(db)
	link, _, err := links.CreateOrGet(ctx, repositories.CreateLinkParams{NodeID: "n", DatabaseID: 4242, FullName: "old/name", URL: "https://github.com/old/name", FreshTTL: 60})
	if err != nil {
		t.Fatal(err)
	}
	stale := func(l models.Link) models.Link {
		if _, err := db.ExecContext(ctx, `UPDATE links SET verify_after = '2000-01-01 00:00:00.000' WHERE id = ?1`, l.ID); err != nil {
			t.Fatal(err)
		}
		l, _ = links.GetByID(ctx, l.ID)
		return l
	}
	gh := &fakeGitHub{links: links, fullName: "new-owner/renamed", files: []string{FileNoIndex}}
	checker := NewChecker(Config{FreshTTL: time.Hour, FailureBackoff: time.Minute, RefreshCooldown: time.Hour}, gh, gh, links)

	l, err := checker.Check(ctx, stale(link))
	if err != nil || !l.NoIndex || l.PolicyCheckedAt == nil || l.CurrentFullName != "new-owner/renamed" || gh.reads[0] != 4242 {
		t.Fatalf("first check: %+v %v %v", l, err, gh.reads)
	}

	// A fresh verification is reused; markers are still read by database id, never by a stored name.
	verifies := gh.verifies
	gh.files = nil
	if l, err = checker.Check(ctx, l); err != nil || l.NoIndex || gh.verifies != verifies || gh.reads[1] != 4242 {
		t.Fatalf("fresh check: %+v %v %v", l, err, gh.reads)
	}

	// A 404 or a rate limit on the root listing is a failed check and keeps the restriction.
	gh.files = []string{FileNoIndex}
	if l, err = checker.Check(ctx, l); err != nil || !l.NoIndex {
		t.Fatalf("restricted: %+v %v", l, err)
	}
	for _, cause := range []error{github.ErrNotFound, github.ErrRateLimited} {
		gh.rootErr, gh.files = cause, nil
		if l, err = checker.Check(ctx, l); !errors.Is(err, cause) || !l.NoIndex || !l.PolicyFailed {
			t.Fatalf("%v dropped the restriction: %+v %v", cause, l, err)
		}
	}
	gh.rootErr = nil

	// Refresh is limited by the cooldown and always re-verifies identity.
	verifies = gh.verifies
	if _, err := checker.Refresh(ctx, l); err != nil || gh.verifies != verifies+1 {
		t.Fatalf("refresh: %v", err)
	}
	var cooldown *CooldownError
	if _, err := checker.Refresh(ctx, l); !errors.As(err, &cooldown) || cooldown.RetryAfter <= 0 {
		t.Fatalf("second refresh: %v", err)
	}

	// An unavailable repository is not a policy failure and keeps its slug.
	gh.gone = true
	if l, err = checker.Check(ctx, stale(l)); err != nil || l.IsActive() || l.ID != link.ID {
		t.Fatalf("unavailable: %+v %v", l, err)
	}
	// Recovery returns it unlisted until markers are confirmed again.
	gh.gone = false
	if l, err = links.ApplySuccessfulVerification(ctx, l.ID, repositories.VerificationUpdate{NodeID: "n", FullName: "new-owner/renamed", URL: "u", FreshTTL: 60, JitterRange: 1}); err != nil || l.Listed() || l.PolicyCheckedAt != nil {
		t.Fatalf("recovered link listed before a fresh check: %+v %v", l, err)
	}
}
