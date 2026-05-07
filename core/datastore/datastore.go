// core/datastore/datastore.go

package datastore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/common/logger"
)

var (
	pool      *pgxpool.Pool
	connStr   string
	initOnce  sync.Once
	closeOnce sync.Once
)

func applyPoolSettings(p *pgxpool.Config, cfg config.DBConfig) {
	if cfg.MaxConns > 0 {
		p.MaxConns = int32(cfg.MaxConns)
	}
	if cfg.MinConns >= 0 {
		p.MinConns = int32(cfg.MinConns)
	}
	if cfg.HealthCheckPeriodSeconds > 0 {
		p.HealthCheckPeriod = time.Duration(cfg.HealthCheckPeriodSeconds) * time.Second
	}
	if cfg.ConnectTimeoutSeconds > 0 {
		p.ConnConfig.ConnectTimeout = time.Duration(cfg.ConnectTimeoutSeconds) * time.Second
	}
}

// Initialize opens the Postgres connection pool, pings it, and runs any
// pending migrations when auto_migrate is enabled in config.
func Initialize(ctx context.Context) error {
	var initErr error
	initOnce.Do(func() {
		cfg := config.Get().Db
		if cfg.URL == "" {
			initErr = errors.New("datastore: DATABASE_URL not set")
			return
		}

		logger.Info().Msg("initializing datastore")
		logger.Debug().Msg("parsing pool configuration")

		poolCfg, err := pgxpool.ParseConfig(cfg.URL)
		if err != nil {
			initErr = fmt.Errorf("parse db url: %w", err)
			return
		}
		applyPoolSettings(poolCfg, cfg)

		connStr = cfg.URL

		connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		logger.Debug().Int("max_conns", int(poolCfg.MaxConns)).Msg("opening pool")

		p, err := pgxpool.NewWithConfig(connectCtx, poolCfg)
		if err != nil {
			initErr = fmt.Errorf("connect db: %w", err)
			return
		}

		if err := p.Ping(connectCtx); err != nil {
			p.Close()
			initErr = fmt.Errorf("ping db: %w", err)
			return
		}

		pool = p
		logger.Info().Msg("datastore connected")

		if cfg.AutoMigrate {
			logger.Info().Msg("running migrations")
			if err := runMigrations(ctx); err != nil {
				p.Close()
				pool = nil
				initErr = fmt.Errorf("migrate: %w", err)
				return
			}
		}

		logger.Info().Msg("datastore ready")
	})
	return initErr
}

// Get returns the shared pgx pool. Panics if Initialize has not run.
func Get() *pgxpool.Pool {
	if pool == nil {
		panic("datastore: not initialised, call Initialize first")
	}
	return pool
}

// Ping verifies database connectivity.
func Ping(ctx context.Context) error {
	if pool == nil {
		return errors.New("datastore: not initialised")
	}
	return pool.Ping(ctx)
}

// Close releases the pool. Safe to call multiple times.
func Close() {
	closeOnce.Do(func() {
		if pool != nil {
			pool.Close()
			pool = nil
		}
	})
}
