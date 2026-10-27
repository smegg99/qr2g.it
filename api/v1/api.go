// api/v1/api.go

package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/common/messages"
	"smegg.me/qr2g.it/common/ratelimit"
	"smegg.me/qr2g.it/core/automation"
	"smegg.me/qr2g.it/core/github"
	"smegg.me/qr2g.it/core/policy"
	"smegg.me/qr2g.it/core/repositories"
	"smegg.me/qr2g.it/core/stats"
	"smegg.me/qr2g.it/core/turnstile"
	"smegg.me/qr2g.it/core/verifier"
)

// Deps are the domain services the HTTP layer calls.
type Deps struct {
	Links      *repositories.LinkRepository
	Stats      *repositories.StatsRepository
	Automation *automation.Service
	Policy     *policy.Checker
	Recorder   *stats.Recorder
	Verifier   *verifier.Verifier
	GitHub     *github.Client
	Turnstile  *turnstile.Verifier
	Ready      func(context.Context) error
	Version    string
}

// Server is the wired HTTP server.
type Server struct {
	Deps

	listenAddr                string
	publicBaseURL             string
	canonicalBaseURL          string
	trustedProxyMode          string
	maxBodyBytes              int64
	shutdownTimeout           time.Duration
	readHeaderTimeout         time.Duration
	writeTimeout              time.Duration
	idleTimeout               time.Duration
	requireShorthandTurnstile bool
	contactEmail              string
	contactCatchall           string
	// say renders server-rendered page text in the configured site language.
	say *i18n.Localizer
	// coverage caches GitHub's public repository count for GetCoverage.
	coverage coverageCache

	createBucket    *ratelimit.Bucket
	resolveBucket   *ratelimit.Bucket
	missBucket      *ratelimit.Bucket
	shorthandBucket *ratelimit.Bucket
	assetBucket     *ratelimit.Bucket
	readBucket      *ratelimit.Bucket
	refreshBucket   *ratelimit.Bucket
	contactBucket   *ratelimit.Bucket
	createDaily     *ratelimit.DailyCounter
	shorthandHourly *ratelimit.HourlyCounter
	bucketCleanup   time.Duration

	// base outlives requests for work they start, such as a first policy check.
	base   context.Context
	engine *gin.Engine
	http   *http.Server
}

func perMin(n int64) float64  { return float64(max(n, 1)) / 60.0 }
func perHour(n int64) float64 { return float64(max(n, 1)) / 3600.0 }
func burst(n int64) int       { return int(max(n, 1)) }

func seconds(n int64) time.Duration {
	if n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}

// New wires the HTTP server from configuration and domain services.
func New(base context.Context, deps Deps) (*Server, error) {
	if deps.Links == nil || deps.Stats == nil || deps.Automation == nil || deps.Policy == nil ||
		deps.Recorder == nil || deps.Verifier == nil || deps.GitHub == nil || deps.Turnstile == nil {
		return nil, errors.New("api: missing dependency")
	}
	cfg := config.Get()
	s, rl := cfg.Server, cfg.Ratelimit

	gin.SetMode(string(s.GinMode))
	engine := gin.New()
	engine.RedirectTrailingSlash = false
	engine.RedirectFixedPath = false
	if err := engine.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("api: trusted proxies: %w", err)
	}
	engine.ForwardedByClientIP = false
	engine.UseRawPath = true
	engine.UnescapePathValues = true

	server := &Server{
		Deps:                      deps,
		listenAddr:                s.ListenAddr,
		publicBaseURL:             strings.TrimRight(s.PublicBaseURL, "/"),
		canonicalBaseURL:          strings.TrimRight(s.CanonicalBaseURL, "/"),
		trustedProxyMode:          string(s.TrustedProxyMode),
		maxBodyBytes:              s.MaxBodyBytes,
		shutdownTimeout:           seconds(s.ShutdownTimeoutSeconds),
		readHeaderTimeout:         seconds(s.ReadHeaderTimeoutSeconds),
		writeTimeout:              seconds(s.WriteTimeoutSeconds),
		idleTimeout:               seconds(s.IdleTimeoutSeconds),
		requireShorthandTurnstile: s.RequireShorthandTurnstile,
		contactEmail:              strings.TrimSpace(cfg.Contact.Email),
		contactCatchall:           strings.TrimSpace(cfg.Contact.Catchall),
		say:                       messages.Localizer(string(cfg.Preferences.Language)),

		createBucket:    ratelimit.NewBucket(perMin(rl.CreatePerMin), burst(rl.CreatePerMin), 30*time.Minute),
		resolveBucket:   ratelimit.NewBucket(perMin(rl.ResolvePerMin), burst(rl.ResolvePerMin), 10*time.Minute),
		missBucket:      ratelimit.NewBucket(perMin(rl.MissPerMin), burst(rl.MissPerMin), 10*time.Minute),
		shorthandBucket: ratelimit.NewBucket(perHour(rl.ShorthandPerHour), burst(rl.ShorthandPerHour), time.Hour),
		assetBucket:     ratelimit.NewBucket(perMin(rl.AssetPerMin), burst(rl.AssetPerMin), 10*time.Minute),
		readBucket:      ratelimit.NewBucket(perMin(rl.ReadPerMin), burst(rl.ReadPerMin), 10*time.Minute),
		refreshBucket:   ratelimit.NewBucket(perHour(rl.RefreshPerHour), burst(rl.RefreshPerHour), time.Hour),
		contactBucket:   ratelimit.NewBucket(perMin(rl.ContactPerMin), burst(rl.ContactPerMin), 10*time.Minute),
		createDaily:     ratelimit.NewDailyCounter(int(rl.CreatePerDay)),
		shorthandHourly: ratelimit.NewHourlyCounter(int(rl.ShorthandPerHour)),
		bucketCleanup:   seconds(rl.CleanupSeconds),

		base:   base,
		engine: engine,
	}
	server.registerRoutes()
	logger.Info().Str("addr", server.listenAddr).Str("trusted_proxy_mode", server.trustedProxyMode).Msg("api initialised")
	return server, nil
}

// Engine exposes the underlying Gin engine for tests.
func (s *Server) Engine() *gin.Engine { return s.engine }

// Run starts the HTTP server and blocks until ctx is done. On shutdown it
// gracefully drains in-flight requests up to the configured timeout.
func (s *Server) Run(ctx context.Context) error {
	s.http = &http.Server{
		Addr:              s.listenAddr,
		Handler:           s.engine,
		ReadHeaderTimeout: s.readHeaderTimeout,
		WriteTimeout:      s.writeTimeout,
		IdleTimeout:       s.idleTimeout,
	}

	s.startBucketCleanup(ctx)
	// The home page's repository count is ready before the first visitor asks for it.
	s.coverageTotal()

	errCh := make(chan error, 1)
	go func() {
		logger.Info().Str("addr", s.listenAddr).Msg("http server listening")
		err := s.http.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		return s.shutdown()
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("http: %w", err)
		}
		return nil
	}
}

func (s *Server) shutdown() error {
	if s.http == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	logger.Info().Msg("http server shutting down")
	return s.http.Shutdown(ctx)
}

func (s *Server) startBucketCleanup(ctx context.Context) {
	interval := s.bucketCleanup
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	for _, b := range []*ratelimit.Bucket{
		s.createBucket, s.resolveBucket, s.missBucket, s.shorthandBucket,
		s.assetBucket, s.readBucket, s.refreshBucket, s.contactBucket,
	} {
		go b.RunCleanup(ctx, interval)
	}
}
