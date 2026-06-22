//go:build darwin

package llm

import "context"

// appleProvider is a stub for Apple Intelligence integration. Bridging into
// the Foundation Models framework requires Swift/Objective-C and is out of
// scope for this initial implementation; we report unavailable so the engine
// falls back to Ollama (or the extractive summarizer).
type appleProvider struct{}

func (appleProvider) Name() string { return "apple" }

func (appleProvider) Available() bool { return false }

func (appleProvider) Generate(_ context.Context, _ string) (string, error) {
	return "", ErrUnavailable
}

// newApplePlatformProvider returns a Darwin-side provider stub. The second
// return value indicates whether the platform is supported (always true on
// Darwin, even though Available() is currently false).
func newApplePlatformProvider() (Provider, bool) {
	return appleProvider{}, true
}
