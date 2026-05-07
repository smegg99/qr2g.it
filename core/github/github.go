// core/github/github.go

package github

import (
	"net/http"
	"sync"
	"time"

	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/common/logger"
)

var (
	client   *Client
	initOnce sync.Once
)

// Initialize wires the GitHub App + REST + GraphQL client from config.
// Safe to call multiple times - only the first call has effect.
func Initialize() error {
	initOnce.Do(func() {
		cfg := config.Get().Github

		reqTimeout := time.Duration(cfg.RequestTimeoutSeconds) * time.Second
		appInstance := newApp(appConfig{
			AppID:            cfg.AppID,
			InstallationID:   cfg.InstallationID,
			PrivateKeyPath:   cfg.PrivateKeyPath,
			UserAgent:        cfg.UserAgent,
			APIVersion:       cfg.APIVersion,
			RESTBaseURL:      cfg.RESTBaseURL,
			RequestTimeout:   reqTimeout,
			TokenRefreshSkew: time.Duration(cfg.TokenRefreshSkewSeconds) * time.Second,
		}, &http.Client{Timeout: reqTimeout})

		client = newClient(ClientConfig{
			UserAgent:      cfg.UserAgent,
			APIVersion:     cfg.APIVersion,
			RESTBaseURL:    cfg.RESTBaseURL,
			GraphQLURL:     cfg.GraphQLURL,
			RequestTimeout: reqTimeout,
			UseAppAuth:     cfg.UseAppAuth,
			UseGraphQL:     cfg.UseGraphQL,
		}, appInstance)

		logger.Info().
			Bool("app_auth", client.UsesAppAuth()).
			Bool("graphql", client.UsesGraphQL()).
			Int64("app_id", cfg.AppID).
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
