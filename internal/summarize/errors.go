package summarize

import "errors"

// Standard error types returned by the summarization subsystem.
var (
	// ErrNoSummarizerAvailable is returned when no usable summarizer can be
	// constructed (e.g., the configured LLM provider is unavailable and the
	// fallback also failed).
	ErrNoSummarizerAvailable = errors.New("no summarizer available")
	// ErrLLMUnavailable indicates that a configured LLM provider cannot be
	// reached.
	ErrLLMUnavailable = errors.New("LLM provider unavailable")
	// ErrAPIError indicates a remote API returned an error.
	ErrAPIError = errors.New("API request failed")
	// ErrTimeout indicates that summarization exceeded its time budget.
	ErrTimeout = errors.New("summarization timed out")
	// ErrContentTooLong indicates that the input content exceeded the
	// provider's maximum size and could not be truncated to fit.
	ErrContentTooLong = errors.New("content exceeds maximum length")
)
