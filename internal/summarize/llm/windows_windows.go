//go:build windows

package llm

import "context"

// windowsProvider is a stub for Windows Copilot Runtime integration.
type windowsProvider struct{}

func (windowsProvider) Name() string { return "windows" }

func (windowsProvider) Available() bool { return false }

func (windowsProvider) Generate(_ context.Context, _ string) (string, error) {
	return "", ErrUnavailable
}

func newWindowsPlatformProvider() (Provider, bool) {
	return windowsProvider{}, true
}
