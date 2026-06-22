//go:build darwin && arm64 && cgo && applefm

package llm

import (
	"context"
	"fmt"
	"strings"

	fm "github.com/blacktop/go-foundationmodels"
)

// appleProvider talks directly to Apple's on-device Foundation Models
// framework via the blacktop/go-foundationmodels CGo bridge. Selected on
// Apple-Silicon Macs with macOS 26+ and Apple Intelligence enabled, when the
// `applefm` build tag is set.
//
// Build requirements (see docs/APPLE_INTELLIGENCE.md):
//
//	CGO_ENABLED=1, Xcode + macOS 26 SDK, libFMShim.a present in the dep
//	directory (built via `go generate` inside the vendored copy of
//	github.com/blacktop/go-foundationmodels).
type appleProvider struct{}

// Name returns the static provider identifier.
func (appleProvider) Name() string { return "apple" }

// Available reports whether Foundation Models is reachable on this device.
// It is a lightweight runtime probe: just a CGo call into the Swift shim,
// no model load.
func (appleProvider) Available() bool {
	return fm.CheckModelAvailability() == fm.ModelAvailable
}

// Generate runs the prompt against a fresh on-device session. We create one
// session per call rather than caching: sessions are cheap, the underlying
// library is not thread-safe, and a per-call lifecycle avoids context-window
// leakage across documents.
//
// The underlying Respond() is synchronous and cannot be cancelled mid-
// computation. To honor ctx, we run it in a goroutine and select on
// ctx.Done(); on cancellation the goroutine is left to finish on its own and
// its result is discarded (the buffered channel keeps it from blocking).
func (appleProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if fm.CheckModelAvailability() != fm.ModelAvailable {
		return "", ErrUnavailable
	}

	sess := fm.NewSession()
	if sess == nil {
		return "", ErrUnavailable
	}

	type result struct {
		text string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		defer sess.Release()
		text := sess.Respond(prompt, nil)
		// The upstream shim encodes failures (guardrail violation, context
		// overflow, etc.) as "Error: ..." prefixed strings rather than Go
		// errors. Translate so callers can errors.Is(... , ErrAPI).
		if strings.HasPrefix(text, "Error:") {
			ch <- result{err: fmt.Errorf("%w: %s", ErrAPI, strings.TrimSpace(strings.TrimPrefix(text, "Error:")))}
			return
		}
		ch <- result{text: text}
	}()

	select {
	case <-ctx.Done():
		// Inner CGo call keeps running; result will be GC'd once the
		// goroutine finishes and drops its references.
		return "", ctx.Err()
	case r := <-ch:
		return r.text, r.err
	}
}

// newApplePlatformProvider returns the real Foundation Models provider.
// The selector calls Available() before committing to it; if Apple
// Intelligence is disabled or the device is ineligible, the engine falls
// back to Ollama.
func newApplePlatformProvider() (Provider, bool) {
	return appleProvider{}, true
}
