// api/api.go
package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/common/ratelimit"
)

// Server is the wired HTTP server.
type Server struct {
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
	serviceName               string

	createBucket    *ratelimit.Bucket
	resolveBucket   *ratelimit.Bucket
	missBucket      *ratelimit.Bucket
	shorthandBucket *ratelimit.Bucket
	createDaily     *ratelimit.DailyCounter
	shorthandHourly *ratelimit.HourlyCounter
	bucketCleanup   time.Duration

	engine *gin.Engine
	http   *http.Server
}

var (
	instance *Server
	initOnce sync.Once
)

func perMin(n int) float64 {
	if n <= 0 {
		return 1
	}
	return float64(n) / 60.0
}

func perHour(n int) float64 {
	if n <= 0 {
		return 1
	}
	return float64(n) / 3600.0
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func seconds(n int64) time.Duration {
	if n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}

func Initialize() error {
	initOnce.Do(func() {
		s := config.Get().Server
		rl := config.Get().Ratelimit

		gin.SetMode(string(s.GinMode))
		engine := gin.New()
		engine.RedirectTrailingSlash = false
		engine.RedirectFixedPath = false
		_ = engine.SetTrustedProxies(nil)
		engine.ForwardedByClientIP = false
		engine.UseRawPath = true
		engine.UnescapePathValues = true

		instance = &Server{
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
			serviceName:               s.ServiceName,

			createBucket:    ratelimit.NewBucket(perMin(int(rl.CreatePerMin)), max1(int(rl.CreatePerMin)), 30*time.Minute),
			resolveBucket:   ratelimit.NewBucket(perMin(int(rl.ResolvePerMin)), max1(int(rl.ResolvePerMin)), 10*time.Minute),
			missBucket:      ratelimit.NewBucket(perMin(int(rl.MissPerMin)), max1(int(rl.MissPerMin)), 10*time.Minute),
			shorthandBucket: ratelimit.NewBucket(perHour(int(rl.ShorthandPerHour)), max1(int(rl.ShorthandPerHour)), time.Hour),
			createDaily:     ratelimit.NewDailyCounter(int(rl.CreatePerDay)),
			shorthandHourly: ratelimit.NewHourlyCounter(int(rl.ShorthandPerHour)),
			bucketCleanup:   seconds(rl.CleanupSeconds),

			engine: engine,
		}
		instance.registerRoutes()

		logger.Info().
			Str("addr", instance.listenAddr).
			Str("trusted_proxy_mode", instance.trustedProxyMode).
			Msg("api initialised")
	})
	return nil
}

// Get returns the global Server. Panics if Initialize has not run.
func Get() *Server {
	if instance == nil {
		panic("api: not initialised, call Initialize first")
	}
	return instance
}

// Engine exposes the underlying Gin engine for testing.
func Engine() *gin.Engine { return Get().engine }

// Run starts the HTTP server and blocks until ctx is done. On shutdown it
// gracefully drains in-flight requests up to the configured timeout.
func Run(ctx context.Context) error { return Get().Run(ctx) }

// Run starts and blocks. Used by Run() above and tests.
func (s *Server) Run(ctx context.Context) error {
	s.http = &http.Server{
		Addr:              s.listenAddr,
		Handler:           s.engine,
		ReadHeaderTimeout: s.readHeaderTimeout,
		WriteTimeout:      s.writeTimeout,
		IdleTimeout:       s.idleTimeout,
	}

	go s.startBucketCleanup(ctx)

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
	} {
		go b.RunCleanup(ctx, interval)
	}
}
