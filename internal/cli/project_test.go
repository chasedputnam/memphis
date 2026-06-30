package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func newProjectCmd() *cobra.Command {
	c := &cobra.Command{Use: "project <spec-doc-or-dir>", Args: cobra.ExactArgs(1), RunE: runProject,
		SilenceUsage: true, SilenceErrors: true}
	c.Flags().String("store", ".", "")
	c.Flags().String("type", "", "")
	c.Flags().Bool("dry-run", false, "")
	c.Flags().Bool("write", false, "")
	c.Flags().Bool("force", false, "")
	c.Flags().String("kiro-agent", "", "")
	c.Flags().Bool("json", false, "")
	c.Flags().Bool("quiet", false, "")
	return c
}

func runProjectT(t *testing.T, args []string, flags map[string]string) (string, error) {
	t.Helper()
	c := newProjectCmd()
	for k, v := range flags {
		if err := c.Flags().Set(k, v); err != nil {
			t.Fatalf("set flag %s: %v", k, err)
		}
	}
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := runProject(c, args)
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String(), err
}

func writeReqSpec(t *testing.T, store, feature, body string) string {
	t.Helper()
	p := filepath.Join(store, "specs", feature, "requirements.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProjectCLI_HappyPath(t *testing.T) {
	store := t.TempDir()
	src := writeReqSpec(t, store, "feat", "# Feat\n\n## Problem\n\nP.\n\n## Requirements\n\n[REQ-001] The system SHALL do a thing.\n")
	_, err := runProjectT(t, []string{src}, map[string]string{"store": store})
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(store, "canon", "feat", "requirements.md")); statErr != nil {
		t.Errorf("expected artifact written: %v", statErr)
	}
}

func TestProjectCLI_DirSkipsTasks(t *testing.T) {
	store := t.TempDir()
	dir := filepath.Join(store, "specs", "feat")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "requirements.md"), []byte("# F\n\n## Problem\n\nP.\n\n## Requirements\n\n[REQ-001] The system SHALL x.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "design.md"), []byte("# F\n\n## Context\n\nc.\n\n## User Need\n\nn.\n\n## Design\n\nd.\n\n## Constraints\n\nc.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("# Tasks\n\n- [ ] 1. do\n"), 0o644)

	_, err := runProjectT(t, []string{dir}, map[string]string{"store": store})
	if err != nil {
		t.Fatalf("project dir: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(store, "canon", "feat", "tasks.md")); !os.IsNotExist(statErr) {
		t.Error("tasks.md must not be projected")
	}
	for _, want := range []string{"requirements.md", "design.md"} {
		if _, statErr := os.Stat(filepath.Join(store, "canon", "feat", want)); statErr != nil {
			t.Errorf("expected %s projected: %v", want, statErr)
		}
	}
}

func TestProjectCLI_DryRunWritesNothing(t *testing.T) {
	store := t.TempDir()
	src := writeReqSpec(t, store, "feat", "# Feat\n\n## Problem\n\nP.\n\n## Requirements\n\n[REQ-001] The system SHALL do a thing.\n")
	_, err := runProjectT(t, []string{src}, map[string]string{"store": store, "dry-run": "true"})
	if err != nil {
		t.Fatalf("dry-run project: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(store, "canon", "feat", "requirements.md")); !os.IsNotExist(statErr) {
		t.Error("dry-run must not write an artifact")
	}
}

func TestProjectCLI_ChangeWithoutWriteExitsNonZero(t *testing.T) {
	store := t.TempDir()
	src := writeReqSpec(t, store, "feat", "# Feat\n\n## Problem\n\nfirst.\n\n## Requirements\n\n[REQ-001] The system SHALL do the first thing.\n")
	if _, err := runProjectT(t, []string{src}, map[string]string{"store": store}); err != nil {
		t.Fatalf("first project: %v", err)
	}
	target := filepath.Join(store, "canon", "feat", "requirements.md")
	before, _ := os.ReadFile(target)

	// Modify source, re-project without --write (non-interactive).
	_ = os.WriteFile(src, []byte("# Feat\n\n## Problem\n\nSECOND.\n\n## Requirements\n\n[REQ-001] The system SHALL do the second thing.\n"), 0o644)
	_, err := runProjectT(t, []string{src}, map[string]string{"store": store})
	if err == nil {
		t.Error("expected non-zero exit (error) when an existing artifact changes without --write")
	}
	after, _ := os.ReadFile(target)
	if string(before) != string(after) {
		t.Error("artifact must be unchanged without --write")
	}
}

func TestProjectCLI_BlockingExitsNonZero(t *testing.T) {
	store := t.TempDir()
	src := writeReqSpec(t, store, "feat", "# Feat\n\n## Problem\n\nP.\n\n## Requirements\n\n[REQ-001] the system shall use a lowercase keyword.\n")
	_, err := runProjectT(t, []string{src}, map[string]string{"store": store})
	if err == nil {
		t.Error("expected non-zero exit when the projected artifact has blocking issues")
	}
}

func TestProjectCLI_JSON(t *testing.T) {
	store := t.TempDir()
	src := writeReqSpec(t, store, "feat", "# Feat\n\n## Problem\n\nP.\n\n## Requirements\n\n[REQ-001] The system SHALL do a thing.\n")
	out, err := runProjectT(t, []string{src}, map[string]string{"store": store, "json": "true"})
	if err != nil {
		t.Fatalf("project --json: %v", err)
	}
	var results []map[string]any
	if jerr := json.Unmarshal([]byte(out), &results); jerr != nil {
		t.Fatalf("output is not a JSON array: %v\n%s", jerr, out)
	}
	if len(results) != 1 || results[0]["Type"] != "requirement" {
		t.Errorf("unexpected JSON result: %v", results)
	}
}
