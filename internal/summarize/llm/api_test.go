package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestAPIProvider_Available(t *testing.T) {
	p := NewAPIProvider("", "", "")
	if p.Available() {
		t.Error("expected unavailable when endpoint/token missing")
	}
	p2 := NewAPIProvider("http://x", "tok", "")
	if !p2.Available() {
		t.Error("expected available")
	}
}

func TestAPIProvider_GenerateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("missing/incorrect auth header")
		}
		body, _ := json.Marshal(chatResponse{Choices: []struct {
			Message chatMessage `json:"message"`
		}{{Message: chatMessage{Role: "assistant", Content: "Generated summary."}}}})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	p := NewAPIProvider(srv.URL, "tok", "")
	got, err := p.Generate(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if got != "Generated summary." {
		t.Errorf("got %q", got)
	}
}

func TestAPIProvider_ClientErrorNotRetried(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad"}}`))
	}))
	defer srv.Close()

	p := NewAPIProvider(srv.URL, "tok", "")
	_, err := p.Generate(context.Background(), "hello")
	if err == nil {
		t.Error("expected error")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (4xx no retry), got %d", calls)
	}
}

func TestAPIProvider_ServerErrorRetries(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body, _ := json.Marshal(chatResponse{Choices: []struct {
			Message chatMessage `json:"message"`
		}{{Message: chatMessage{Role: "assistant", Content: "ok"}}}})
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	p := NewAPIProvider(srv.URL, "tok", "")
	got, err := p.Generate(context.Background(), "hello")
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if got != "ok" {
		t.Errorf("got %q", got)
	}
	if calls < 2 {
		t.Errorf("expected >=2 calls, got %d", calls)
	}
}

// Truncation behavior is exercised in engine_test.go alongside the
// intelligent-truncation logic; the old byte-based helper has been replaced.

// TestAPIProvider_RateLimitHonorsRetryAfter pins the spec behavior: a 429
// response with a Retry-After header makes the provider sleep at least that
// long before its next attempt, then succeed. The actual sleep is measured
// to confirm we don't retry immediately (i.e., we do respect the rate
// limit). The header is set to a small value (1s) to keep the test fast.
func TestAPIProvider_RateLimitHonorsRetryAfter(t *testing.T) {
	calls := 0
	var firstCallEnd, secondCallStart time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			firstCallEnd = time.Now()
			return
		}
		secondCallStart = time.Now()
		body, _ := json.Marshal(chatResponse{Choices: []struct {
			Message chatMessage `json:"message"`
		}{{Message: chatMessage{Role: "assistant", Content: "after-rate-limit"}}}})
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	p := NewAPIProvider(srv.URL, "tok", "")
	got, err := p.Generate(context.Background(), "hello")
	if err != nil {
		t.Fatalf("expected success after honoring Retry-After, got %v", err)
	}
	if got != "after-rate-limit" {
		t.Errorf("unexpected body: %q", got)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (one 429, one success), got %d", calls)
	}
	gap := secondCallStart.Sub(firstCallEnd)
	if gap < 900*time.Millisecond {
		t.Errorf("expected at least ~1s wait after Retry-After:1, got %v", gap)
	}
}

// TestAPIProvider_RateLimitWithoutRetryAfter falls back to the backoff
// schedule when the server doesn't supply Retry-After.
func TestAPIProvider_RateLimitWithoutRetryAfter(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"slow down"}}`))
			return
		}
		body, _ := json.Marshal(chatResponse{Choices: []struct {
			Message chatMessage `json:"message"`
		}{{Message: chatMessage{Role: "assistant", Content: "ok"}}}})
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	p := NewAPIProvider(srv.URL, "tok", "")
	got, err := p.Generate(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got != "ok" {
		t.Errorf("got %q", got)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

// TestAPIProvider_PersistentRateLimitSurfacesError ensures the provider
// gives up after exhausting retries and returns an error wrapping
// ErrRateLimited so callers can fall back to extractive summarization.
func TestAPIProvider_PersistentRateLimitSurfacesError(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "0") // 0s — fast test
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"still limited"}}`))
	}))
	defer srv.Close()

	p := NewAPIProvider(srv.URL, "tok", "")
	_, err := p.Generate(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error after persistent 429s")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected errors.Is(err, ErrRateLimited), got: %v", err)
	}
	if calls != len(retrySchedule) {
		t.Errorf("expected %d attempts, got %d", len(retrySchedule), calls)
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"", 0},
		{"3", 3 * time.Second},
		{"0", 0},
		{"-1", 0},
		{"not-a-number", 0},
		{strconv.Itoa(int(maxRetryAfter/time.Second) + 10), maxRetryAfter}, // clamp
	}
	for _, tc := range tests {
		got := parseRetryAfter(tc.in)
		if got != tc.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
	// HTTP-date form: ~2 seconds in the future, must round up to ~2s (and
	// be non-zero). We allow some slack because http.ParseTime is second-precision.
	future := time.Now().UTC().Add(2 * time.Second).Format(http.TimeFormat)
	got := parseRetryAfter(future)
	if got <= 0 || got > 3*time.Second {
		t.Errorf("parseRetryAfter(http-date 2s ahead) = %v, want >0 and <=3s", got)
	}
}
