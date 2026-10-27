// core/github/rest_test.go

package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// A spent search allowance must not pause background work, which spends the core one.
func TestPublicRepositoryCountKeepsTheCoreAllowance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/repositories" || r.URL.Query().Get("q") != "is:public fork:true" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-RateLimit-Resource", "search")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))
		_, _ = w.Write([]byte(`{"total_count": 276430654, "incomplete_results": true, "items": []}`))
	}))
	defer srv.Close()
	c := NewClient(ClientConfig{RESTBaseURL: srv.URL, RequestTimeout: time.Second, Reserve: 10})
	n, err := c.PublicRepositoryCount(context.Background())
	if err != nil || n != 276430654 {
		t.Fatalf("count = %d, %v", n, err)
	}
	if !c.BackgroundAllowed(time.Now()) {
		t.Fatal("the search allowance was recorded as the core allowance")
	}
}
