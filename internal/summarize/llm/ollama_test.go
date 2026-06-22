package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOllamaProvider_GenerateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := json.Marshal(ollamaResponse{Response: "Hello from Ollama"})
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	p := NewOllamaProvider(srv.URL, "phi3:mini")
	got, err := p.Generate(context.Background(), "say hi")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got != "Hello from Ollama" {
		t.Errorf("got %q", got)
	}
}

func TestOllamaProvider_Available(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("unexpected probe path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewOllamaProvider(srv.URL, "phi3:mini")
	if !p.Available() {
		t.Error("expected available")
	}
}

func TestOllamaProvider_AvailableWhenDown(t *testing.T) {
	// Bind a server then close immediately so the port is dead.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close()
	p := NewOllamaProvider(addr, "phi3:mini")
	if p.Available() {
		t.Error("expected unavailable")
	}
}

// TestOllamaProvider_AvailableTimesOutWhenServerHangs verifies the spec
// requirement: the Available() probe is bounded by a short context timeout
// (2s) so that an unresponsive Ollama instance does not stall import. A
// dead-port test only exercises the connect-refused path; this test exercises
// the connect-but-no-response path that requires the timeout to fire.
func TestOllamaProvider_AvailableTimesOutWhenServerHangs(t *testing.T) {
	handlerEntered := make(chan struct{}, 1)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case handlerEntered <- struct{}{}:
		default:
		}
		<-release // block until the test releases us
	}))
	// release first, then close — otherwise srv.Close blocks on the in-flight handler.
	defer func() {
		close(release)
		srv.Close()
	}()

	p := NewOllamaProvider(srv.URL, "phi3:mini")
	start := time.Now()
	avail := p.Available()
	elapsed := time.Since(start)

	if avail {
		t.Error("expected Available()=false when server hangs past the probe timeout")
	}
	// The probe uses a 2s context timeout. Allow generous CI slack but assert
	// we returned well before the http.Client's 60s body timeout.
	if elapsed > 5*time.Second {
		t.Errorf("Available() took %v; probe timeout (~2s) was not honored", elapsed)
	}
	// Confirm we actually reached the server (rather than failing on connect).
	select {
	case <-handlerEntered:
		// good
	default:
		t.Error("server handler was never entered — test setup wrong")
	}
}

func TestOllamaProvider_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := json.Marshal(ollamaResponse{Error: "model not found"})
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	p := NewOllamaProvider(srv.URL, "phi3:mini")
	_, err := p.Generate(context.Background(), "hi")
	if err == nil {
		t.Error("expected error")
	}
}
