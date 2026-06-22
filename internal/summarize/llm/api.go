package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIProvider talks to OpenAI-compatible chat-completions endpoints.
type APIProvider struct {
	endpoint string
	token    string
	model    string
	client   *http.Client
}

// NewAPIProvider constructs an APIProvider. The model defaults to "gpt-4o-mini".
func NewAPIProvider(endpoint, token, model string) *APIProvider {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &APIProvider{
		endpoint: endpoint,
		token:    token,
		model:    model,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

// Name returns the static provider identifier.
func (p *APIProvider) Name() string { return "api" }

// Available is true when both endpoint and token are non-empty.
func (p *APIProvider) Available() bool {
	return p.endpoint != "" && p.token != ""
}

// chatRequest is the minimal OpenAI-compatible chat completions request shape.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// rateLimitedError carries the server-suggested Retry-After delay so the
// Generate retry loop can honor it. It wraps ErrRateLimited so callers can
// match with errors.Is.
type rateLimitedError struct {
	retryAfter time.Duration
	body       string
}

func (e *rateLimitedError) Error() string {
	if e.retryAfter > 0 {
		return fmt.Sprintf("%s: retry after %s (%s)", ErrRateLimited, e.retryAfter, e.body)
	}
	return fmt.Sprintf("%s: %s", ErrRateLimited, e.body)
}

func (e *rateLimitedError) Unwrap() error { return ErrRateLimited }

// maxRetryAfter caps server-suggested delays so a misbehaving endpoint cannot
// stall the importer. Documents that genuinely require longer cooldowns will
// fall back to extractive summarization on the next failure.
const maxRetryAfter = 30 * time.Second

// retrySchedule defines the per-attempt backoff for non-rate-limit transient
// errors (network failures, 5xx). A rate-limit response overrides the
// schedule's delay with the server's Retry-After hint (capped).
var retrySchedule = []time.Duration{0, 1 * time.Second, 2 * time.Second, 4 * time.Second}

// Generate sends the prompt to the endpoint with retry-with-backoff. Rate
// limits (429) are detected specifically and honor the Retry-After header.
func (p *APIProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if !p.Available() {
		return "", ErrUnavailable
	}

	body := chatRequest{
		Model: p.model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
	}

	var lastErr error
	for i, baseDelay := range retrySchedule {
		delay := baseDelay
		// If the previous attempt was rate-limited, prefer the server's hint.
		var rl *rateLimitedError
		if errors.As(lastErr, &rl) && rl.retryAfter > 0 {
			delay = rl.retryAfter
		}
		if delay > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}

		text, err := p.doRequest(ctx, body)
		if err == nil {
			return text, nil
		}
		lastErr = err

		if i < len(retrySchedule)-1 && isRetryable(err) {
			continue
		}
		return "", err
	}
	return "", lastErr
}

// doRequest issues a single HTTP request and parses the response.
func (p *APIProvider) doRequest(ctx context.Context, body chatRequest) (string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.token)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", &rateLimitedError{
			retryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
			body:       strings.TrimSpace(string(respBody)),
		}
	}
	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("%w: server returned %d", ErrUnavailable, resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("%w: %d %s", ErrAPI, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("%w: %s", ErrAPI, parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("%w: empty response", ErrAPI)
	}
	return parsed.Choices[0].Message.Content, nil
}

// parseRetryAfter parses an HTTP Retry-After header. The header can be either
// an integer number of seconds or an HTTP-date (RFC 7231 §7.1.3). Returns 0
// if the header is empty or unparseable. The returned delay is clamped to
// [0, maxRetryAfter].
func parseRetryAfter(header string) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}
	if secs, err := strconv.Atoi(header); err == nil {
		return clampDelay(time.Duration(secs) * time.Second)
	}
	if t, err := http.ParseTime(header); err == nil {
		return clampDelay(time.Until(t))
	}
	return 0
}

func clampDelay(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	if d > maxRetryAfter {
		return maxRetryAfter
	}
	return d
}

// isRetryable reports whether the given error warrants another attempt.
// Rate-limit and transient errors retry; 4xx (other than 429) does not.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrRateLimited) {
		return true
	}
	// ErrAPI is for 4xx responses — not retryable. Everything else is transient.
	return !strings.Contains(err.Error(), ErrAPI.Error())
}
