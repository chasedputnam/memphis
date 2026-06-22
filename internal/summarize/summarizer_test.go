package summarize

import (
	"os"
	"strings"
	"testing"
)

// osWriteFile aliases os.WriteFile for the test helper.
var osWriteFile = os.WriteFile

func TestNewSummarizer_DefaultExtractive(t *testing.T) {
	s, err := NewSummarizer(Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name() != DefaultAlgorithm {
		t.Errorf("expected name %q, got %q", DefaultAlgorithm, s.Name())
	}
}

func TestNewSummarizer_UnknownMode(t *testing.T) {
	_, err := NewSummarizer(Config{Mode: "bogus"})
	if err == nil {
		t.Error("expected error for unknown mode")
	}
}

func TestNewSummarizer_ExplicitAlgorithm(t *testing.T) {
	s, err := NewSummarizer(Config{Mode: ModeExtractive, Algorithm: "luhn"})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if s.Name() != "luhn" {
		t.Errorf("expected name luhn, got %q", s.Name())
	}
}

func TestExtractive_Summarize(t *testing.T) {
	s, err := NewExtractive("lsa", "english")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	sum, err := s.Summarize("Penguins are flightless birds. Penguins live in cold places. Penguins eat fish.", "Penguins")
	if err != nil {
		t.Fatalf("Summarize error: %v", err)
	}
	if sum.Source != "lsa" {
		t.Errorf("expected source lsa, got %q", sum.Source)
	}
	if sum.Text == "" {
		t.Error("expected non-empty summary")
	}
	if len(sum.Text) > MaxSummaryLength+3 {
		t.Errorf("summary too long: %d chars", len(sum.Text))
	}
}

func TestExtractive_EmptyContent(t *testing.T) {
	s, _ := NewExtractive("lsa", "english")
	sum, err := s.Summarize("", "")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if sum.Source != SourceNone {
		t.Errorf("expected SourceNone, got %q", sum.Source)
	}
	if sum.Text != "" {
		t.Errorf("expected empty text, got %q", sum.Text)
	}
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		in   string
		want Mode
		err  bool
	}{
		{"", DefaultMode, false},
		{"extractive", ModeExtractive, false},
		{"llm", ModeLLM, false},
		{"LLM", ModeLLM, false},
		{"foo", "", true},
	}
	for _, tc := range tests {
		got, err := ParseMode(tc.in)
		if tc.err {
			if err == nil {
				t.Errorf("ParseMode(%q): expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseMode(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseMode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNewExtractiveWithConfig_EdmundsonAppliesBonusWords(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	cfgPath := tmp + "/edmundson.yaml"
	body := "bonus_words:\n  - wonderful\n"
	if err := writeFile(cfgPath, body); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	t.Setenv("HOME", t.TempDir())

	adapter, err := NewExtractiveWithConfig(Config{
		Mode:                ModeExtractive,
		Algorithm:           "edmundson",
		Language:            "english",
		EdmundsonConfigPath: cfgPath,
	})
	if err != nil {
		t.Fatalf("NewExtractiveWithConfig: %v", err)
	}
	if adapter.edmundsonConfig == nil {
		t.Fatal("expected edmundsonConfig to be populated")
	}
	if len(adapter.edmundsonConfig.Bonus) == 0 || adapter.edmundsonConfig.Bonus[0] != "wonderful" {
		t.Errorf("Bonus = %v", adapter.edmundsonConfig.Bonus)
	}

	sum, err := adapter.Summarize("This is the first sentence about cats. The second sentence is wonderful and explains everything.", "Topic")
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if sum.Source != "edmundson" {
		t.Errorf("Source = %q", sum.Source)
	}
	if sum.Text == "" {
		t.Error("expected non-empty summary")
	}
}

func writeFile(path, content string) error {
	return osWriteFile(path, []byte(content), 0644)
}

func TestExtractive_TruncatesAt200(t *testing.T) {
	long := strings.Repeat("important data ", 50) + "."
	s, _ := NewExtractive("lsa", "english")
	sum, err := s.Summarize(long+" Second short sentence.", "Title")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(sum.Text) > MaxSummaryLength+3 {
		t.Errorf("summary too long: %d chars", len(sum.Text))
	}
}
