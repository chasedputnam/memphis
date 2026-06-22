//go:build darwin && (!applefm || !arm64 || !cgo)

package llm

import "context"

// appleProvider is the Darwin fallback when the real Apple Foundation Models
// bridge has not been opted into. The real provider lives in
// apple_darwin_fm.go and is built only when the `applefm` build tag is set on
// an Apple-Silicon machine with CGo enabled. Reporting Available() = false
// causes the engine to fall back to Ollama (or extractive summarization).
type appleProvider struct{}

func (appleProvider) Name() string { return "apple" }

func (appleProvider) Available() bool { return false }

func (appleProvider) Generate(_ context.Context, _ string) (string, error) {
	return "", ErrUnavailable
}

// newApplePlatformProvider returns the stub Darwin provider. The second
// return value signals that the platform is "supported" in the sense that the
// engine should consult the provider before falling back; Available() = false
// then routes to Ollama.
func newApplePlatformProvider() (Provider, bool) {
	return appleProvider{}, true
}
