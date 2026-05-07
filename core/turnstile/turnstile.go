// core/turnstile/turnstile.go

package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"smegg.me/qr2g.it/common/config"
	"smegg.me/qr2g.it/common/logger"
)

// Verifier performs Siteverify lookups.
type Verifier struct {
	secretKey        string
	siteverifyURL    string
	expectedAction   string
	expectedHostname string
	enabled          bool
	requestTimeout   time.Duration
	hc               *http.Client
}

var (
	instance *Verifier
	initOnce sync.Once
)

// Initialize wires the Verifier from config. Safe to call multiple times.
func Initialize() error {
	var initErr error
	initOnce.Do(func() {
		cfg := config.Get().Turnstile
		timeout := time.Duration(cfg.RequestTimeoutSeconds) * time.Second
		if cfg.Enabled && strings.TrimSpace(cfg.SecretKey) == "" {
			initErr = ErrEmptySecret
			return
		}
		instance = &Verifier{
			secretKey:        cfg.SecretKey,
			siteverifyURL:    cfg.SiteverifyURL,
			expectedAction:   strings.TrimSpace(cfg.ExpectedAction),
			expectedHostname: strings.TrimSpace(cfg.ExpectedHostname),
			enabled:          cfg.Enabled,
			requestTimeout:   timeout,
			hc:               &http.Client{Timeout: timeout},
		}
		if !instance.IsEnabled() {
			logger.Warn().Msg("turnstile disabled")
		} else {
			logger.Info().Msg("turnstile verifier ready")
		}
	})
	return initErr
}

// Get returns the global Verifier. Panics if Initialize has not run.
func Get() *Verifier {
	if instance == nil {
		panic("turnstile: not initialised, call Initialize first")
	}
	return instance
}

// Errors a caller may want to map to specific responses.
var (
	ErrEmptyToken      = errors.New("turnstile: empty token")
	ErrEmptySecret     = errors.New("turnstile: secret not configured")
	ErrVerifyFailed    = errors.New("turnstile: verification failed")
	ErrTransientVerify = errors.New("turnstile: transient verification error")
	ErrDisabled        = errors.New("turnstile: disabled")
)

// IsEnabled reports whether the verifier is wired up to actually call
// Siteverify. When false, callers should typically allow requests through.
func (v *Verifier) IsEnabled() bool {
	return v != nil && v.enabled && v.secretKey != ""
}

type siteverifyResponse struct {
	Success     bool      `json:"success"`
	ChallengeTs time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	Action      string    `json:"action"`
	CData       string    `json:"cdata"`
	ErrorCodes  []string  `json:"error-codes"`
	Metadata    struct {
		ResultWithTestingKey bool `json:"result_with_testing_key"`
	} `json:"metadata"`
}

// Verify validates token (and optionally remote IP). On success returns nil.
// On any non-success returns ErrVerifyFailed (with detail) or ErrTransientVerify.
func (v *Verifier) Verify(ctx context.Context, token, remoteIP string) error {
	if v == nil || !v.enabled {
		return ErrDisabled
	}
	if v.secretKey == "" {
		return ErrEmptySecret
	}
	if strings.TrimSpace(token) == "" {
		return ErrEmptyToken
	}

	form := url.Values{}
	form.Set("secret", v.secretKey)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		v.siteverifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("%w: build req: %v", ErrTransientVerify, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := v.hc.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTransientVerify, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<10))

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%w: http %d", ErrTransientVerify, resp.StatusCode)
	}

	var out siteverifyResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("%w: decode: %v", ErrTransientVerify, err)
	}
	if !out.Success {
		return fmt.Errorf("%w: %s", ErrVerifyFailed, strings.Join(out.ErrorCodes, ","))
	}
	if v.isTestingValidation(out) {
		return nil
	}
	if v.expectedAction != "" && out.Action != v.expectedAction {
		return fmt.Errorf("%w: action mismatch", ErrVerifyFailed)
	}
	if v.expectedHostname != "" && !strings.EqualFold(out.Hostname, v.expectedHostname) {
		return fmt.Errorf("%w: hostname mismatch", ErrVerifyFailed)
	}
	return nil
}

func (v *Verifier) isTestingValidation(out siteverifyResponse) bool {
	return out.Metadata.ResultWithTestingKey || isCloudflareTestingSecret(v.secretKey)
}

func isCloudflareTestingSecret(secret string) bool {
	switch strings.TrimSpace(secret) {
	case "1x0000000000000000000000000000000AA",
		"2x0000000000000000000000000000000AA",
		"3x0000000000000000000000000000000AA":
		return true
	default:
		return false
	}
}
