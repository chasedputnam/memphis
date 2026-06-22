//go:build !windows

package llm

func newWindowsPlatformProvider() (Provider, bool) {
	return nil, false
}
