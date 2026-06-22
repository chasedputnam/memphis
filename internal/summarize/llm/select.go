package llm

import (
	"runtime"
)

// selectProvider returns the best available provider for the given config:
//  1. External API if api_endpoint and api_token are configured.
//  2. Platform-native LLM if available (Apple Intelligence, Copilot Runtime).
//  3. Ollama at local_endpoint (default localhost:11434).
//
// If nothing is available, a degraded provider is returned that always returns
// ErrNoProvider — callers fall back to extractive summarization.
func selectProvider(cfg *Config) (Provider, error) {
	if cfg.HasAPI() {
		return NewAPIProvider(cfg.APIEndpoint, cfg.APIToken, cfg.Model), nil
	}

	// Try platform-native first.
	switch runtime.GOOS {
	case "darwin":
		if p, ok := newApplePlatformProvider(); ok && p.Available() {
			return p, nil
		}
	case "windows":
		if p, ok := newWindowsPlatformProvider(); ok && p.Available() {
			return p, nil
		}
	}

	// Fall back to Ollama.
	endpoint := cfg.LocalEndpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	model := cfg.LocalModel
	if model == "" {
		model = "phi3:mini"
	}
	return NewOllamaProvider(endpoint, model), nil
}
