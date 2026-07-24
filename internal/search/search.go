// Package search ranks documentation entries against a human query.
package search

import (
	"sort"
	"strings"
	"unicode"

	"github.com/sahilm/fuzzy"

	"rugtac/internal/catalog"
)

// Result couples an entry with its fuzzy score. Higher scores are better.
type Result struct {
	Entry catalog.Entry
	Score int
}

// Find returns entries that match every word in query. Each word is matched
// independently, allowing a query such as "abelian nf" to span an entry's
// description and name. Empty queries return the full catalog by name.
func Find(entries []catalog.Entry, query string) []Result {
	terms := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return unicode.IsSpace(r)
	})
	if len(terms) == 0 {
		results := make([]Result, len(entries))
		for i, entry := range entries {
			results[i] = Result{Entry: entry}
		}
		sort.SliceStable(results, func(i, j int) bool {
			return strings.ToLower(results[i].Entry.Name) < strings.ToLower(results[j].Entry.Name)
		})
		return results
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	results := make([]Result, 0, len(entries))
	for _, entry := range entries {
		score, matches := scoreEntry(entry, terms)
		if !matches {
			continue
		}
		name := strings.ToLower(entry.Name)
		if name == queryLower {
			score += 10_000
		} else if strings.HasPrefix(name, queryLower) {
			score += 1_000
		}
		results = append(results, Result{Entry: entry, Score: score})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return strings.ToLower(results[i].Entry.Name) < strings.ToLower(results[j].Entry.Name)
	})
	return results
}

// FindStrings fuzzy-ranks a small list of labels. Empty queries return a
// sorted copy, which is useful for secondary pickers such as library selection.
func FindStrings(values []string, query string) []string {
	if strings.TrimSpace(query) == "" {
		results := append([]string(nil), values...)
		sort.Strings(results)
		return results
	}
	matches := fuzzy.Find(strings.ToLower(query), values)
	results := make([]string, len(matches))
	for i, match := range matches {
		results[i] = match.Str
	}
	return results
}

func scoreEntry(entry catalog.Entry, terms []string) (int, bool) {
	fields := []string{entry.Name}
	fields = append(fields, strings.FieldsFunc(entry.Name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})...)
	for _, alias := range entry.Aliases {
		fields = append(fields, alias)
		fields = append(fields, strings.FieldsFunc(alias, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})...)
	}
	fields = append(fields, strings.FieldsFunc(
		entry.Source+" "+entry.Library+" "+entry.Module+" "+entry.Summary+" "+entry.Description,
		func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '_' },
	)...)

	total := 0
	for _, term := range terms {
		matches := fuzzy.Find(term, fields)
		if len(matches) == 0 {
			return 0, false
		}
		// fuzzy.Find sorts by match quality, so the best field is first.
		total += matches[0].Score
		for _, field := range fields {
			if strings.EqualFold(field, term) {
				total += 10_000
				break
			}
		}
	}
	return total, true
}
