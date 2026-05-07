// core/github/app.go

package github

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// appConfig contains the values needed to authenticate as a GitHub App
// installation. Internal to the github package.
type appConfig struct {
	AppID            int64
	InstallationID   int64
	PrivateKeyPath   string
	UserAgent        string
	APIVersion       string
	RESTBaseURL      string
	RequestTimeout   time.Duration
	TokenRefreshSkew time.Duration
}

// app caches an installation token for use as a Bearer credential.
// Methods are safe for concurrent use.
type app struct {
	cfg     appConfig
	hc      *http.Client
	privKey *rsa.PrivateKey
	keyMu   sync.RWMutex

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func newApp(cfg appConfig, hc *http.Client) *app {
	if hc == nil {
		hc = &http.Client{Timeout: cfg.RequestTimeout}
	}
	return &app{cfg: cfg, hc: hc}
}

// IsConfigured reports whether App credentials look usable. It does not
// validate the private key on disk.
func (a *app) IsConfigured() bool {
	return a.cfg.AppID > 0 && a.cfg.InstallationID > 0 && a.cfg.PrivateKeyPath != ""
}

// Token returns a non-expired installation token, refreshing if needed.
func (a *app) Token(ctx context.Context) (string, error) {
	if !a.IsConfigured() {
		return "", errors.New("github app: not configured")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.token != "" && time.Until(a.expiresAt) > a.cfg.TokenRefreshSkew {
		return a.token, nil
	}

	tok, exp, err := a.mintInstallationToken(ctx)
	if err != nil {
		return "", err
	}
	a.token = tok
	a.expiresAt = exp
	return tok, nil
}

// Reset wipes any cached token. Useful on auth failure.
func (a *app) Reset() {
	a.mu.Lock()
	a.token = ""
	a.expiresAt = time.Time{}
	a.mu.Unlock()
}

func (a *app) loadKey() (*rsa.PrivateKey, error) {
	a.keyMu.RLock()
	k := a.privKey
	a.keyMu.RUnlock()
	if k != nil {
		return k, nil
	}

	a.keyMu.Lock()
	defer a.keyMu.Unlock()
	if a.privKey != nil {
		return a.privKey, nil
	}
	pem, err := os.ReadFile(a.cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("github app: read private key: %w", err)
	}
	parsed, err := jwt.ParseRSAPrivateKeyFromPEM(pem)
	if err != nil {
		return nil, fmt.Errorf("github app: parse private key: %w", err)
	}
	a.privKey = parsed
	return parsed, nil
}

func (a *app) signAppJWT() (string, error) {
	key, err := a.loadKey()
	if err != nil {
		return "", err
	}
	now := time.Now()
	// GitHub allows up to 10-minute JWTs and rejects future-dated iat by
	// more than ~60 s of skew. Backdate slightly to absorb clock drift.
	claims := jwt.RegisteredClaims{
		Issuer:    strconv.FormatInt(a.cfg.AppID, 10),
		IssuedAt:  jwt.NewNumericDate(now.Add(-30 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(8 * time.Minute)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := tok.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("github app: sign jwt: %w", err)
	}
	return signed, nil
}

func (a *app) mintInstallationToken(ctx context.Context) (string, time.Time, error) {
	appJWT, err := a.signAppJWT()
	if err != nil {
		return "", time.Time{}, err
	}

	url := fmt.Sprintf("%s/app/installations/%d/access_tokens",
		a.cfg.RESTBaseURL, a.cfg.InstallationID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("github app: build req: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", a.cfg.APIVersion)
	req.Header.Set("User-Agent", a.cfg.UserAgent)

	resp, err := a.hc.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("github app: token request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode/100 != 2 {
		return "", time.Time{}, fmt.Errorf("github app: token http %d: %s",
			resp.StatusCode, redact(body))
	}

	var out installationTokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", time.Time{}, fmt.Errorf("github app: parse token response: %w", err)
	}
	if out.Token == "" {
		return "", time.Time{}, errors.New("github app: empty token in response")
	}
	return out.Token, out.ExpiresAt, nil
}

// redact returns a short, token-free preview safe to include in error logs.
func redact(b []byte) string {
	const max = 256
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "...(truncated)"
}
