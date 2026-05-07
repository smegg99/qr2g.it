// core/github/github.go

package github

import (
	"sync"
	"time"

	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/common/logger"
)

var (
	client   *Client
	initOnce sync.Once
)

// Initialize wires the GitHub REST client from config.
// Safe to call multiple times - only the first call has effect.
func Initialize() error {
	initOnce.Do(func() {
		cfg := config.Get().Github

		reqTimeout := time.Duration(cfg.RequestTimeoutSeconds) * time.Second
		client = newClient(ClientConfig{
			UserAgent:      cfg.UserAgent,
			APIVersion:     cfg.APIVersion,
			RESTBaseURL:    cfg.RESTBaseURL,
			RequestTimeout: reqTimeout,
		})

		logger.Info().
			Msg("github client ready")
	})
	return nil
}

// Get returns the global Client. Panics if Initialize has not run.
func Get() *Client {
	if client == nil {
		panic("github: not initialised, call Initialize first")
	}
	return client
}
