package llm

import "errors"

// ErrNoProvider indicates no LLM provider is available.
var ErrNoProvider = errors.New("no LLM provider available")

// ErrUnavailable indicates a provider exists but is currently unreachable.
var ErrUnavailable = errors.New("LLM provider unavailable")

// ErrAPI indicates the remote API returned an error response.
var ErrAPI = errors.New("LLM API error")

// ErrRateLimited indicates the remote API returned 429 Too Many Requests.
// Callers (and the provider's internal retry loop) honor the embedded
// Retry-After hint when present.
var ErrRateLimited = errors.New("LLM API rate limited")
