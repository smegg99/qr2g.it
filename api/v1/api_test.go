// api/v1/api_test.go

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"

	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/automation"
	"smegg.me/qr2g.it/core/datastore"
	"smegg.me/qr2g.it/core/github"
	"smegg.me/qr2g.it/core/policy"
	"smegg.me/qr2g.it/core/repositories"
	"smegg.me/qr2g.it/core/stats"
	"smegg.me/qr2g.it/core/turnstile"
	"smegg.me/qr2g.it/core/verifier"
)

// fakeRepo is one repository the fake GitHub serves.
type fakeRepo struct {
	ID          int64
	FullName    string
	Description string
	Language    string
	Files       []string
	Readme      string
	Gone        bool // answers 404
	Broken      bool // README and contents answer 500
	Limited     bool // answers GitHub's rate-limit 403
}

type fakeGitHub struct {
	mu    sync.Mutex
	repos map[int64]*fakeRepo
	calls int
}

func (f *fakeGitHub) find(fullName string) *fakeRepo {
	for _, r := range f.repos {
		if strings.EqualFold(r.FullName, fullName) {
			return r
		}
	}
	return nil
}

func (f *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if r.URL.Path == "/search/repositories" {
		w.Header().Set("X-RateLimit-Resource", "search")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))
		_ = json.NewEncoder(w).Encode(map[string]any{"total_count": 276430654, "incomplete_results": true, "items": []any{}})
		return
	}
	var repo *fakeRepo
	switch {
	case parts[0] == "repositories":
		id, _ := strconv.ParseInt(parts[1], 10, 64)
		repo = f.repos[id]
		parts = parts[2:]
	case parts[0] == "repos" && len(parts) >= 3:
		repo = f.find(parts[1] + "/" + parts[2])
		parts = parts[3:]
	}
	if repo == nil || repo.Gone {
		http.NotFound(w, r)
		return
	}
	if repo.Limited {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(90*time.Second).Unix(), 10))
		http.Error(w, "rate limited", http.StatusForbidden)
		return
	}
	if len(parts) > 0 && repo.Broken {
		http.Error(w, "boom", http.StatusInternalServerError)
		return
	}
	switch {
	case len(parts) == 0:
		_ = json.NewEncoder(w).Encode(map[string]any{
			"node_id": fmt.Sprintf("R_%d", repo.ID), "id": repo.ID, "full_name": repo.FullName,
			"html_url": "https://github.com/" + repo.FullName, "description": repo.Description,
			"language": repo.Language, "default_branch": "main", "stargazers_count": 42,
		})
	case parts[0] == "contents":
		entries := []map[string]string{{"name": "README.md", "type": "file"}}
		for _, name := range repo.Files {
			entries = append(entries, map[string]string{"name": name, "type": "file"})
		}
		_ = json.NewEncoder(w).Encode(entries)
	case parts[0] == "languages":
		_ = json.NewEncoder(w).Encode(map[string]int64{repo.Language: 9000, "Shell": 1000})
	case parts[0] == "readme":
		if repo.Readme == "" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, repo.Readme)
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeGitHub) set(id int64, change func(*fakeRepo)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	change(f.repos[id])
}

type env struct {
	server     *Server
	gh         *fakeGitHub
	links      *repositories.LinkRepository
	stats      *repositories.StatsRepository
	recorder   *stats.Recorder
	checker    *policy.Checker
	automation *automation.Service
}

var shared *env

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "qr2git-api-")
	if err != nil {
		panic(err)
	}
	code := func() int {
		defer os.RemoveAll(dir)
		shared = setup(dir)
		return m.Run()
	}()
	os.Exit(code)
}

func setup(dir string) *env {
	logger.Initialize()
	gh := &fakeGitHub{repos: map[int64]*fakeRepo{
		1: {ID: 1, FullName: "smegg99/Goptivum", Description: "Plan zajęć dla szkół", Language: "Go"},
		2: {ID: 2, FullName: "octo/hidden", Description: "Secret plans", Language: "Rust", Files: []string{policy.FileNoIndex}},
		3: {ID: 3, FullName: "octo/quiet", Language: "Go", Readme: "# Quiet\n<!-- qr2git: noanalytics -->\n"},
		4: {ID: 4, FullName: "octo/docs", Language: "TypeScript", Readme: "Example:\n```\n<!-- qr2git: noindex -->\n```\n"},
		5: {ID: 5, FullName: "octo/vanishing"},
		6: {ID: 6, FullName: "octo/batch"},
		7: {ID: 7, FullName: "octo/race"},
		8: {ID: 8, FullName: "octo/limited", Limited: true},
	}}
	ghServer := httptest.NewServer(gh)
	siteverify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		token := r.Form.Get("response")
		action := strings.TrimPrefix(token, "good-")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": strings.HasPrefix(token, "good-"), "action": action, "hostname": "qr2g.it"})
	}))

	c := config.Get()
	*c = config.Config{}
	c.Server.PublicBaseURL = "https://qr2g.it"
	c.Server.CanonicalBaseURL = "https://qr2g.it"
	c.Server.GinMode = "test"
	c.Server.MaxBodyBytes = 4096
	c.Server.TrustedProxyMode = "none"
	c.Db.Path = filepath.Join(dir, "api.sqlite")
	c.Db.AutoMigrate = true
	c.Github = config.GithubConfig{RESTBaseURL: ghServer.URL, UserAgent: "test", APIVersion: "2026-03-10", RequestTimeoutSeconds: 5}
	c.Turnstile = config.TurnstileConfig{Enabled: true, SecretKey: "real-secret", SiteverifyURL: siteverify.URL, RequestTimeoutSeconds: 5}
	c.Verifier = config.VerifierConfig{FreshTTLSeconds: 3600, MaxFailCount: 1, BackoffBaseSeconds: 1, BackoffMaxSeconds: 1, RequestTimeoutSeconds: 5, BatchSize: 10, Concurrency: 1, IntervalSeconds: 60}
	c.Ratelimit = config.RateLimitConfig{CreatePerMin: 1000, CreatePerDay: 1000, ResolvePerMin: 1000, MissPerMin: 1000,
		ShorthandPerHour: 1000, GithubPerSec: 1000, GithubBurst: 1000, CleanupSeconds: 600,
		AssetPerMin: 1000, ReadPerMin: 1000, RefreshPerHour: 1000, ContactPerMin: 1000}
	c.Contact = config.ContactConfig{Email: "main@example.test", Catchall: "catchall@example.test"}

	ctx := context.Background()
	must(datastore.Initialize(ctx))
	must(github.Initialize())
	must(turnstile.Initialize())
	must(verifier.Initialize())
	db := datastore.Get()
	links := repositories.NewLinkRepository(db)
	statsRepo := repositories.NewStatsRepository(db)
	recorder := stats.NewRecorder(statsRepo, 0)
	checker := policy.NewChecker(policy.Config{FreshTTL: 86400e9, FailureBackoff: 60e9, RefreshCooldown: 300e9}, github.Get(), verifier.Get(), links)
	tokens := automation.NewService(repositories.NewAutomationRepository(db))
	server, err := New(ctx, Deps{Links: links, Stats: statsRepo, Automation: tokens, Policy: checker, Recorder: recorder,
		Verifier: verifier.Get(), GitHub: github.Get(), Turnstile: turnstile.Get(), Ready: datastore.Ping, Version: "test"})
	must(err)
	return &env{server: server, gh: gh, links: links, stats: statsRepo, recorder: recorder, checker: checker, automation: tokens}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

type reply struct {
	*httptest.ResponseRecorder
}

func (r reply) json(t *testing.T) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", r.Body.String(), err)
	}
	return out
}

func do(method, path, body string, headers ...string) reply {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	shared.server.Engine().ServeHTTP(rec, req)
	return reply{rec}
}

func create(repo, token string) reply {
	return do(http.MethodPost, "/api/v1/links", fmt.Sprintf(`{"url":"https://github.com/%s","turnstileToken":%q}`, repo, token))
}

const browser = "Mozilla/5.0 (X11; Linux x86_64) Gecko/20100101 Firefox/140.0"

func expect(t *testing.T, r reply, status int, code string) {
	t.Helper()
	if r.Code != status {
		t.Fatalf("status %d, want %d: %s", r.Code, status, r.Body.String())
	}
	if code != "" {
		if got := r.json(t)["code"]; got != code {
			t.Fatalf("code %v, want %s", got, code)
		}
	}
}

func slugOf(t *testing.T, r reply) string {
	t.Helper()
	s, _ := r.json(t)["slug"].(string)
	if s == "" {
		t.Fatalf("no slug in %s", r.Body.String())
	}
	return s
}

func linkID(t *testing.T, slug string) int64 {
	t.Helper()
	id, err := strconv.ParseInt(slug, 36, 64)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func check(t *testing.T, slug string) {
	t.Helper()
	l, err := shared.links.GetByID(context.Background(), linkID(t, slug))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := shared.checker.Check(context.Background(), l); err != nil {
		t.Fatalf("policy check: %v", err)
	}
}

func TestCreateOutcomesAndValidation(t *testing.T) {
	expect(t, create("smegg99/Goptivum", ""), http.StatusForbidden, "turnstile_required")
	expect(t, create("smegg99/Goptivum", "bad"), http.StatusForbidden, "turnstile_failed")
	expect(t, create("smegg99/Goptivum", "good-contact"), http.StatusForbidden, "turnstile_failed")
	expect(t, do(http.MethodPost, "/api/v1/links", `{"url":`), http.StatusBadRequest, "invalid_body")
	expect(t, do(http.MethodPost, "/api/v1/links", `{"url":"https://gitlab.com/a/b"}`), http.StatusBadRequest, "invalid_url")
	expect(t, create("nobody/missing", "good-create_link"), http.StatusNotFound, "repo_not_found")
	limited := create("octo/limited", "good-create_link")
	expect(t, limited, http.StatusServiceUnavailable, "github_rate_limited")
	if after, _ := strconv.Atoi(limited.Header().Get("Retry-After")); after < 60 || after > 91 {
		t.Fatalf("GitHub rate limit Retry-After = %q", limited.Header().Get("Retry-After"))
	}

	first := create("smegg99/Goptivum", "good-create_link")
	expect(t, first, http.StatusCreated, "")
	again := create("SMEGG99/goptivum", "good-create_link")
	expect(t, again, http.StatusOK, "")
	if slugOf(t, first) != slugOf(t, again) {
		t.Fatal("repeated creation changed the slug")
	}
	// A throttled GitHub still lets a known repository reopen its link, but never invents a new one.
	shared.gh.set(1, func(r *fakeRepo) { r.Limited = true })
	throttled := create("smegg99/Goptivum", "good-create_link")
	shared.gh.set(1, func(r *fakeRepo) { r.Limited = false })
	expect(t, throttled, http.StatusOK, "")
	if slugOf(t, throttled) != slugOf(t, first) {
		t.Fatal("the throttled fallback returned another link")
	}
	body := first.json(t)
	if body["description"] != "Plan zajęć dla szkół" || body["language"] != "Go" || body["currentFullName"] != "smegg99/Goptivum" ||
		!strings.HasSuffix(body["createdAt"].(string), "T00:00:00Z") || body["githubNodeId"] != "R_1" {
		t.Fatalf("shipped link shape or metadata missing: %v", body)
	}
	if policy := body["policy"].(map[string]any); policy["listing"] != "pending" {
		t.Fatalf("new link listing = %v, want pending", policy["listing"])
	}
}

func TestConcurrentCreateKeepsOneCanonicalLink(t *testing.T) {
	var wg sync.WaitGroup
	results := make(chan reply, 12)
	for range 12 {
		wg.Go(func() { results <- create("octo/race", "good-create_link") })
	}
	wg.Wait()
	close(results)
	created, slugs := 0, map[string]bool{}
	for r := range results {
		if r.Code == http.StatusCreated {
			created++
		} else if r.Code != http.StatusOK {
			t.Fatalf("status %d: %s", r.Code, r.Body.String())
		}
		slugs[slugOf(t, r)] = true
	}
	if created != 1 || len(slugs) != 1 {
		t.Fatalf("created=%d slugs=%v, want one creation and one slug", created, slugs)
	}
}

func TestLookupIsAPureRead(t *testing.T) {
	sl := slugOf(t, create("smegg99/Goptivum", "good-create_link"))
	shared.gh.mu.Lock()
	before := shared.gh.calls
	shared.gh.mu.Unlock()
	got := do(http.MethodGet, "/api/v1/links/lookup?repository=smegg99/goptivum", "")
	expect(t, got, http.StatusOK, "")
	if slugOf(t, got) != sl {
		t.Fatal("lookup returned another link")
	}
	expect(t, do(http.MethodGet, "/api/v1/links/lookup?repository=nobody/none", ""), http.StatusNotFound, "link_not_found")
	expect(t, do(http.MethodGet, "/api/v1/links/lookup?repository=nonsense", ""), http.StatusBadRequest, "invalid_repository")
	shared.gh.mu.Lock()
	defer shared.gh.mu.Unlock()
	if shared.gh.calls != before {
		t.Fatalf("lookup called GitHub %d times", shared.gh.calls-before)
	}
}

func TestPolicyMarkersAndPublicSurfaces(t *testing.T) {
	listed := slugOf(t, create("smegg99/Goptivum", "good-create_link"))
	hidden := slugOf(t, create("octo/hidden", "good-create_link"))
	docs := slugOf(t, create("octo/docs", "good-create_link"))
	for _, s := range []string{listed, hidden, docs} {
		check(t, s)
	}

	expect(t, do(http.MethodGet, "/api/v1/repositories/"+hidden, ""), http.StatusNotFound, "not_listed")
	expect(t, do(http.MethodGet, "/api/v1/links/"+hidden+"/stats", ""), http.StatusNotFound, "not_listed")
	expect(t, do(http.MethodGet, "/api/v1/repositories/"+docs, ""), http.StatusOK, "")
	hiddenLink := do(http.MethodGet, "/api/v1/links/"+hidden, "")
	if got := hiddenLink.json(t)["policy"].(map[string]any)["listing"]; got != "owner_noindex" {
		t.Fatalf("hidden listing = %v", got)
	}
	if strings.Contains(hiddenLink.Body.String(), "Secret plans") || strings.Contains(hiddenLink.Body.String(), "Rust") {
		t.Fatalf("direct read exposes hidden metadata: %s", hiddenLink.Body.String())
	}

	page := do(http.MethodGet, "/api/v1/repositories?limit=50", "")
	expect(t, page, http.StatusOK, "")
	if strings.Contains(page.Body.String(), "octo/hidden") || !strings.Contains(page.Body.String(), "smegg99/Goptivum") {
		t.Fatalf("explore listing wrong: %s", page.Body.String())
	}
	search := do(http.MethodGet, "/api/v1/repositories?q=zaj%C4%99%C4%87&language=go", "")
	if !strings.Contains(search.Body.String(), "smegg99/Goptivum") || strings.Contains(search.Body.String(), "octo/docs") {
		t.Fatalf("search wrong: %s", search.Body.String())
	}
	if strings.Contains(do(http.MethodGet, "/api/v1/languages", "").Body.String(), "Rust") {
		t.Fatal("languages leak the hidden repository")
	}
	sitemap := do(http.MethodGet, "/sitemap.xml", "").Body.String()
	if strings.Contains(sitemap, "/explore") || !strings.Contains(sitemap, "/docs<") {
		t.Fatalf("sitemap wrong: %s", sitemap)
	}

	// A failed check keeps the restriction; a successful one without the marker restores the default.
	shared.gh.set(2, func(r *fakeRepo) { r.Files, r.Broken = nil, true })
	l, _ := shared.links.GetByID(context.Background(), linkID(t, hidden))
	if l, _ = shared.checker.Check(context.Background(), l); !l.NoIndex || !l.PolicyFailed {
		t.Fatalf("failure dropped the restriction: %+v", l)
	}
	shared.gh.set(2, func(r *fakeRepo) { r.Broken = false })
	check(t, hidden)
	expect(t, do(http.MethodGet, "/api/v1/repositories/"+hidden, ""), http.StatusOK, "")
}

func TestRedirectCountingFilters(t *testing.T) {
	sl := slugOf(t, create("smegg99/Goptivum", "good-create_link"))
	quiet := slugOf(t, create("octo/quiet", "good-create_link"))
	check(t, sl)
	check(t, quiet)
	ctx := context.Background()
	_ = shared.recorder.Flush(ctx)
	before, _ := shared.stats.Daily(ctx, linkID(t, sl), "2000-01-01")

	cases := []struct {
		method  string
		path    string
		headers []string
		status  int
	}{
		{http.MethodGet, "/" + sl, []string{"User-Agent", browser}, http.StatusFound},
		{http.MethodGet, "/" + sl, []string{"User-Agent", browser}, http.StatusFound},
		{http.MethodHead, "/" + sl, []string{"User-Agent", browser}, http.StatusFound},
		{http.MethodGet, "/" + sl, []string{"User-Agent", "Slackbot-LinkExpanding 1.0"}, http.StatusFound},
		{http.MethodGet, "/" + sl, []string{"User-Agent", "curl/8.0"}, http.StatusFound},
		{http.MethodGet, "/" + sl, []string{"User-Agent", browser, "Sec-Purpose", "prefetch"}, http.StatusFound},
		{http.MethodGet, "/" + sl, []string{"User-Agent", browser, "Authorization", "Bearer x"}, http.StatusFound},
		{http.MethodGet, "/smegg99/Goptivum", []string{"User-Agent", browser}, http.StatusFound},
		{http.MethodGet, "/" + sl + "+", []string{"User-Agent", browser}, http.StatusOK},
		{http.MethodGet, "/api/v1/links/" + sl + "/resolve", []string{"User-Agent", browser}, http.StatusOK},
		{http.MethodGet, "/api/v1/links/" + sl + "/asset", []string{"User-Agent", browser}, http.StatusOK},
		{http.MethodGet, "/" + quiet, []string{"User-Agent", browser}, http.StatusFound},
	}
	for _, c := range cases {
		if r := do(c.method, c.path, "", c.headers...); r.Code != c.status {
			t.Fatalf("%s %s %v: %d %s", c.method, c.path, c.headers, r.Code, r.Body.String())
		}
	}
	if err := shared.recorder.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	after, _ := shared.stats.Daily(ctx, linkID(t, sl), "2000-01-01")
	total := func(days []repositories.DayCount) (n int) {
		for _, d := range days {
			n += d.Redirects
		}
		return n
	}
	if got := total(after) - total(before); got != 2 {
		t.Fatalf("counted %d redirects, want 2", got)
	}
	if quietDays, _ := shared.stats.Daily(ctx, linkID(t, quiet), "2000-01-01"); total(quietDays) != 0 {
		t.Fatal("counted a repository with noanalytics")
	}
	statsBody := do(http.MethodGet, "/api/v1/links/"+quiet+"/stats", "").json(t)
	if statsBody["collecting"] != false || len(statsBody["pauses"].([]any)) != 1 {
		t.Fatalf("noanalytics stats: %v", statsBody)
	}
	linkStats := do(http.MethodGet, "/api/v1/links/"+sl+"/stats", "").json(t)
	if linkStats["total7"].(float64) < 2 || linkStats["total"].(float64) < 2 || linkStats["lastRedirectAt"] == nil {
		t.Fatalf("stats do not show the counted redirects: %v", linkStats)
	}

	top := do(http.MethodGet, "/api/v1/repositories/top", "")
	items, ok := top.json(t)["items"].([]any)
	if !ok {
		t.Fatalf("top: %d %s", top.Code, top.Body.String())
	}
	var found map[string]any
	for _, it := range items {
		item := it.(map[string]any)
		if item["slug"] == quiet {
			t.Fatal("top lists a repository with noanalytics")
		}
		if item["slug"] == sl {
			found = item
		}
	}
	if top.Code != http.StatusOK || found == nil || found["total"].(float64) < 2 || found["lastRedirectAt"] == nil || len(found["days"].([]any)) == 0 {
		t.Fatalf("top: %d %s", top.Code, top.Body.String())
	}
	if one := do(http.MethodGet, "/api/v1/repositories/top?limit=1", "").json(t)["items"].([]any); len(one) != 1 {
		t.Fatalf("limit=1 returned %d items", len(one))
	}
}

func TestUnavailableRecoversWithTheSameSlug(t *testing.T) {
	sl := slugOf(t, create("octo/vanishing", "good-create_link"))
	ctx := context.Background()
	id := linkID(t, sl)
	due := func() {
		if _, err := datastore.Get().ExecContext(ctx, `UPDATE links SET verify_after = '2000-01-01 00:00:00.000' WHERE id = ?1`, id); err != nil {
			t.Fatal(err)
		}
	}
	shared.gh.set(5, func(r *fakeRepo) { r.Gone = true })
	due()
	expect(t, do(http.MethodGet, "/"+sl, "", "User-Agent", browser), http.StatusGone, "link_unavailable")
	body := do(http.MethodGet, "/api/v1/links/"+sl, "").json(t)
	if body["state"] != "unavailable" || body["currentFullName"] != nil || body["currentUrl"] != nil || body["githubNodeId"] != nil {
		t.Fatalf("unavailable link publishes details: %v", body)
	}
	expect(t, do(http.MethodGet, "/api/v1/links/"+sl+"/asset", ""), http.StatusGone, "link_unavailable")

	shared.gh.set(5, func(r *fakeRepo) { r.Gone = false })
	due()
	expect(t, do(http.MethodGet, "/"+sl, "", "User-Agent", browser), http.StatusFound, "")
	if again := create("octo/vanishing", "good-create_link"); slugOf(t, again) != sl {
		t.Fatal("recovery issued a new slug")
	}
}

func TestAssetsRenderAndDecode(t *testing.T) {
	sl := slugOf(t, create("smegg99/Goptivum", "good-create_link"))
	r := do(http.MethodGet, "/api/v1/links/"+sl+"/asset?preset=presentation&format=png&size=960&caption=Za%C5%BC%C3%B3%C5%82%C4%87&download=true", "")
	expect(t, r, http.StatusOK, "")
	if r.Header().Get("Content-Type") != "image/png" ||
		r.Header().Get("Content-Disposition") != `attachment; filename=Goptivum-presentation.png` ||
		r.Header().Get("X-Asset-Width") != "960" || r.Header().Get("X-Asset-Height") != "540" {
		t.Fatalf("headers: %v", r.Header())
	}
	img, _, err := image.Decode(bytes.NewReader(r.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	bmp, _ := gozxing.NewBinaryBitmapFromImage(img)
	decoded, err := qrcode.NewQRCodeReader().Decode(bmp, map[gozxing.DecodeHintType]any{gozxing.DecodeHintType_TRY_HARDER: true})
	if err != nil || decoded.GetText() != "https://qr2g.it/"+sl {
		t.Fatalf("decoded %v %v", decoded, err)
	}
	bad := do(http.MethodGet, "/api/v1/links/"+sl+"/asset?fg=%23ffffff&bg=%23ffffff", "")
	expect(t, bad, http.StatusBadRequest, "invalid_design")
	if details := bad.json(t)["details"].(map[string]any); details["foreground"] != "low_contrast" {
		t.Fatalf("details: %v", details)
	}
	expect(t, do(http.MethodGet, "/api/v1/links/"+sl+"/asset?margin=abc", ""), http.StatusBadRequest, "invalid_request")
	if r.Header().Get("Cache-Control") != "public, max-age=3600" || bad.Header().Get("Cache-Control") == "public, max-age=3600" {
		t.Fatalf("cache headers: ok=%q error=%q", r.Header().Get("Cache-Control"), bad.Header().Get("Cache-Control"))
	}
}

func TestAutomationTokensAndBatch(t *testing.T) {
	ctx := context.Background()
	raw, tok, err := shared.automation.Issue(ctx, "ci", 4)
	if err != nil {
		t.Fatal(err)
	}
	auth := []string{"Authorization", "Bearer " + raw}
	r := do(http.MethodPost, "/api/v1/links", `{"url":"https://github.com/smegg99/Goptivum"}`, auth...)
	if r.Code != http.StatusOK && r.Code != http.StatusCreated {
		t.Fatalf("token create: %d %s", r.Code, r.Body.String())
	}
	if r.Header().Get("X-RateLimit-Limit") != "4" || r.Header().Get("X-RateLimit-Remaining") != "3" {
		t.Fatalf("quota headers: %v", r.Header())
	}
	expect(t, do(http.MethodPost, "/api/v1/links/batch", `{"repositories":["octo/batch"]}`), http.StatusUnauthorized, "token_required")
	oversized := `{"repositories":[` + strings.TrimSuffix(strings.Repeat(`"a/b",`, 26), ",") + `]}`
	expect(t, do(http.MethodPost, "/api/v1/links/batch", oversized, auth...), http.StatusBadRequest, "invalid_body")
	if used, _ := shared.automation.UsedToday(ctx, tok); used != 1 {
		t.Fatalf("oversized batch spent quota: used %d", used)
	}
	batch := do(http.MethodPost, "/api/v1/links/batch", `{"repositories":["octo/batch","not a repo","nobody/missing"]}`, auth...)
	expect(t, batch, http.StatusOK, "")
	results := batch.json(t)["results"].([]any)
	outcomes := []string{}
	for _, item := range results {
		m := item.(map[string]any)
		code := ""
		if e, ok := m["error"].(map[string]any); ok {
			code = e["code"].(string)
		}
		outcomes = append(outcomes, m["outcome"].(string)+":"+code)
	}
	if strings.Join(outcomes, ",") != "created:,failed:invalid_repository,failed:repo_not_found" {
		t.Fatalf("batch outcomes %v", outcomes)
	}
	over := do(http.MethodGet, "/api/v1/links/lookup?repository=octo/batch", "", auth...)
	expect(t, over, http.StatusTooManyRequests, "quota_exceeded")
	if over.Header().Get("Retry-After") == "" {
		t.Fatal("quota refusal without Retry-After")
	}
	usage := do(http.MethodGet, "/api/v1/automation/usage", "", auth...)
	if usage.json(t)["usedToday"].(float64) != 4 {
		t.Fatalf("usage: %s", usage.Body.String())
	}
	must(shared.automation.Revoke(ctx, tok.ID))
	expect(t, do(http.MethodGet, "/api/v1/automation/usage", "", auth...), http.StatusUnauthorized, "token_invalid")
	expect(t, do(http.MethodGet, "/api/v1/automation/usage", "", "Authorization", "Bearer q2g_forged"), http.StatusUnauthorized, "token_invalid")
}

func TestRefreshCooldownAndModeration(t *testing.T) {
	sl := slugOf(t, create("octo/docs", "good-create_link"))
	if r := do(http.MethodPost, "/api/v1/links/"+sl+"/policy/refresh", ""); r.Code != http.StatusOK && r.Code != http.StatusTooManyRequests {
		t.Fatalf("refresh: %d %s", r.Code, r.Body.String())
	}
	limitedReply := do(http.MethodPost, "/api/v1/links/"+sl+"/policy/refresh", "")
	expect(t, limitedReply, http.StatusTooManyRequests, "rate_limited")
	if limitedReply.Header().Get("Retry-After") == "" {
		t.Fatal("cooldown without Retry-After")
	}

	ctx := context.Background()
	must(shared.links.SetModeration(ctx, linkID(t, sl), "remove", "owner request"))
	expect(t, do(http.MethodGet, "/"+sl, "", "User-Agent", browser), http.StatusGone, "link_unavailable")
	expect(t, create("octo/docs", "good-create_link"), http.StatusGone, "link_unavailable")
	expect(t, do(http.MethodGet, "/api/v1/repositories/"+sl, ""), http.StatusNotFound, "not_listed")
	must(shared.links.SetModeration(ctx, linkID(t, sl), "", ""))
}

func TestContactReveal(t *testing.T) {
	if got := do(http.MethodPost, "/api/v1/contact/reveal", `{"turnstileToken":"good-contact"}`).json(t)["email"]; got != "main@example.test" {
		t.Fatalf("verified got %v", got)
	}
	for _, body := range []string{`{"turnstileToken":"bad"}`, `{}`, `{"turnstileToken":"good-create_link"}`} {
		if got := do(http.MethodPost, "/api/v1/contact/reveal", body).json(t)["email"]; got != "catchall@example.test" {
			t.Fatalf("%s got %v", body, got)
		}
	}
	saved := shared.server.contactCatchall
	shared.server.contactCatchall = ""
	defer func() { shared.server.contactCatchall = saved }()
	expect(t, do(http.MethodPost, "/api/v1/contact/reveal", `{}`), http.StatusServiceUnavailable, "contact_unavailable")
}

func TestLinkCarriesLanguageShares(t *testing.T) {
	body := create("smegg99/Goptivum", "good-create_link").json(t)
	languages, ok := body["languages"].([]any)
	if !ok || len(languages) != 2 {
		t.Fatalf("languages: %v", body["languages"])
	}
	first := languages[0].(map[string]any)
	if first["name"] != body["language"] || first["share"].(float64) != 0.9 {
		t.Fatalf("first language: %v", first)
	}
	if body["stars"] != 42.0 {
		t.Fatalf("stars: %v", body["stars"])
	}
}

func TestCoverage(t *testing.T) {
	slugOf(t, create("smegg99/Goptivum", "good-create_link"))
	// The request answers at once with the mapped count; GitHub's total arrives from the background.
	first := do(http.MethodGet, "/api/v1/coverage", "").json(t)
	if first["mapped"].(float64) < 1 {
		t.Fatalf("coverage: %v", first)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		body := do(http.MethodGet, "/api/v1/coverage", "").json(t)
		if total, ok := body["publicRepositories"].(float64); ok {
			if total != 276430654 || body["countedAt"] == nil {
				t.Fatalf("coverage: %v", body)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("GitHub's total never arrived: %v", body)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestRoutingBoundaries(t *testing.T) {
	expect(t, do(http.MethodGet, "/api/v1/nothing", ""), http.StatusNotFound, "not_found")
	expect(t, do(http.MethodGet, "/ready", ""), http.StatusOK, "")
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		if r := do(method, "/docs", "", "Accept", "text/html"); r.Code != http.StatusOK {
			t.Fatalf("%s /docs entered the dispatcher: %d", method, r.Code)
		}
	}
	if r := do(http.MethodGet, "/api/v1/openapi.yaml", ""); r.Code != http.StatusOK || !strings.Contains(r.Body.String(), "operationId: renderLinkAsset") {
		t.Fatalf("contract not served: %d", r.Code)
	}
	if r := do(http.MethodGet, "/smegg99/Goptivum", "", "User-Agent", "curl/8.0"); r.Code != http.StatusOK || !strings.HasPrefix(r.Body.String(), "https://qr2g.it/") {
		t.Fatalf("plain-text shorthand: %d %s", r.Code, r.Body.String())
	}
	qr := do(http.MethodGet, "/smegg99/Goptivum?qr", "", "User-Agent", "curl/8.0")
	lines := strings.Split(strings.TrimSuffix(qr.Body.String(), "\n"), "\n")
	if qr.Code != http.StatusOK || len(lines) < 10 || !strings.Contains(lines[0], "█") || !strings.HasPrefix(lines[len(lines)-1], "https://qr2g.it/") {
		t.Fatalf("terminal QR: %d %s", qr.Code, qr.Body.String())
	}
	if inv := do(http.MethodGet, "/smegg99/Goptivum?qr=invert", "", "User-Agent", "curl/8.0").Body.String(); inv == qr.Body.String() || !strings.HasPrefix(inv, " ") {
		t.Fatal("inverted terminal QR is not inverted")
	}
}
