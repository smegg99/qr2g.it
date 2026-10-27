// cmd/serve.go

package cmd

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	api "smegg.me/qr2g.it/api/v1"
	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/common/messages"
	"smegg.me/qr2g.it/core/automation"
	"smegg.me/qr2g.it/core/datastore"
	"smegg.me/qr2g.it/core/github"
	"smegg.me/qr2g.it/core/policy"
	"smegg.me/qr2g.it/core/repositories"
	"smegg.me/qr2g.it/core/stats"
	"smegg.me/qr2g.it/core/turnstile"
	"smegg.me/qr2g.it/core/verifier"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: messages.CliServe(say),
		Args:  usage(cobra.NoArgs),
		RunE:  func(cmd *cobra.Command, _ []string) error { return serve(cmd.Context()) },
	}
}

func seconds(n int64) time.Duration { return time.Duration(n) * time.Second }

func serve(parent context.Context) error {
	logger.Initialize()
	logger.Info().Str("version", Version).Msg("starting qr2g.it")
	if err := loadConfig(); err != nil {
		return err
	}
	cfg := config.Get()

	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := datastore.Initialize(ctx); err != nil {
		return err
	}
	if err := github.Initialize(); err != nil {
		return err
	}
	if err := turnstile.Initialize(); err != nil {
		return err
	}
	if err := verifier.Initialize(); err != nil {
		return err
	}

	db := datastore.Get()
	links := repositories.NewLinkRepository(db)
	statsRepo := repositories.NewStatsRepository(db)
	recorder := stats.NewRecorder(statsRepo, seconds(cfg.Statistics.FlushIntervalSeconds))
	checker := policy.NewChecker(policy.Config{
		Enabled:         cfg.Policy.Enabled,
		Interval:        seconds(cfg.Policy.IntervalSeconds),
		BatchSize:       int(cfg.Policy.BatchSize),
		FreshTTL:        seconds(cfg.Policy.FreshTTLSeconds),
		FailureBackoff:  seconds(cfg.Policy.FailureBackoffSeconds),
		RefreshCooldown: seconds(cfg.Policy.RefreshCooldownSeconds),
	}, github.Get(), verifier.Get(), links)

	server, err := api.New(ctx, api.Deps{
		Links:      links,
		Stats:      statsRepo,
		Automation: automation.NewService(repositories.NewAutomationRepository(db)),
		Policy:     checker,
		Recorder:   recorder,
		Verifier:   verifier.Get(),
		GitHub:     github.Get(),
		Turnstile:  turnstile.Get(),
		Ready:      datastore.Ping,
		Version:    Version,
	})
	if err != nil {
		return err
	}
	return run(ctx, stop, server, recorder, checker)
}

// run blocks until a signal or a failed worker, then stops intake before the
// statistics flush and closes the database last.
func run(ctx context.Context, stop context.CancelFunc, server *api.Server, recorder *stats.Recorder, checker *policy.Checker) error {
	workers := map[string]func(context.Context) error{
		"verifier": verifier.Get().Run,
		"policy":   checker.Run,
	}
	var wg sync.WaitGroup
	errs := make(chan error, len(workers)+1)
	for name, work := range workers {
		wg.Go(func() {
			if err := work(ctx); err != nil && !errors.Is(err, context.Canceled) {
				logger.Error().Err(err).Str("worker", name).Msg("worker stopped")
				errs <- err
				stop()
			}
		})
	}
	flushCtx, stopFlush := context.WithCancel(context.WithoutCancel(ctx))
	flushed := make(chan struct{})
	go func() {
		defer close(flushed)
		_ = recorder.Run(flushCtx)
	}()

	serverErr := server.Run(ctx)
	if serverErr != nil {
		stop()
	}
	wg.Wait()
	stopFlush()
	<-flushed
	close(errs)
	if serverErr != nil {
		return serverErr
	}
	return <-errs
}
