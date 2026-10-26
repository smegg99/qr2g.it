// core/datastore/datastore.go
package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/common/logger"
)

var (
	pool *sql.DB
	mu   sync.Mutex
)

// Open opens a file-backed SQLite database and optionally applies migrations.
func Open(ctx context.Context, path string, autoMigrate bool) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("datastore: database path is empty")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("database path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	u := url.URL{Scheme: "file", Path: path}
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(10000)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Set("_txlock", "immediate")
	q.Set("_time_format", "sqlite")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// Serialize this process's writes; immediate transactions also protect other processes.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if autoMigrate {
		if err := runMigrations(ctx, db); err != nil {
			db.Close()
			return nil, fmt.Errorf("migrate database: %w", err)
		}
	}
	return db, nil
}

// Initialize opens the configured SQLite database.
func Initialize(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()
	if pool != nil {
		return nil
	}
	cfg := config.Get().Db
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	db, err := Open(connectCtx, cfg.Path, cfg.AutoMigrate)
	if err != nil {
		return err
	}
	pool = db
	logger.Info().Msg("SQLite datastore ready")
	return nil
}

// Get returns the initialized shared database.
func Get() *sql.DB {
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
	return pool.PingContext(ctx)
}

// Close releases the shared database.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if pool != nil {
		pool.Close()
		pool = nil
	}
}
