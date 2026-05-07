package turnstile

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testingPassSecret = "1x0000000000000000000000000000000AA"

func verifierWithResponse(t *testing.T, secret, response string) *Verifier {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if got := r.Form.Get("secret"); got != secret {
			t.Errorf("secret = %q, want %q", got, secret)
		}
		if strings.TrimSpace(r.Form.Get("response")) == "" {
			t.Errorf("response token was empty")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)

	return &Verifier{
		secretKey:        secret,
		siteverifyURL:    srv.URL,
		expectedAction:   "create_link",
		expectedHostname: "qr2g.it",
		enabled:          true,
		requestTimeout:   time.Second,
		hc:               srv.Client(),
	}
}

func TestVerifyAllowsCloudflareTestingSecretDummyResponse(t *testing.T) {
	v := verifierWithResponse(t, testingPassSecret, `{
		"success": true,
		"challenge_ts": "2026-05-07T14:47:08.400Z",
		"hostname": "example.com",
		"error-codes": []
	}`)

	if err := v.Verify(context.Background(), "XXXX.DUMMY.TOKEN.XXXX", "127.0.0.1"); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifyAllowsTestingMetadataDummyResponse(t *testing.T) {
	v := verifierWithResponse(t, "real-secret", `{
		"success": true,
		"challenge_ts": "2026-05-07T14:47:08.400Z",
		"hostname": "example.com",
		"error-codes": [],
		"metadata": { "result_with_testing_key": true }
	}`)

	if err := v.Verify(context.Background(), "XXXX.DUMMY.TOKEN.XXXX", "127.0.0.1"); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifyRejectsStrictMismatchForRealSecret(t *testing.T) {
	v := verifierWithResponse(t, "real-secret", `{
		"success": true,
		"challenge_ts": "2026-05-07T14:47:08.400Z",
		"hostname": "example.com",
		"action": "other_action",
		"error-codes": []
	}`)

	err := v.Verify(context.Background(), "real-token", "127.0.0.1")
	if !errors.Is(err, ErrVerifyFailed) {
		t.Fatalf("Verify() error = %v, want %v", err, ErrVerifyFailed)
	}
}
