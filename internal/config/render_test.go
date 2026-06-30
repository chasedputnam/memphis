package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeAndLoad renders cfg, writes it to a temp store, and loads it back.
func writeAndLoad(t *testing.T, cfg Config) Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".okf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(dir), []byte(Render(cfg)), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after Render failed: %v", err)
	}
	return loaded
}

func TestRender_DefaultRoundTrips(t *testing.T) {
	got := writeAndLoad(t, Default())
	if !reflect.DeepEqual(got, Default()) {
		t.Fatalf("Render(Default()) did not round-trip:\n got  %+v\n want %+v", got, Default())
	}
}

func TestRender_OverrideMatrixRoundTrips(t *testing.T) {
	providers := append(KnownProvidersForTest(), "none")
	keys := []string{"OKF", "PROJ", "RAC"}
	rootSets := [][]string{{"canon"}, {"rac", "decisions"}}

	for _, key := range keys {
		for _, roots := range rootSets {
			for _, prov := range providers {
				cfg := Config{
					RepositoryKey: key,
					CanonRoots:    roots,
					Ticketing:     Ticketing{Provider: prov},
				}
				got := writeAndLoad(t, cfg)
				if got.RepositoryKey != key {
					t.Errorf("key=%s roots=%v prov=%s: RepositoryKey=%q", key, roots, prov, got.RepositoryKey)
				}
				if !reflect.DeepEqual(got.CanonRoots, roots) {
					t.Errorf("key=%s roots=%v prov=%s: CanonRoots=%v", key, roots, prov, got.CanonRoots)
				}
				if got.Ticketing.Provider != prov {
					t.Errorf("key=%s roots=%v prov=%s: Provider=%q", key, roots, prov, got.Ticketing.Provider)
				}
			}
		}
	}
}

func TestRender_QuotesUnsafeScalarsSoTheyRoundTrip(t *testing.T) {
	// Values containing YAML metacharacters must still load back unchanged,
	// so init never produces a store that fails config.Load (Requirement 1).
	cfg := Config{
		RepositoryKey: "a: b",
		CanonRoots:    []string{"weird, root", "has#hash"},
		Ticketing:     Ticketing{Provider: "none"},
	}
	got := writeAndLoad(t, cfg)
	if got.RepositoryKey != "a: b" {
		t.Errorf("RepositoryKey=%q, want %q", got.RepositoryKey, "a: b")
	}
	if !reflect.DeepEqual(got.CanonRoots, cfg.CanonRoots) {
		t.Errorf("CanonRoots=%v, want %v", got.CanonRoots, cfg.CanonRoots)
	}
}

func TestRender_EmptyEnforcementLoadsEmpty(t *testing.T) {
	got := writeAndLoad(t, Default())
	if len(got.Enforcement.Blocking) != 0 || len(got.Enforcement.Advisory) != 0 || len(got.Enforcement.Disabled) != 0 {
		t.Fatalf("expected empty enforcement, got %+v", got.Enforcement)
	}
}

func TestRender_NonEmptyEnforcementRoundTrips(t *testing.T) {
	// Render emits an uncommented enforcement block when a policy is set; this
	// covers the writeStringList branch and confirms it round-trips.
	cfg := Default()
	cfg.Enforcement = Enforcement{
		Blocking: []string{"missing_required_section"},
		Advisory: []string{"ears_conformance"},
		Disabled: []string{"iso29148_singular"},
	}
	got := writeAndLoad(t, cfg)
	if !reflect.DeepEqual(got.Enforcement, cfg.Enforcement) {
		t.Errorf("Enforcement=%+v, want %+v", got.Enforcement, cfg.Enforcement)
	}
}

func TestRender_HasCommentForEachDocumentedKey(t *testing.T) {
	out := Render(Default())
	for _, marker := range []string{"repository_key", "canon_roots", "ticketing", "enforcement"} {
		// each documented key must have a preceding `#` comment line mentioning it
		if !hasCommentMentioning(out, marker) {
			t.Errorf("Render() output missing a comment describing %q\n---\n%s", marker, out)
		}
	}
}

func TestRender_CommentsAreInert(t *testing.T) {
	out := Render(Default())
	stripped := stripCommentLines(out)

	full := loadFromText(t, out)
	bare := loadFromText(t, stripped)
	if !reflect.DeepEqual(full, bare) {
		t.Fatalf("stripping comments changed the resolved config:\n full %+v\n bare %+v", full, bare)
	}
}

// --- helpers ---

func hasCommentMentioning(out, key string) bool {
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") && strings.Contains(strings.ToLower(trimmed), strings.ReplaceAll(key, "_", " ")) {
			return true
		}
	}
	// fall back: a comment that contains the raw key token
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, key) {
			return true
		}
	}
	return false
}

func stripCommentLines(out string) string {
	var b strings.Builder
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func loadFromText(t *testing.T, text string) Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".okf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(dir), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	return cfg
}

func TestDefault_SpecRoots(t *testing.T) {
	got := Default().SpecRoots
	want := []string{"specs", ".kiro/specs"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Default().SpecRoots=%v, want %v", got, want)
	}
}

func TestRender_SpecRootsRoundTrips(t *testing.T) {
	cfg := Default()
	cfg.SpecRoots = []string{"specs", ".kiro/specs", "docs/specs"}
	got := writeAndLoad(t, cfg)
	if !reflect.DeepEqual(got.SpecRoots, cfg.SpecRoots) {
		t.Errorf("SpecRoots=%v, want %v", got.SpecRoots, cfg.SpecRoots)
	}
}

func TestLoad_EmptySpecRootsBackfillsDefault(t *testing.T) {
	// A config file that omits spec_roots entirely must not silently disable spec
	// discovery: Load backfills the default roots, mirroring canon_roots behavior.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".okf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(dir), []byte("repository_key: OKF\ncanon_roots:\n  - canon\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.SpecRoots, DefaultSpecRoots()) {
		t.Errorf("SpecRoots=%v, want default %v", cfg.SpecRoots, DefaultSpecRoots())
	}
}

func TestRender_DefaultRendersBothSpecRoots(t *testing.T) {
	out := Render(Default())
	for _, want := range []string{"specs", ".kiro/specs"} {
		if !strings.Contains(out, want) {
			t.Errorf("Render(Default()) missing spec root %q\n---\n%s", want, out)
		}
	}
}

// KnownProvidersForTest is an illustrative set of provider strings used only to
// drive the render→load round-trip matrix. Render does not validate providers,
// so this list need not stay exactly in sync with validate.KnownProviders; it
// is duplicated here merely to keep the config package's tests free of a
// dependency on the validate package.
func KnownProvidersForTest() []string {
	return []string{"azure-devops", "github", "jira", "linear", "servicenow"}
}
