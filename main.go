// main.go

// @title           qr2g.it API
// @version         1.0
// @description     Permanent, QR-friendly short links for public GitHub repositories.
// @description     Slugs are uppercase base36 of the link's id; the canonical
// @description     short URL uses the uppercase host (QR-optimised) while the
// @description     redirect target is the lowercase public host.
// @license.name    MIT
// @license.url     https://github.com/smegg99/qr2g.it/blob/main/LICENSE
// @BasePath        /
// @schemes         http https
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"smegg.me/qr2g.it/api/v1"
	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/common/logger"
	"smegg.me/qr2g.it/core/datastore"
	"smegg.me/qr2g.it/core/github"
	"smegg.me/qr2g.it/core/turnstile"
	"smegg.me/qr2g.it/core/verifier"
)

func Cleanup() error {
	logger.Info().Msg("cleaning up...")
	datastore.Close()
	return nil
}

func main() {
	defer func() {
		if err := Cleanup(); err != nil {
			logger.Fatal().Err(err).Msg("cleanup")
		}
	}()

	logger.Initialize()
	logger.Info().Msg("starting qr2g.it...")

	logger.Debug().Msg("loading configuration")
	if path, err := config.Initialize(); err != nil {
		if e, ok := config.IsDefaultGenerated(err); ok {
			logger.Info().Str("config_path", e.Path).Msg("default config generated")
			os.Exit(0)
		}
		logger.Fatal().Msg(err.Error())
	} else {
		logger.Debug().Str("config_path", path).Msg("configuration loaded")
	}

	cfg := config.Get()

	logger.Debug().Msg("reinitializing logger with config")
	if err := logger.ReInitialize(cfg.Logger); err != nil {
		logger.Fatal().Err(err).Msg("logger")
	}

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Debug().Msg("initializing datastore")
	if err := datastore.Initialize(rootCtx); err != nil {
		logger.Fatal().Err(err).Msg("datastore")
	}

	logger.Debug().Msg("initializing github client")
	if err := github.Initialize(); err != nil {
		logger.Fatal().Err(err).Msg("github")
	}

	logger.Debug().Msg("initializing turnstile")
	if err := turnstile.Initialize(); err != nil {
		logger.Fatal().Err(err).Msg("turnstile")
	}

	logger.Debug().Msg("initializing verifier")
	if err := verifier.Initialize(); err != nil {
		logger.Fatal().Err(err).Msg("verifier")
	}

	logger.Debug().Msg("initializing api server")
	if err := api.Initialize(); err != nil {
		logger.Fatal().Err(err).Msg("api")
	}

	if err := RunServer(rootCtx); err != nil {
		logger.Fatal().Err(err).Msg("server")
	}
}

// RunServer starts the verifier and HTTP API and blocks until SIGINT or
// SIGTERM, then performs graceful shutdown.
func RunServer(rootCtx context.Context) error {
	signalCtx, stopSignals := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	var wg sync.WaitGroup
	wg.Add(2)

	verifierErrCh := make(chan error, 1)
	go func() {
		defer wg.Done()
		verifierErrCh <- verifier.Get().Run(signalCtx)
	}()

	serverErrCh := make(chan error, 1)
	go func() {
		defer wg.Done()
		serverErrCh <- api.Run(signalCtx)
	}()

	select {
	case <-signalCtx.Done():
		logger.Info().Msg("shutdown signal received")
	case err := <-serverErrCh:
		if err != nil {
			logger.Error().Err(err).Msg("server stopped")
		}
		stopSignals()
	case err := <-verifierErrCh:
		if err != nil {
			logger.Error().Err(err).Msg("verifier stopped")
		}
	}

	wg.Wait()

	if err := <-serverErrCh; err != nil {
		return err
	}
	if err := <-verifierErrCh; err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
