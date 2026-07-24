// Package catalog loads the local collection of searchable documentation.
package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const projectIndexPath = "data/tactics.json"

// Entry is one searchable piece of documentation.
//
// The JSON representation is intentionally straightforward so users can keep
// their own index in source control and load it with rugtac -data FILE.
type Entry struct {
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	Usage       string   `json:"usage,omitempty"`
	URL         string   `json:"url,omitempty"`
	Module      string   `json:"module,omitempty"`
	Library     string   `json:"library,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
}

// DefaultPath finds the development index or an index installed alongside the
// binary under PREFIX/share/rugtac.
func DefaultPath() string {
	if regularFile(projectIndexPath) {
		return projectIndexPath
	}
	if executable, err := os.Executable(); err == nil {
		installed := filepath.Clean(filepath.Join(
			filepath.Dir(executable), "..", "share", "rugtac", "tactics.json",
		))
		if regularFile(installed) {
			return installed
		}
	}
	return projectIndexPath
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// Load reads entries from a local JSON file.
func Load(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read index %q: %w", path, err)
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode index: %w", err)
	}
	if err := validate(entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func validate(entries []Entry) error {
	if len(entries) == 0 {
		return fmt.Errorf("index contains no entries")
	}

	seen := make(map[string]struct{}, len(entries))
	for i, entry := range entries {
		if strings.TrimSpace(entry.Name) == "" {
			return fmt.Errorf("index entry %d has no name", i+1)
		}
		key := strings.ToLower(entry.Source + "\x00" + entry.Name)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("index contains duplicate %q in source %q", entry.Name, entry.Source)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// Libraries returns the sorted non-core library names present in entries.
func Libraries(entries []Entry) []string {
	seen := make(map[string]struct{})
	for _, entry := range entries {
		if entry.Library != "" && entry.Library != "core" {
			seen[entry.Library] = struct{}{}
		}
	}
	libraries := make([]string, 0, len(seen))
	for library := range seen {
		libraries = append(libraries, library)
	}
	sort.Strings(libraries)
	return libraries
}

// Filter returns entries in scope. Scope may be "all", "core", or a library
// name such as "data" or "tactic".
func Filter(entries []Entry, scope string) []Entry {
	if scope == "" || strings.EqualFold(scope, "all") {
		return entries
	}
	filtered := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if strings.EqualFold(entry.Library, scope) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
