// Package indexer converts mathlib's generated tactic documentation to the
// small JSON catalog consumed by rugtac.
package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"

	"rugtac/internal/catalog"
)

const (
	// DefaultURL is doc-gen4's complete index of tactics available with mathlib.
	DefaultURL = "https://leanprover-community.github.io/mathlib4_docs/tactics.html"
	maxHTML    = 16 << 20
)

// Download fetches and parses a tactic index.
func Download(ctx context.Context, client *http.Client, sourceURL string) ([]catalog.Entry, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("User-Agent", "rugtac-index/1")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download tactic docs: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("download tactic docs: %s", response.Status)
	}

	return Parse(io.LimitReader(response.Body, maxHTML), sourceURL)
}

// Parse converts doc-gen4's tactics page into one entry per tactic name.
func Parse(r io.Reader, sourceURL string) ([]catalog.Entry, error) {
	document, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse tactic docs: %w", err)
	}

	byName := make(map[string]catalog.Entry)
	type family struct {
		entry    catalog.Entry
		variants []string
	}
	var families []family
	walk(document, func(node *html.Node) {
		if node.Type != html.ElementNode || node.Data != "div" ||
			!strings.HasPrefix(attribute(node, "id"), "userName-") {
			return
		}
		entry, variants, ok := parseEntry(node, sourceURL)
		if !ok {
			return
		}
		merge(byName, entry)
		families = append(families, family{entry: entry, variants: variants})
	})

	// Some tactic families document multiple registered names under one
	// heading. Add those variants only when the page has no canonical heading
	// for them, so canonical documentation always wins.
	for _, family := range families {
		for _, name := range family.variants {
			if _, exists := byName[name]; exists {
				continue
			}
			variant := family.entry
			variant.Name = name
			variant.Summary = "Documented variant of " + family.entry.Name + "."
			variant.Aliases = append(variant.Aliases, family.entry.Name)
			byName[name] = variant
		}
	}

	if len(byName) == 0 {
		return nil, fmt.Errorf("parse tactic docs: no tactic entries found")
	}
	entries := make([]catalog.Entry, 0, len(byName))
	for _, entry := range byName {
		entry.Aliases = append(entry.Aliases, semanticAliases(entry.Name)...)
		entry.Aliases = unique(entry.Aliases)
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

func semanticAliases(name string) []string {
	switch name {
	case "abel":
		return []string{"abelian"}
	case "abel_nf":
		return []string{"abelian", "abelian nf", "abelian normal form"}
	default:
		return nil
	}
}

func parseEntry(node *html.Node, sourceURL string) (catalog.Entry, []string, bool) {
	heading := findFirst(node, "h2")
	definitions := findDefinitions(node)
	if heading == nil || definitions == nil {
		return catalog.Entry{}, nil, false
	}

	name := normalizedText(heading)
	module := normalizedText(definitions)
	link := findFirst(definitions, "a")
	href := ""
	if link != nil {
		href = attribute(link, "href")
	}
	description := descriptionBeforeDefinitions(node)
	if description == "" {
		description = "This tactic has no documentation."
	}

	entry := catalog.Entry{
		Name:        name,
		Source:      classify(module, href),
		Summary:     summarize(description),
		Description: description,
		Usage:       firstCodeBlock(node),
		URL:         resolve(sourceURL, href),
		Module:      module,
		Library:     library(href),
	}
	return entry, documentedVariants(node, name), name != ""
}

func findDefinitions(node *html.Node) *html.Node {
	var found *html.Node
	walk(node, func(candidate *html.Node) {
		if found != nil || candidate.Type != html.ElementNode || candidate.Data != "dt" {
			return
		}
		if normalizedText(candidate) != "Defined in module:" {
			return
		}
		for sibling := candidate.NextSibling; sibling != nil; sibling = sibling.NextSibling {
			if sibling.Type == html.ElementNode && sibling.Data == "dd" {
				found = sibling
				return
			}
		}
	})
	return found
}

func descriptionBeforeDefinitions(node *html.Node) string {
	var parts []string
	afterHeading := false
	var visit func(*html.Node) bool
	visit = func(current *html.Node) bool {
		if current.Type == html.ElementNode {
			switch current.Data {
			case "h2":
				afterHeading = true
				return true
			case "dl":
				return false
			}
		}
		if afterHeading && current.Type == html.TextNode {
			parts = append(parts, current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			if !visit(child) {
				return false
			}
		}
		if afterHeading && current.Type == html.ElementNode && isBlock(current.Data) {
			parts = append(parts, "\n")
		}
		return true
	}
	visit(node)
	return normalize(strings.Join(parts, ""))
}

func firstCodeBlock(node *html.Node) string {
	pre := findFirst(node, "pre")
	if pre == nil {
		return ""
	}
	return strings.TrimSpace(text(pre))
}

func documentedVariants(node *html.Node, name string) []string {
	if name == "" || !isIdentifier(name) {
		return nil
	}
	var variants []string
	walk(node, func(candidate *html.Node) {
		if candidate.Type != html.ElementNode || candidate.Data != "code" {
			return
		}
		fields := strings.Fields(normalizedText(candidate))
		if len(fields) == 0 {
			return
		}
		value := strings.Trim(fields[0], "`(),;:")
		if isVariant(name, value) {
			variants = append(variants, value)
		}
	})
	return unique(variants)
}

func isVariant(name, candidate string) bool {
	if candidate == name || !isIdentifier(candidate) || !strings.HasPrefix(candidate, name) {
		return false
	}
	suffix := strings.TrimPrefix(candidate, name)
	first, _ := utf8.DecodeRuneInString(suffix)
	return first == '_' || first == '?' || first == '!' || first == '\'' || unicode.IsDigit(first)
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '\'' || r == '?' || r == '!' || (i == 0 && r == '#') {
			continue
		}
		return false
	}
	return true
}

func classify(module, href string) string {
	if strings.HasPrefix(module, "Lean.") || strings.HasPrefix(module, "Init.") ||
		strings.HasPrefix(module, "Std.") || strings.HasPrefix(href, "./Lean/") ||
		strings.HasPrefix(href, "./Init/") || strings.HasPrefix(href, "./Std/") {
		return "Lean"
	}
	return "mathlib"
}

func library(href string) string {
	path := strings.TrimPrefix(href, "./")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "other"
	}
	switch parts[0] {
	case "Init", "Lean", "Std", "Batteries":
		return "core"
	case "Mathlib":
		if len(parts) > 1 {
			return kebab(parts[1])
		}
		return "mathlib"
	default:
		return kebab(parts[0])
	}
}

func kebab(value string) string {
	var b strings.Builder
	for i, r := range value {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('-')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

func merge(entries map[string]catalog.Entry, incoming catalog.Entry) {
	existing, found := entries[incoming.Name]
	if !found {
		entries[incoming.Name] = incoming
		return
	}
	existing.Source = mergeValue(existing.Source, incoming.Source, "/")
	existing.Module = mergeValue(existing.Module, incoming.Module, "; ")
	if incoming.Description != existing.Description &&
		incoming.Description != "This tactic has no documentation." {
		if existing.Description == "This tactic has no documentation." {
			existing.Description = incoming.Description
		} else {
			existing.Description += "\n\n" + incoming.Description
		}
		existing.Summary = summarize(existing.Description)
	}
	if existing.Usage == "" {
		existing.Usage = incoming.Usage
	}
	if existing.URL == "" {
		existing.URL = incoming.URL
	}
	existing.Aliases = append(existing.Aliases, incoming.Aliases...)
	entries[incoming.Name] = existing
}

func mergeValue(left, right, separator string) string {
	if left == "" {
		return right
	}
	if right == "" || left == right || strings.Contains(left, right) {
		return left
	}
	return left + separator + right
}

func summarize(description string) string {
	line := description
	if index := strings.IndexByte(line, '\n'); index >= 0 {
		line = line[:index]
	}
	runes := []rune(line)
	if len(runes) <= 160 {
		return line
	}
	return strings.TrimSpace(string(runes[:157])) + "..."
}

func resolve(base, reference string) string {
	if reference == "" {
		return base
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return reference
	}
	referenceURL, err := url.Parse(reference)
	if err != nil {
		return reference
	}
	return baseURL.ResolveReference(referenceURL).String()
}

func findFirst(node *html.Node, element string) *html.Node {
	var found *html.Node
	walk(node, func(candidate *html.Node) {
		if found == nil && candidate.Type == html.ElementNode && candidate.Data == element {
			found = candidate
		}
	})
	return found
}

func walk(node *html.Node, visit func(*html.Node)) {
	visit(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walk(child, visit)
	}
}

func attribute(node *html.Node, key string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}
	return ""
}

func normalizedText(node *html.Node) string {
	return strings.Join(strings.Fields(text(node)), " ")
}

func text(node *html.Node) string {
	var b strings.Builder
	walk(node, func(candidate *html.Node) {
		if candidate.Type == html.TextNode {
			b.WriteString(candidate.Data)
		}
	})
	return b.String()
}

func normalize(value string) string {
	lines := strings.Split(value, "\n")
	clean := lines[:0]
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			clean = append(clean, line)
		}
	}
	return strings.Join(clean, "\n")
}

func isBlock(element string) bool {
	switch element {
	case "p", "pre", "li", "ul", "ol", "blockquote", "div", "br":
		return true
	default:
		return false
	}
}

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

// WriteJSON atomically replaces path with a formatted local index.
func WriteJSON(path string, entries []catalog.Entry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".tactics-*.json")
	if err != nil {
		return fmt.Errorf("create temporary index: %w", err)
	}
	temporaryPath := file.Name()
	defer os.Remove(temporaryPath)

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(entries); err != nil {
		file.Close()
		return fmt.Errorf("encode index: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close index: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace index: %w", err)
	}
	return nil
}
