// Package changelog handles reading and writing changelog.txt files for OKF bundles.
package changelog

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const ChangelogFile = "changelog.txt"

// ChangelogEntry represents a single changelog entry.
type ChangelogEntry struct {
	Timestamp time.Time
	Message   string
}

// Changelog represents the changelog file contents.
type Changelog struct {
	Source             string           // First line: URL or file path
	SummarizeMode      string           // Optional: extractive | llm
	SummarizeAlgorithm string           // Optional: lsa | lexrank | textrank | ...
	Language           string           // Optional: language for extractive summarization
	Entries            []ChangelogEntry // Subsequent timestamped entries
}

// ReadChangelog reads and parses a changelog.txt file from the bundle.
func ReadChangelog(bundlePath string) (*Changelog, error) {
	changelogPath := filepath.Join(bundlePath, ChangelogFile)

	file, err := os.Open(changelogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("changelog.txt not found in bundle")
		}
		return nil, fmt.Errorf("failed to open changelog: %w", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	cl := &Changelog{}

	// First line is the source
	if scanner.Scan() {
		cl.Source = strings.TrimSpace(scanner.Text())
	} else {
		return nil, fmt.Errorf("changelog.txt is empty")
	}

	if cl.Source == "" {
		return nil, fmt.Errorf("changelog.txt has empty source line")
	}

	// Subsequent lines may be metadata headers (Key: Value) or entries.
	// Metadata lines must appear before entries; once we see an entry, the
	// remainder of the file is treated as entries.
	inHeaders := true
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if inHeaders {
			if key, value, ok := parseMetadataLine(line); ok {
				switch key {
				case "Summarize-Mode":
					cl.SummarizeMode = value
				case "Summarize-Algorithm":
					cl.SummarizeAlgorithm = value
				case "Language":
					cl.Language = value
				}
				continue
			}
			// First non-metadata line ends the header block.
			inHeaders = false
		}

		entry, err := parseEntry(line)
		if err != nil {
			// Skip malformed entries but continue parsing
			continue
		}
		cl.Entries = append(cl.Entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading changelog: %w", err)
	}

	return cl, nil
}

// parseMetadataLine parses a header line of the form "Key: Value". Returns
// (key, value, true) for a recognized format, or ("", "", false) if the line
// is not a metadata header (e.g., it's a timestamped entry).
func parseMetadataLine(line string) (string, string, bool) {
	// Reject lines that look like timestamped entries (contain " - " and
	// start with an RFC3339-ish date).
	if strings.Contains(line, " - ") {
		first := strings.SplitN(line, " - ", 2)[0]
		if _, err := time.Parse(time.RFC3339, strings.TrimSpace(first)); err == nil {
			return "", "", false
		}
	}
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	// Keys are simple identifiers (no spaces).
	for _, r := range key {
		if r == ' ' || r == '\t' {
			return "", "", false
		}
	}
	return key, value, true
}

// parseEntry parses a changelog entry line.
// Format: "2024-01-15T10:30:00Z - Message here"
func parseEntry(line string) (ChangelogEntry, error) {
	parts := strings.SplitN(line, " - ", 2)
	if len(parts) != 2 {
		return ChangelogEntry{}, fmt.Errorf("invalid entry format")
	}

	timestamp, err := time.Parse(time.RFC3339, strings.TrimSpace(parts[0]))
	if err != nil {
		return ChangelogEntry{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	return ChangelogEntry{
		Timestamp: timestamp,
		Message:   strings.TrimSpace(parts[1]),
	}, nil
}

// WriteChangelog writes a changelog.txt file to the bundle.
func WriteChangelog(bundlePath string, cl *Changelog) error {
	changelogPath := filepath.Join(bundlePath, ChangelogFile)

	var lines []string
	lines = append(lines, cl.Source)

	if cl.SummarizeMode != "" {
		lines = append(lines, fmt.Sprintf("Summarize-Mode: %s", cl.SummarizeMode))
	}
	if cl.SummarizeAlgorithm != "" {
		lines = append(lines, fmt.Sprintf("Summarize-Algorithm: %s", cl.SummarizeAlgorithm))
	}
	if cl.Language != "" {
		lines = append(lines, fmt.Sprintf("Language: %s", cl.Language))
	}

	for _, entry := range cl.Entries {
		lines = append(lines, fmt.Sprintf("%s - %s", entry.Timestamp.Format(time.RFC3339), entry.Message))
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(changelogPath, []byte(content), 0644)
}

// CreateChangelog creates a new changelog with an initial entry.
func CreateChangelog(bundlePath, source string, conceptCount int) error {
	return CreateChangelogWithMetadata(bundlePath, source, conceptCount, "", "", "")
}

// CreateChangelogWithMetadata creates a new changelog with summarization
// preferences recorded for future updates.
func CreateChangelogWithMetadata(bundlePath, source string, conceptCount int, summarizeMode, summarizeAlgorithm, language string) error {
	cl := &Changelog{
		Source:             source,
		SummarizeMode:      summarizeMode,
		SummarizeAlgorithm: summarizeAlgorithm,
		Language:           language,
		Entries: []ChangelogEntry{
			{
				Timestamp: time.Now().UTC(),
				Message:   fmt.Sprintf("Initial bundle created with %d concepts", conceptCount),
			},
		},
	}
	return WriteChangelog(bundlePath, cl)
}

// AppendEntry adds a new entry to an existing changelog.
func AppendEntry(bundlePath string, message string) error {
	cl, err := ReadChangelog(bundlePath)
	if err != nil {
		return err
	}

	cl.Entries = append(cl.Entries, ChangelogEntry{
		Timestamp: time.Now().UTC(),
		Message:   message,
	})

	return WriteChangelog(bundlePath, cl)
}

// GetSource reads just the source from a changelog file.
func GetSource(bundlePath string) (string, error) {
	cl, err := ReadChangelog(bundlePath)
	if err != nil {
		return "", err
	}
	return cl.Source, nil
}
