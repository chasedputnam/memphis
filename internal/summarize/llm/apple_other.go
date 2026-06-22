//go:build !darwin

package llm

// newApplePlatformProvider is a no-op on non-Darwin platforms.
func newApplePlatformProvider() (Provider, bool) {
	return nil, false
}
